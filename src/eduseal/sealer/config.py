from pydantic import BaseModel
from typing import Optional, List
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

class GRPCServer(BaseModel):
    addr: str
    tls_enabled: bool = False
    private_key_path: Optional[str] = None
    certificate_chain_path: Optional[str] = None

class Queue(BaseModel):
    username: str
    password: str
    addr: List[str]

class CFG(BaseModel):
    grpc_server: GRPCServer
    queue: Queue
    pkcs11: PKCS11
    metadata: PdfSignatureMetadata

def parse(log: Logger) -> CFG:
    file_name = os.getenv("EDUSEAL_CONFIG_YAML")
    if file_name is None:
        log.error("no config file env variable found")
        sys.exit(1)

    config_namespace = os.getenv("EDUSEAL_CONFIG_NAMESPACE")
    if config_namespace is None:
        log.error("no config namespace env variable found")
        sys.exit(1)

    try:
        with open(file_name, "r")as f:
             data = yaml.load(f, yaml.FullLoader)
             cfg = CFG.model_validate(data[config_namespace])
    except Exception as e:
            log.error(f"open file {file_name} failed, error: {e}")
            sys.exit(1)
    return cfg