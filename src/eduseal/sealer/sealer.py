
import time
import logging
from io import BytesIO
import base64
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

from eduseal.models import PDFSignRequest, PDFSignReply
from eduseal.sealer.config import parse, CFG

class Sealer:
    def __init__(self, service_name: str):
        self.service_name = service_name
        self.logger = logging.getLogger(self.service_name)
        self.logger.setLevel(logging.DEBUG)
        
        ch = logging.StreamHandler()
        ch.setLevel(logging.DEBUG)

        formatter = logging.Formatter("%(asctime)s - %(name)s - %(levelname)s - %(message)s")

        ch.setFormatter(formatter)

        self.logger.addHandler(ch)
        #self.logger.propagate = False

        self.config: CFG = parse(log=self.logger)

        self.pkc11_session: Session
        self.init_pkcs11_session()

    def init_pkcs11_session(self) -> None:
        self.logger.info("init pkcs11 session")
        try:
            self.pkc11_session = open_pkcs11_session(
                lib_location=self.config.pkcs11.module, 
                slot_no=self.config.pkcs11.slot, 
                token_label=self.config.pkcs11.label,
                user_pin=self.config.pkcs11.pin,
            )
        except UserAlreadyLoggedIn:
            self.logger.info("pkcs11 user already logged in!")

    def marshal(self, data: PDFSignRequest) -> str:
        return json.dumps(data)

    def unmarshal(self, data: dict) -> PDFSignRequest:
        return PDFSignRequest.model_validate(data)

    def seal(self, in_data: PDFSignRequest)-> PDFSignReply:
        self.logger.debug("start sealing")
       # self.logger.debug(f"transaction_id: {in_data.transaction_id}")
       # self.logger.debug(f"base64_data: {in_data.base64_data}")

        try:
            pdf_writer = IncrementalPdfFileWriter(input_stream=BytesIO(base64.urlsafe_b64decode(in_data.base64_data)), strict=False)
        except PdfReadError as _e:
            self.logger.debug(f"input pdf is not valid, err: {_e}")
            return PDFSignReply(
                transaction_id=in_data.transaction_id,
                base64_data=None,
                create_ts=int(time.time()),
                error=f"input pdf is not valid, err: {_e}",
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
            return PDFSignReply(
                transaction_id=in_data.transaction_id,
                base64_data=None,
                create_ts=int(time.time()),
                error=f"pkcs11 signer creation failed, err: {_e}",
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
        self.logger.debug("signature meta created")

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

            return PDFSignReply(
                transaction_id=in_data.transaction_id,
                base64_data=None,
                create_ts=int(time.time()),
                error=err_msg,
            )

        base64_encoded = base64.b64encode(signed_pdf.getvalue()).decode("utf-8")

        signed_pdf.close()

        self.logger.info("signing done")
        self.logger.debug(f"transaction_id: {in_data.transaction_id}")
        self.logger.debug(f"base64_data: {base64_encoded}")
    
        return PDFSignReply(
            transaction_id=in_data.transaction_id,
            base64_data=base64_encoded,
            create_ts=int(time.time()),
            error="",
        )