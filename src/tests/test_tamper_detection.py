"""Regression tests for Validator.Validate() against tampered PDFs.

These tests drive the gRPC servicer method directly with the fixture PDFs
and pin the fields of the returned ValidateReply.
"""

from unittest.mock import MagicMock

import pytest

from eduseal.validator.v1_validator_pb2 import ValidateRequest


def test_validate_untampered_pdf_reports_intact(
    validator, grpc_context: MagicMock, untampered_pdf_b64: str
) -> None:
    reply = validator.Validate(ValidateRequest(data=untampered_pdf_b64), grpc_context)

    assert reply.error == ""
    assert reply.intact_signature is True
    assert reply.coverage == "ENTIRE_FILE"
    assert reply.modification_level == "NONE"
    assert reply.docmdp_ok is True


def test_validate_tampered_pdf_is_not_intact(
    validator, grpc_context: MagicMock, tampered_pdf_b64: str
) -> None:
    """The forged PDF must not be reported as intact even though the raw
    signature bytes still verify. This is the regression the college found."""
    reply = validator.Validate(ValidateRequest(data=tampered_pdf_b64), grpc_context)

    assert reply.error == ""
    assert reply.intact_signature is False
    assert reply.valid_signature is False
    assert reply.coverage == "ENTIRE_REVISION"
    assert reply.modification_level == "OTHER"
    assert reply.docmdp_ok is False


def test_validate_rejects_invalid_base64(
    validator, grpc_context: MagicMock
) -> None:
    reply = validator.Validate(ValidateRequest(data="!!!not-base64!!!"), grpc_context)

    assert reply.intact_signature is False
    assert reply.valid_signature is False
    assert reply.error != ""


def test_validate_rejects_non_pdf_payload(
    validator, grpc_context: MagicMock
) -> None:
    import base64

    reply = validator.Validate(
        ValidateRequest(data=base64.b64encode(b"not a pdf").decode("ascii")),
        grpc_context,
    )

    assert reply.intact_signature is False
    assert reply.valid_signature is False
    assert reply.error != ""


def test_validate_rejects_pdf_without_signature(
    validator, grpc_context: MagicMock
) -> None:
    import base64
    from io import BytesIO

    from pyhanko.pdf_utils.writer import PdfFileWriter

    buf = BytesIO()
    PdfFileWriter().write(buf)
    payload = base64.b64encode(buf.getvalue()).decode("ascii")

    reply = validator.Validate(ValidateRequest(data=payload), grpc_context)

    assert reply.intact_signature is False
    assert reply.valid_signature is False
    assert reply.error == "No signature found"


def test_validate_missing_transaction_id_reports_empty(
    validator,
    grpc_context: MagicMock,
    untampered_pdf_b64: str,
    monkeypatch: "pytest.MonkeyPatch",
) -> None:
    """A sealed PDF must carry a transaction_id keyword; missing it is a
    hard error so callers cannot correlate a result back to a request."""
    def _raise(pdf):
        raise ValueError("No transaction_id found in sealed PDF")

    monkeypatch.setattr(validator, "get_transaction_id_from_keywords", _raise)

    reply = validator.Validate(ValidateRequest(data=untampered_pdf_b64), grpc_context)

    assert reply.error == "No transaction_id found in sealed PDF"
    assert reply.transaction_id == ""
    assert reply.intact_signature is False
    assert reply.valid_signature is False
