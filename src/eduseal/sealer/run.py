import logging
from io import BytesIO
import os
import base64
import signal
import json

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
from nats.js.api import ConsumerConfig

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

    async def Seal(self, in_data: SealRequest)-> SealReply:
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

        base64_encoded = base64.b64encode(signed_pdf.getvalue()).decode("utf-8")

        signed_pdf.close()

        self.logger.info(f"signing done {in_data.transaction_id}")
    
        return SealReply(
            sealer_backend=self.service_name,
            transaction_id=in_data.transaction_id,
            data=base64_encoded,
            error="",
        )

class QueueServer4(Common):
    def __init__(self) -> None:
        super().__init__()
        self.sealer = Sealer()

    async def start(self):
        self.logger.debug("start queue server")
        nc = NATS()
        js = nc.jetstream()

        async def stop():
            await asyncio.sleep(1)
            asyncio.get_running_loop().stop()

        def signal_handler():
            if nc.is_closed:
                return
            print("Disconnecting...")
            asyncio.create_task(nc.close())
            asyncio.create_task(stop())

        for sig in ("SIGINT", "SIGTERM"):
            asyncio.get_running_loop().add_signal_handler(
                getattr(signal, sig), signal_handler
            )

        async def disconnected_cb():
            self.logger.info("Got disconnected...")

        async def reconnected_cb():
            self.logger.info("Got reconnected...")

        async def error_cb(e):
            self.logger.error(f"error: {e}")

        async def closed_cb():
            self.logger.info("Connection to NATS is closed...")

        await nc.connect(
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
        self.logger.info(f"Connected to NATS at {nc.connected_url.netloc}...")

        async def help_request(msg):
            self.logger.info(f"Received a message on subject: {msg.subject} header: {msg.headers}")

            await msg.in_progress()

            reply = await self.sealer.Seal(in_data=SealRequest(**json.loads(msg.data)))
            d = dict(
                transaction_id=reply.transaction_id,
                data=reply.data,
                error=reply.error,
                sealer_backend=reply.sealer_backend,
            )
            await js.publish(
                subject="CACHE",
                payload=json.dumps(d).encode(),
                headers={"Nats-Msg-Id": msg.headers["Nats-Msg-Id"]},
            )
            await msg.ack()


        sub = await js.pull_subscribe(subject="SEAL", durable="sealer")

        while True:
            msgs = await sub.fetch(1, timeout=31560000)
            self.logger.info(f"msg: {msgs[0].headers}")
            await help_request(msgs[0])

if __name__ == "__main__":
    server = QueueServer4()
    loop = asyncio.get_event_loop()
    try:
        loop.run_until_complete(server.start())
        loop.run_forever()
        loop.close()
    except Exception as e:
        server.logger.error(f"error {e}")
        pass