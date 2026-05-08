import logging
from io import BytesIO
import os
import base64
import signal
import ssl
import json
import time

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


import asyncio
from nats.aio.client import Client as NATS
from nats.js.api import StreamConfig, RetentionPolicy, ConsumerConfig, AckPolicy

class Common():
    def __init__(self) -> None:
        self.service_name = os.getenv("EDUSEAL_SERVICE_NAME", "eduseal_sealer")
        self.logger = logging.getLogger(self.service_name)
        self.logger.setLevel(logging.DEBUG)
        
        # Clear any existing handlers to avoid duplicates
        self.logger.handlers.clear()
        
        # Console handler
        ch = logging.StreamHandler()
        ch.setLevel(logging.DEBUG)
        ch.setFormatter(logging.Formatter("%(asctime)s - %(name)s - %(levelname)s - %(message)s"))
        self.logger.addHandler(ch)
        
        # File handler
        fh = logging.FileHandler("/var/log/sunet/sealer.log")
        fh.setLevel(logging.DEBUG)
        fh.setFormatter(logging.Formatter("%(asctime)s - %(name)s - %(levelname)s - %(message)s"))
        self.logger.addHandler(fh)
        
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

    async def Seal(self, in_data: SealRequest)-> SealReply:
        self.logger.debug("start sealing")
        self.logger.debug(f"transaction_id: {in_data.transaction_id}")
        seal_start_time = time.monotonic()

        try:
            decoded_pdf = base64.urlsafe_b64decode(in_data.data)
        except Exception as _e:
            self.logger.debug(f"input pdf is not base64 encoded, err: {_e}")
            return SealReply(
                transaction_id=in_data.transaction_id,
                data="",
                error=f"input pdf is not base64 encoded, err: {_e}",
                sealer_backend=self.service_name,
            )

        try:
            pdf_writer = IncrementalPdfFileWriter(input_stream=BytesIO(decoded_pdf), strict=False)
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

        signed_pdf = BytesIO()

        try:
            await signers.async_sign_pdf(
                pdf_out=pdf_writer,
                output=signed_pdf,
                signer=pkcs11_signer,
                signature_meta=signature_meta,
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

        try:
            base64_encoded = base64.b64encode(signed_pdf.getvalue()).decode("utf-8")
        except Exception as _e:
            err_msg = f"output pdf base64 encoding failed, err: {_e}"
            self.logger.error("error: " + err_msg)
            return SealReply(
                transaction_id=in_data.transaction_id,
                data="",
                error=err_msg,
                sealer_backend=self.service_name,
            )

        signed_pdf.close()

        seal_elapsed = time.monotonic() - seal_start_time
        if seal_elapsed > 4.0:
            self.logger.error(f"sealing took {seal_elapsed:.2f}s (>{4.0}s) for transaction_id: {in_data.transaction_id}")

        self.logger.info(f"signing done {in_data.transaction_id}")
    
        return SealReply(
            sealer_backend=self.service_name,
            transaction_id=in_data.transaction_id,
            data=base64_encoded,
            error="",
        )

class QueueServer(Common):
    def __init__(self) -> None:
        super().__init__()
        self.sealer = Sealer()
        self._shutdown_event = asyncio.Event()

    async def _graceful_nats_shutdown(self, nc: NATS):
        """Gracefully shut down the NATS connection."""
        self.logger.info("Graceful shutdown: draining NATS connection...")
        try:
            await nc.drain()
        except Exception as e:
            self.logger.warning(f"Error during NATS drain: {e}")
            try:
                if not nc.is_closed:
                    await nc.close()
            except Exception as close_e:
                self.logger.warning(f"Error during NATS close: {close_e}")

        self.logger.info("Graceful shutdown complete")

        if os.path.exists('/tmp/healthcheck'):
            os.remove('/tmp/healthcheck')

    async def start(self):
        self.logger.debug("start queue server")
        nc = NATS()
        js = nc.jetstream()

        def signal_handler():
            if self._shutdown_event.is_set():
                return
            self.logger.info("Received shutdown signal, initiating graceful shutdown (waiting for any in-progress sealing to complete)...")
            self._shutdown_event.set()

        for sig in ("SIGINT", "SIGTERM"):
            asyncio.get_running_loop().add_signal_handler(
                getattr(signal, sig), signal_handler
            )

        async def disconnected_cb():
            self.logger.info("Queue: got disconnected...")
            if os.path.exists('/tmp/healthcheck'):
                os.remove('/tmp/healthcheck')
                self.logger.info("Healthcheck removed (unhealthy)")

        async def reconnected_cb():
            self.logger.info("Queue: got reconnected...")
            open('/tmp/healthcheck', 'w').close()
            self.logger.info("Healthcheck restored (healthy)")

        async def error_cb(e):
            self.logger.error(f"Queue: error: {e}")

        async def closed_cb():
            self.logger.info("Queue: Connection to NATS is closed...")

        max_retries = self.config.queue_retry.max_retries
        retry_delay = self.config.queue_retry.retry_delay

        tls_context = None
        if self.config.queue.tls.enabled:
            tls_context = ssl.create_default_context(purpose=ssl.Purpose.SERVER_AUTH)
            if self.config.queue.tls.root_ca_path:
                tls_context.load_verify_locations(self.config.queue.tls.root_ca_path)
            if self.config.queue.tls.cert_file_path and self.config.queue.tls.key_file_path:
                tls_context.load_cert_chain(
                    certfile=self.config.queue.tls.cert_file_path,
                    keyfile=self.config.queue.tls.key_file_path,
                )

        for attempt in range(1, max_retries + 1):
            try:
                connect_kwargs = dict(
                    servers=self.config.queue.addr,
                    user=self.config.queue.username,
                    password=self.config.queue.password,
                    closed_cb=closed_cb,
                    allow_reconnect=True,
                    reconnected_cb=reconnected_cb,
                    disconnected_cb=disconnected_cb,
                    error_cb=error_cb,
                    max_reconnect_attempts=-1,
                    reconnect_time_wait=5,
                )
                if tls_context is not None:
                    connect_kwargs["tls"] = tls_context
                    if self.config.queue.tls.server_name:
                        connect_kwargs["tls_hostname"] = self.config.queue.tls.server_name

                await nc.connect(**connect_kwargs)
                self.logger.info(f"Connected to NATS at {nc.connected_url.netloc}...")
                open('/tmp/healthcheck', 'w').close()
                self.logger.info("Healthcheck created (healthy)")
                break
            except Exception as e:
                self.logger.error(f"NATS connection attempt {attempt}/{max_retries} failed: {e}")
                if attempt < max_retries:
                    self.logger.info(f"Retrying NATS connection in {retry_delay}s...")
                    await asyncio.sleep(retry_delay)
                else:
                    self.logger.error(f"All {max_retries} NATS connection attempts failed, exiting")
                    raise

        async def sealer_msg_handler(msg):
            self.logger.info(f"Received a message on subject: {msg.subject} header: {msg.headers}")

            await msg.in_progress()

            max_retries = self.config.seal_retry.max_retries
            retry_delay = self.config.seal_retry.retry_delay
            last_reply = None

            for attempt in range(1, max_retries + 1):
                self.logger.info(f"Seal attempt {attempt}/{max_retries} for msg {msg.headers.get('Nats-Msg-Id', 'unknown')}")

                reply = await self.sealer.Seal(in_data=SealRequest(**json.loads(msg.data)))
                last_reply = reply

                if not reply.error:
                    self.logger.info(f"Seal succeeded on attempt {attempt}/{max_retries}")
                    break

                self.logger.warning(f"Seal attempt {attempt}/{max_retries} failed: {reply.error}")

                if attempt < max_retries:
                    self.logger.info(f"Re-initializing PKCS11 session before retry (waiting {retry_delay}s)")
                    await asyncio.sleep(retry_delay)
                    await msg.in_progress()
                    try:
                        self.sealer.init_pkcs11_session()
                    except Exception as _e:
                        self.logger.error(f"PKCS11 session re-init failed: {_e}")
                else:
                    self.logger.error(f"All {max_retries} seal attempts failed for msg {msg.headers.get('Nats-Msg-Id', 'unknown')}")

            if last_reply.error:
                self.logger.error(f"Nacking message {msg.headers.get('Nats-Msg-Id', 'unknown')} after all retries exhausted")
                await msg.nak(delay=retry_delay)
                return

            d = dict(
                transaction_id=last_reply.transaction_id,
                data=last_reply.data,
                error=last_reply.error,
                sealer_backend=last_reply.sealer_backend,
            )
            try:
                await js.publish(
                    subject="CACHE",
                    payload=json.dumps(d).encode(),
                    headers={"Nats-Msg-Id": msg.headers["Nats-Msg-Id"]},
                )
                await msg.ack()
            except Exception as e:
                self.logger.error(f"Failed to publish/ack sealed result: {e}")
                try:
                    await msg.nak(delay=self.config.seal_retry.retry_delay)
                except Exception as nak_e:
                    self.logger.error(f"Failed to nak message: {nak_e}")
                raise


        # Ensure JetStream stream and consumer exist (matching apigw seal_stream.go config)
        try:
            await js.find_stream_name_by_subject("SEAL")
            self.logger.info("JetStream stream for subject SEAL already exists")
        except Exception:
            self.logger.info("Creating JetStream stream 'seal_stream' for subject SEAL")
            await js.add_stream(StreamConfig(
                name="seal_stream",
                subjects=["SEAL"],
                retention=RetentionPolicy.WORK_QUEUE,
                no_ack=False,
            ))

        try:
            await js.consumer_info("seal_stream", "sealer")
            self.logger.info("JetStream consumer 'sealer' already exists")
        except Exception:
            self.logger.info("Creating JetStream consumer 'sealer'")
            await js.add_consumer("seal_stream", ConsumerConfig(
                durable_name="sealer",
                ack_policy=AckPolicy.EXPLICIT,
                filter_subject="SEAL",
                max_deliver=5,
            ))

        sub_sealer = await js.pull_subscribe(subject="SEAL", durable="sealer", stream="seal_stream")
        healthcheck_path = '/tmp/healthcheck'

        while not self._shutdown_event.is_set():
            # Periodic NATS connectivity check
            if nc.is_connected:
                if not os.path.exists(healthcheck_path):
                    open(healthcheck_path, 'w').close()
                    self.logger.info("Healthcheck restored (NATS connected)")
            else:
                if os.path.exists(healthcheck_path):
                    os.remove(healthcheck_path)
                    self.logger.warning("Healthcheck removed (NATS not connected)")

            try:
                msgs = await sub_sealer.fetch(1, timeout=1)
            except asyncio.TimeoutError:
                continue
            except Exception as e:
                if self._shutdown_event.is_set():
                    break
                self.logger.error(f"Fetch error: {e}")
                await asyncio.sleep(1)
                continue

            try:
                self.logger.info(f"msg: {msgs[0].headers}")
                await sealer_msg_handler(msgs[0])
            except Exception as e:
                self.logger.error(f"Message handler error: {e}")
                await asyncio.sleep(1)

        await self._graceful_nats_shutdown(nc)
        asyncio.get_running_loop().stop()


if __name__ == "__main__":
    healthcheck_path = '/tmp/healthcheck'
    server = QueueServer()
    loop = asyncio.new_event_loop()
    asyncio.set_event_loop(loop)
    try:
        loop.run_until_complete(server.start())
        loop.run_forever()
    except Exception as e:
        server.logger.error(f"error {e}")
    finally:
        if os.path.exists(healthcheck_path):
            os.remove(healthcheck_path)
        # Cancel all pending tasks to avoid "Task was destroyed but it is pending!" warnings
        pending = asyncio.all_tasks(loop)
        for task in pending:
            task.cancel()
        if pending:
            loop.run_until_complete(asyncio.gather(*pending, return_exceptions=True))
        loop.close()