from pydantic import BaseModel
from typing import Optional
import yaml
import os
import sys
from logging import Logger

class PdfSignatureMetadata(BaseModel):
    location: str
    reason: str
    name: str
    contact_info: str
    field_name: str

class PKCS11(BaseModel):
    label: str
    pin: str
    module: str
    key_label: Optional[str] = None
    cert_label: Optional[str] = None
    slot: Optional[int] = None


class CFG(BaseModel):
    pkcs11: PKCS11
    metadata: PdfSignatureMetadata

def parse(log: Logger) -> CFG:
    file_name = os.getenv("EDUSEAL_CONFIG_YAML")
    if file_name is None:
        log.error("no config file env variable found")
        sys.exit(1)

    service_name = os.getenv("EDUSEAL_SERVICE_NAME")
    if service_name is None:
        log.error("no service name env variable found")
        sys.exit(1)

    try:
        with open(file_name, "r")as f:
             data = yaml.load(f, yaml.FullLoader)
             cfg = CFG.model_validate(data[service_name])
    except Exception as e:
            log.error(f"open file {file_name} failed, error: {e}")
            sys.exit(1)
    return cfg