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
    validation_certificates_path: str
    redis: RedisConfig

def parse(log: Logger) -> CFG:
    file_name = os.getenv("EDUSEAL_CONFIG_YAML")
    if file_name is None:
        log.error("no config file env variable found")
        sys.exit(1)

    try:
        with open(file_name, "r")as f:
             data = yaml.load(f, yaml.FullLoader)
             cfg = CFG.model_validate(data["validator"])
    except Exception as e:
            log.error(f"open file {file_name} failed, error: {e}")
            sys.exit(1)
    return cfg