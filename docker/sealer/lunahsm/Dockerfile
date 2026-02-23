FROM debian:stable-slim

ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get update && apt-get install -y --no-install-recommends \
    python3-pip \
    python3-venv \
    python3-dev \
    build-essential \
    netcat-traditional \
    dnsutils \
    openssl \
    curl \
    procps \
    opensc \
    opensc-pkcs11 \
    usbutils \
    libpcsclite1 \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /opt/sunet
COPY requirements.txt ./
RUN python3 -mvenv /opt/sunet/venv \
    && /opt/sunet/venv/bin/pip install --no-cache-dir --upgrade pip wheel \
    && /opt/sunet/venv/bin/pip install --no-cache-dir --index-url https://pypi.sunet.se/simple -r /opt/sunet/requirements.txt

# Remove build dependencies no longer needed at runtime
RUN apt-get purge -y --auto-remove build-essential python3-dev && rm -rf /var/lib/apt/lists/*

COPY src ./src
COPY docker/sealer/lunahsm/start.sh ./
COPY docker/sealer/lunahsm/snow_white.sh ./

RUN mkdir -p /var/log/sunet && chmod 770 /var/log/sunet

HEALTHCHECK --interval=5s CMD ls /tmp/healthcheck || exit 1

ENTRYPOINT [ "/bin/bash"]

CMD [ "/opt/sunet/start.sh" ]