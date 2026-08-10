import os
import threading
import logging
from typing import Callable, Optional


class CertReloader:
    """Watches cert+key files by mtime and calls back on change."""

    def __init__(
        self,
        cert_path: str,
        key_path: str,
        on_reload: Callable[[], None],
        logger: logging.Logger,
        interval: float = 30.0,
    ):
        self._cert_path = cert_path
        self._key_path = key_path
        self._on_reload = on_reload
        self._logger = logger
        self._interval = interval
        self._stop = threading.Event()
        self._cert_mtime: Optional[float] = self._mtime(cert_path)
        self._key_mtime: Optional[float] = self._mtime(key_path)
        self._thread = threading.Thread(target=self._poll, daemon=True, name="cert-reloader")
        self._thread.start()

    @staticmethod
    def _mtime(path: str) -> Optional[float]:
        try:
            return os.stat(path).st_mtime
        except OSError:
            return None

    def _poll(self) -> None:
        while not self._stop.wait(self._interval):
            cert_mt = self._mtime(self._cert_path)
            key_mt = self._mtime(self._key_path)
            if cert_mt != self._cert_mtime or key_mt != self._key_mtime:
                self._cert_mtime = cert_mt
                self._key_mtime = key_mt
                self._logger.info("TLS certificate files changed, reloading (%s, %s)", self._cert_path, self._key_path)
                try:
                    self._on_reload()
                except Exception:
                    self._logger.exception("cert reload callback failed")

    def stop(self) -> None:
        self._stop.set()
        self._thread.join(timeout=5)
