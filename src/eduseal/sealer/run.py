import grpc
import time
from concurrent import futures
import logging
from io import BytesIO
import os
import base64

from pkcs11 import Session, UserAlreadyLoggedIn
from pyhanko.sign.pkcs11 import open_pkcs11_session
from pyhanko.sign import signers
from pyhanko.sign.fields import SigSeedSubFilter
from pyhanko.pdf_utils.incremental_writer import IncrementalPdfFileWriter
from pyhanko.sign.pkcs11 import PKCS11Signer
from pyhanko.sign.signers.pdf_signer import PdfSigner
from pyhanko.pdf_utils.crypt.api import PdfKeyNotAvailableError
from pyhanko.pdf_utils.misc import PdfReadError

from eduseal.sealer.v1_sealer_pb2 import SealRequest, SealReply
import eduseal.sealer.v1_sealer_pb2_grpc as pb2_grpc
from eduseal.sealer.config import parse, CFG

class Common():
    def __init__(self) -> None:
        self.service_name = os.getenv("EDUSEAL_SERVICE_NAME", "eduseal_sealer")
        self.logger = logging.getLogger(self.service_name)
        self.logger.setLevel(logging.DEBUG)
        
        ch = logging.StreamHandler()
        ch.setLevel(logging.DEBUG)

        ch.setFormatter(logging.Formatter("%(asctime)s - %(name)s - %(levelname)s - %(message)s"))

        self.logger.addHandler(ch)
        self.logger.propagate = False

        self.logger.info(f"init sealer {self.service_name}")


        self.config: CFG = parse(log=self.logger)


class Sealer(Common, pb2_grpc.SealerServicer):
    def __init__(self):
        Common.__init__(self)
        self.pkc11_session: Session
        self.init_pkcs11_session()

    def init_pkcs11_session(self) -> None:
        self.logger.info("init pkcs11 session")
        self.logger.debug(f"pkcs11 module: {self.config.pkcs11.module}")
        self.logger.debug(f"pkcs11 slot: {self.config.pkcs11.slot}")
        self.logger.debug(f"pkcs11 label: {self.config.pkcs11.label}")
        try:
            self.pkc11_session = open_pkcs11_session(
                lib_location=self.config.pkcs11.module, 
                slot_no=self.config.pkcs11.slot, 
                token_label=self.config.pkcs11.label,
                user_pin=self.config.pkcs11.pin,
            )
        except UserAlreadyLoggedIn:
            self.logger.info("pkcs11 user already logged in!")

    def Seal(self, in_data: SealRequest, context)-> SealReply:
        self.logger.debug("start sealing")
        self.logger.debug(f"transaction_id: {in_data.transaction_id}")

        try:
            pdf_writer = IncrementalPdfFileWriter(input_stream=BytesIO(base64.urlsafe_b64decode(in_data.data)), strict=False)
        except PdfReadError as _e:
            self.logger.debug(f"input pdf is not valid, err: {_e}")
            return SealReply(
                transaction_id=in_data.transaction_id,
                data="",
                error=f"input pdf is not valid, err: {_e}",
                sealer_backend=self.service_name,
            )

        pdf_writer.document_meta.keywords = [f"transaction_id:{in_data.transaction_id}"]
        self.logger.debug("add meta data to pdf")

        try:
            pkcs11_signer = PKCS11Signer(
                pkcs11_session=self.pkc11_session,
                cert_label=self.config.pkcs11.cert_label,
                key_label=self.config.pkcs11.key_label,
                use_raw_mechanism=True,
            )
        except Exception as _e:
            self.logger.debug(f"pkcs11 signer creation failed, err: {_e}")
            return SealReply(
                transaction_id=in_data.transaction_id,
                data="",
                error=f"pkcs11 signer creation failed, err: {_e}",
                sealer_backend=self.service_name,
            )
        self.logger.debug("pkcs11 signer created")

        try:
            signature_meta = signers.PdfSignatureMetadata(
                field_name="Signature1",
                location=self.config.metadata.location,
                reason=self.config.metadata.reason,
                name=self.config.metadata.name,
                contact_info=self.config.metadata.contact_info,
                subfilter=SigSeedSubFilter.ADOBE_PKCS7_DETACHED
            )
        except Exception as _e:
            self.logger.debug(f"signature meta creation failed, err: {_e}")
            return SealReply(
                transaction_id=in_data.transaction_id,
                data="",
                error=f"signature meta creation failed, err: {_e}",
                sealer_backend=self.service_name,
            )

        signer = PdfSigner(
            signature_meta=signature_meta,
            signer=pkcs11_signer,
        )

        signed_pdf = BytesIO()

        try:
            signer.sign_pdf(
                pdf_out=pdf_writer,
                output=signed_pdf,
            )

        except PdfKeyNotAvailableError as _e:
            err_msg = f"input pdf is encrypted, err: {_e}"
            self.logger.error("error: " + err_msg)
            return SealReply(
                transaction_id=in_data.transaction_id,
                data="",
                error=err_msg,
                sealer_backend=self.service_name,
            )

        base64_encoded = base64.b64encode(signed_pdf.getvalue()).decode("utf-8")

        signed_pdf.close()

        self.logger.info("signing done")
        self.logger.debug(f"transaction_id: {in_data.transaction_id}")
        self.logger.debug(f"base64_data: {base64_encoded}")
    
        return SealReply(
            sealer_backend=self.service_name,
            transaction_id=in_data.transaction_id,
            data=base64_encoded,
            error="",
        )

class GRPCServer(Common):
    def __init__(self) -> None:
        super().__init__()

    def start(self):
        server = grpc.server(futures.ThreadPoolExecutor(max_workers=10))
        pb2_grpc.add_SealerServicer_to_server(Sealer(), server)


        if self.config.grpc_server.tls_enabled:
            assert self.config.grpc_server.private_key_path is not None
            assert self.config.grpc_server.certificate_chain_path is not None

            with open(self.config.grpc_server.private_key_path, 'rb') as f:
                private_key = f.read()
            with open(self.config.grpc_server.certificate_chain_path, 'rb') as f:
                certificate_chain = f.read()

            server_credentials = grpc.ssl_server_credentials( ( (private_key, certificate_chain), ) )

        server.add_secure_port(self.config.grpc_server.addr, server_credentials)
        server.start()
        time.sleep(2)
        open('/tmp/healthcheck','w')
        server.wait_for_termination()


if __name__ == "__main__":
    sealer = GRPCServer()
    sealer.start()