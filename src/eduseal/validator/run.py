import logging
from io import BytesIO
import base64
import time
import os
import signal
import threading
from typing import Optional
import itertools
from concurrent import futures
import grpc

from pyhanko.sign.validation import validate_pdf_signature
from pyhanko_certvalidator import ValidationContext
from pyhanko.keys import load_cert_from_pemder
from pyhanko.pdf_utils.reader import PdfFileReader

from eduseal.validator.v1_validator_pb2 import ValidateReply, ValidateRequest
import eduseal.validator.v1_validator_pb2_grpc as pb2_grpc
from eduseal.validator.config import parse, CFG
from eduseal.cert_reloader import CertReloader

class Common():
    def __init__(self):
        self.service_name = os.getenv("EDUSEAL_SERVICE_NAME", "eduseal_validator")
        self.logger = logging.getLogger(self.service_name)
        self.logger.setLevel(logging.DEBUG)
        
        # Clear any existing handlers to avoid duplicates
        self.logger.handlers.clear()
        
        # Console handler
        ch = logging.StreamHandler()
        ch.setLevel(logging.DEBUG)
        formatter = logging.Formatter("%(asctime)s - %(name)s - %(levelname)s - %(message)s")
        ch.setFormatter(formatter)
        self.logger.addHandler(ch)
        
        # File handler
        fh = logging.FileHandler("/var/log/sunet/validator.log")
        fh.setLevel(logging.DEBUG)
        fh.setFormatter(formatter)
        self.logger.addHandler(fh)
        
        self.logger.propagate = False

        self.config: CFG = parse(log=self.logger)

class Validator(Common, pb2_grpc.ValidatorServicer):
    def __init__(self):
        Common.__init__(self)

        self._active_validations = 0
        self._active_lock = threading.Lock()

        trust_roots = self._load_trust_roots(self.config.validation_certificates_path)
        if not trust_roots:
            self.logger.error(f"No certificates found in {self.config.validation_certificates_path}")
            raise RuntimeError(f"No certificates found in {self.config.validation_certificates_path}")

        self.validation_context = ValidationContext(
            trust_roots=trust_roots,
        )

    @property
    def active_validations(self) -> int:
        with self._active_lock:
            return self._active_validations

    def _load_trust_roots(self, certs_path: str) -> list:
        """Load all .crt, .pem, and .der certificates from the configured directory."""
        trust_roots = []
        for entry in sorted(os.listdir(certs_path)):
            if not entry.lower().endswith((".crt", ".pem", ".der")):
                continue
            full_path = os.path.join(certs_path, entry)
            if not os.path.isfile(full_path):
                continue
            try:
                cert = load_cert_from_pemder(full_path)
                trust_roots.append(cert)
                self.logger.info(f"Loaded trust root: {entry}")
            except Exception as e:
                self.logger.warning(f"Skipping {entry}: {e}")
        return trust_roots

    def Validate(self, in_data: ValidateRequest, context) -> ValidateReply:
        with self._active_lock:
            self._active_validations += 1
        try:
            return self._do_validate(in_data, context)
        finally:
            with self._active_lock:
                self._active_validations -= 1

    def _do_validate(self, in_data: ValidateRequest, context) -> ValidateReply:
        try:
            pdf_data = base64.b64decode(in_data.data.encode("utf-8"), validate=False)
        except Exception as e:
            self.logger.error(f"Error decoding base64: {e}")
            return ValidateReply(
                validation_backend=self.service_name,
                error=f"Error decoding base64: {e}",
            )
        try:
            pdf = PdfFileReader(BytesIO(pdf_data), strict=False)
        except Exception as e:
            self.logger.error(f"Error reading PDF: {e}")
            return ValidateReply(
                validation_backend=self.service_name,
                error=f"Error reading PDF: {e}",
            )

        if len(pdf.embedded_signatures) == 0:
            self.logger.error("No signature found")
            return ValidateReply(
                validation_backend=self.service_name,
                error="No signature found",
            )

        try:
            status = validate_pdf_signature(
                embedded_sig=pdf.embedded_signatures[0],
                signer_validation_context=self.validation_context,
            )
        except Exception as e:
            self.logger.error(f"Validation error: {e}")
            return ValidateReply(
                validation_backend=self.service_name,
                error=f"Validation error {e}",
            )

        try:
            transaction_id = self.get_transaction_id_from_keywords(pdf=pdf)
        except Exception as e:
            self.logger.error(f"Error getting transaction_id: {e}")
            return ValidateReply(
                validation_backend=self.service_name,
                error=f"Error getting transaction_id {e}",
            )

        self.logger.info(f"Validate a signed base64 PDF, transaction_id:{transaction_id}")

        return ValidateReply(
            validation_backend=self.service_name,
            intact_signature=status.intact,
            valid_signature=status.valid,
            transaction_id=transaction_id,
            error="",
        )

    def get_transaction_id_from_keywords(self,pdf: PdfFileReader) -> Optional[str]:
        """simple function to get transaction_id from a list of keywords"""
        for keyword in pdf.document_meta_view.keywords:
            entry = keyword.split(sep=":")
            if entry[0] == "transaction_id":
                self.logger.info(msg=f"found transaction_id: {entry[1]}")
                return entry[1]
        return None

