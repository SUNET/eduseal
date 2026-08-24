import base64
import os
from pathlib import Path
from unittest.mock import MagicMock

import pytest
import yaml

TESTS_DIR = Path(__file__).resolve().parent
TEST_PDFS_DIR = TESTS_DIR / "test_pdfs"
REPO_ROOT = TESTS_DIR.parent.parent

CONFIG_NAMESPACE = "test_validator"
SEALER_CONFIG_NAMESPACE = "test_sealer"


def _write_test_config(tmp_path: Path) -> Path:
    """Write a minimal config.yaml pointing at a temp trust-roots directory.

    The trust roots do not need to actually chain to the fixtures' signer
    (they are HARICA-signed and the chain is not available here). The
    validator just requires at least one loadable cert on startup.
    """
    certs_dir = tmp_path / "certs"
    certs_dir.mkdir()
    src_ca = REPO_ROOT / "developer_tools" / "pki" / "rootCA.crt"
    assert src_ca.is_file(), f"missing dev root CA: {src_ca}"
    (certs_dir / "rootCA.crt").write_bytes(src_ca.read_bytes())

    config_path = tmp_path / "config.yaml"
    config_path.write_text(
        yaml.safe_dump(
            {
                CONFIG_NAMESPACE: {
                    "grpc_server": {"addr": ":0"},
                    "validation_certificates_path": str(certs_dir),
                }
            }
        )
    )
    return config_path


@pytest.fixture(scope="session")
def validator(tmp_path_factory: pytest.TempPathFactory):
    tmp_path = tmp_path_factory.mktemp("validator")
    config_path = _write_test_config(tmp_path)

    os.environ["EDUSEAL_CONFIG_YAML"] = str(config_path)
    os.environ["EDUSEAL_CONFIG_NAMESPACE"] = CONFIG_NAMESPACE
    os.environ["EDUSEAL_LOG_PATH"] = str(tmp_path / "validator.log")

    from eduseal.validator.run import Validator

    return Validator()


@pytest.fixture(scope="session")
def grpc_context() -> MagicMock:
    return MagicMock(name="grpc.ServicerContext")


def _b64(pdf_name: str) -> str:
    path = TEST_PDFS_DIR / pdf_name
    assert path.is_file(), f"missing test fixture: {path}"
    return base64.b64encode(path.read_bytes()).decode("ascii")


@pytest.fixture(scope="session")
def untampered_pdf_b64() -> str:
    return _b64("test_untamped.pdf")


@pytest.fixture(scope="session")
def tampered_pdf_b64() -> str:
    return _b64("test_tamped_with.pdf")


def _write_sealer_config(tmp_path: Path) -> Path:
    config_path = tmp_path / "sealer_config.yaml"
    config_path.write_text(
        yaml.safe_dump(
            {
                SEALER_CONFIG_NAMESPACE: {
                    "grpc_server": {"addr": ":0"},
                    "queue": {
                        "username": "test",
                        "password": "test",
                        "addr": ["nats://localhost:4222"],
                    },
                    "pkcs11": {
                        "label": "test",
                        "pin": "test",
                        "module": "/nonexistent/libsofthsm2.so",
                        "key_label": "test",
                        "cert_label": "test",
                    },
                    "metadata": {
                        "location": "test",
                        "reason": "test",
                        "name": "test",
                        "contact_info": "test",
                        "field_name": "Signature1",
                    },
                }
            }
        )
    )
    return config_path


@pytest.fixture(scope="session")
def sealer(tmp_path_factory: pytest.TempPathFactory):
    tmp_path = tmp_path_factory.mktemp("sealer")
    config_path = _write_sealer_config(tmp_path)

    os.environ["EDUSEAL_CONFIG_YAML"] = str(config_path)
    os.environ["EDUSEAL_CONFIG_NAMESPACE"] = SEALER_CONFIG_NAMESPACE
    os.environ["EDUSEAL_LOG_PATH"] = str(tmp_path / "sealer.log")

    import eduseal.sealer.run as sealer_run

    # PKCS#11 is not available in unit tests; short-circuit the session opener
    # so we can exercise Seal()'s pre-PKCS11 branches (base64, PDF parse,
    # encryption detection).
    original_open = sealer_run.open_pkcs11_session
    sealer_run.open_pkcs11_session = lambda **kwargs: MagicMock(name="pkcs11.Session")
    try:
        instance = sealer_run.Sealer()
    finally:
        sealer_run.open_pkcs11_session = original_open

    return instance


def _make_encrypted_pdf_b64() -> str:
    from io import BytesIO

    from pyhanko.pdf_utils.writer import PdfFileWriter

    w = PdfFileWriter()
    w.encrypt("owner-pw", "user-pw")
    buf = BytesIO()
    w.write(buf)
    return base64.urlsafe_b64encode(buf.getvalue()).decode("ascii")


@pytest.fixture(scope="session")
def encrypted_pdf_b64() -> str:
    return _make_encrypted_pdf_b64()
