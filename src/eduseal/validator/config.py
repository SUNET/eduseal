from pydantic import BaseModel
from typing import Optional
import yaml
import os
import sys
from logging import Logger


class GRPCServer(BaseModel):
    addr: str
    tls_enabled: bool = False
    private_key_path: Optional[str] = None
    certificate_chain_path: Optional[str] = None

class CFG(BaseModel):
    grpc_server: GRPCServer
    validation_certificates_path: str

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