class GRPCServer(Common):
    GRACEFUL_SHUTDOWN_TIMEOUT = 30  # seconds to wait for in-progress RPCs

    def __init__(self) -> None:
        super().__init__()
        self._shutdown_event = threading.Event()
        self._cert_reloader = None

    def _remove_healthcheck(self):
        """Remove healthcheck file so load balancer stops sending new requests."""
        try:
            os.remove('/tmp/healthcheck')
            self.logger.info("Healthcheck file removed")
        except FileNotFoundError:
            pass

    def _signal_handler(self, signum, frame):
        sig_name = signal.Signals(signum).name
        self.logger.info(f"Received {sig_name}, starting graceful shutdown")
        self._shutdown_event.set()

    def start(self):
        server = grpc.server(futures.ThreadPoolExecutor(max_workers=10))
        validator_servicer = Validator()
        pb2_grpc.add_ValidatorServicer_to_server(validator_servicer, server)

        if self.config.grpc_server.tls_enabled:
            assert self.config.grpc_server.private_key_path is not None
            assert self.config.grpc_server.certificate_chain_path is not None

            def _fetch_cert_config():
                with open(self.config.grpc_server.private_key_path, 'rb') as f:
                    private_key = f.read()
                with open(self.config.grpc_server.certificate_chain_path, 'rb') as f:
                    certificate_chain = f.read()
                return grpc.ssl_server_certificate_configuration(
                    [(private_key, certificate_chain)]
                )

            initial_config = _fetch_cert_config()
            server_credentials = grpc.dynamic_ssl_server_credentials(
                initial_config,
                _fetch_cert_config,
            )

            # Log when cert files change on disk
            self._cert_reloader = CertReloader(
                self.config.grpc_server.certificate_chain_path,
                self.config.grpc_server.private_key_path,
                lambda: self.logger.info("gRPC cert files changed; new connections will use updated certificate"),
                self.logger,
            )

            server.add_secure_port(self.config.grpc_server.addr, server_credentials)

        # Register signal handlers for graceful shutdown
        signal.signal(signal.SIGTERM, self._signal_handler)
        signal.signal(signal.SIGINT, self._signal_handler)

        server.start()
        time.sleep(2)
        open('/tmp/healthcheck', 'w').close()
        self.logger.info("Server started and ready to accept requests")

        # Wait until a shutdown signal is received
        self._shutdown_event.wait()

        # Step 1: Remove healthcheck so no new traffic arrives
        self._remove_healthcheck()

        # Step 2: Gracefully stop – stops accepting new RPCs and waits
        # for in-progress ones to complete (up to the timeout).
        active = validator_servicer.active_validations
        self.logger.info(
            f"Graceful shutdown: waiting up to {self.GRACEFUL_SHUTDOWN_TIMEOUT}s "
            f"for {active} in-progress validation(s) to complete"
        )
        shutdown_event = server.stop(grace=self.GRACEFUL_SHUTDOWN_TIMEOUT)
        shutdown_event.wait()

        if self._cert_reloader is not None:
            self._cert_reloader.stop()

        self.logger.info("Server stopped gracefully")


if __name__ == "__main__":
    validator = GRPCServer()
    validator.start()