import logging
from io import BytesIO
import base64
import json
import os
from typing import Optional
import itertools

from pyhanko.sign.validation import validate_pdf_signature
from pyhanko_certvalidator import ValidationContext
from pyhanko_certvalidator.registry import TrustRootList
from pyhanko.keys import load_cert_from_pemder
from pyhanko.pdf_utils.reader import PdfFileReader

from eduseal.validator.v1_validator_pb2 import ValidateReply, ValidateRequest
from eduseal.validator.config import parse, CFG

class Validator:
    def __init__(self):
        self.service_name = os.getenv("EDUSEAL_SERVICE_NAME", "eduseal_validator")
        self.logger = logging.getLogger(self.service_name)
        self.logger.setLevel(logging.DEBUG)
        
        ch = logging.StreamHandler()
        ch.setLevel(logging.DEBUG)

        formatter = logging.Formatter("%(asctime)s - %(name)s - %(levelname)s - %(message)s")

        ch.setFormatter(formatter)

        self.logger.addHandler(ch)
        self.logger.propagate = False

        self.config: CFG = parse(log=self.logger)

        self.validation_context = ValidationContext(
            trust_roots=self.build_trust_roots(),
        )

    def validate(self, in_data: ValidateRequest) -> ValidateReply:
        try:
            pdf = PdfFileReader(BytesIO(base64.b64decode(in_data.pdf.encode("utf-8"), validate=False)), strict=False)
        except Exception as e:
            self.logger.error(f"Error reading PDF: {e}")
            return ValidateReply(
                service_name=self.service_name,
                error="Error reading PDF"
            )

        if len(pdf.embedded_signatures) == 0:
            self.logger.error("No signature found")
            return ValidateReply(
                service_name=self.service_name,
                error="No signature found"
            )


        root_cert = load_cert_from_pemder('/validation_certificates/rootCA.crt')
        vc = ValidationContext(trust_roots=[root_cert])

        try:
            status = validate_pdf_signature(
                embedded_sig=pdf.embedded_signatures[0],
                signer_validation_context=vc,
            )
        except Exception as e:
            self.logger.error(f"Validation error: {e}")
            return ValidateReply(
                service_name=self.service_name,
                error="Validation error"
            )

        try:
            transaction_id = self.get_transaction_id_from_keywords(pdf=pdf)
        except Exception as e:
            self.logger.error(f"Error getting transaction_id: {e}")
            return ValidateReply(
                service_name=self.service_name,
                error="Error getting transaction_id"
            )

        self.logger.info(f"Validate a signed base64 PDF, transaction_id:{transaction_id}")
        #self.logger.info(f"Signature path: {status.validation_path}")
        self.logger.info(f"Signature time: {status.validation_time}")


        return ValidateReply(
            service_name=self.service_name,
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

    def build_trust_roots(self) -> TrustRootList:
        trust_root_list: TrustRootList = None
        for file in os.listdir(self.config.validation_certificates_path):
            filename = os.fsdecode(file)
            if filename.endswith(".crt"):
                self.logger.info(f"found trust root file: {filename}")
                abs_path = self.config.validation_certificates_path + "/" + filename
                self.logger.info(f"trust root absolute path: {abs_path}")
                itertools.chain(load_cert_from_pemder(abs_path), trust_root_list)
        return trust_root_list