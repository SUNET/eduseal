from pydantic import BaseModel
from typing import Optional
import yaml
import os
import sys
from logging import Logger

class RedisConfig(BaseModel):
    host: str
    port: int
    db: int
    password: Optional[str] = None

class CFG(BaseModel):
    validate_queue_name: str
    trust_root_folder: str
    redis: RedisConfig

def parse(log: Logger) -> CFG:
    file_name = os.getenv("CONFIG_YAML")
    if file_name is None:
        log.error("no config file env variable found")
        sys.exit(1)

    try:
        with open(file_name, "r")as f:
             data = yaml.load(f, yaml.FullLoader)
             cfg = CFG.model_validate(data["py_pdfvalidator"])
    except Exception as e:
            log.error(f"open file {file_name} failed, error: {e}")
            sys.exit(1)
    return cfg