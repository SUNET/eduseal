"""Tests for Sealer.Seal() error branches."""

import asyncio

from eduseal.sealer.v1_sealer_pb2 import SealRequest


def _seal(sealer, req: SealRequest):
    return asyncio.run(sealer.Seal(req))


def test_seal_rejects_encrypted_pdf(sealer, encrypted_pdf_b64: str) -> None:
    reply = _seal(
        sealer,
        SealRequest(transaction_id="tx-encrypted", data=encrypted_pdf_b64),
    )

    assert reply.data == ""
    assert reply.transaction_id == "tx-encrypted"
    assert reply.error == (
        "input pdf is encrypted; please provide an unencrypted document"
    )
