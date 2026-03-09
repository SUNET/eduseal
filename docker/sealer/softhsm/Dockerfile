FROM debian:stable-slim

ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    python3-pip \
    netcat-traditional \
    dnsutils \
    git \
    openssl \
    vim-tiny \
    bsdmainutils \
    curl \
    wget \
    unzip \
    python3-venv \
    iputils-ping \
    procps \
    net-tools \
    bind9-host \
    opensc-pkcs11 \
    opensc \
    usbutils \
    libpcsclite1 \
    libgdk-pixbuf-xlib-2.0-0 \
    libgtk2.0-0 \
    softhsm2 \
    && rm -rf /var/lib/apt/lists/*
#libsnappy-dev \

WORKDIR /opt/sunet
COPY requirements.txt ./
COPY src ./src
COPY docker/sealer/softhsm/start.sh ./
COPY docker/sealer/softhsm/import_dev_pki_into_hsm.sh ./

RUN python3 -mvenv /opt/sunet/venv \
    && /opt/sunet/venv/bin/pip install --no-cache-dir --upgrade pip wheel \
    && /opt/sunet/venv/bin/pip install --no-cache-dir --index-url https://pypi.sunet.se/simple -r /opt/sunet/requirements.txt

RUN mkdir -p /var/log/sunet && chmod 770 /var/log/sunet

HEALTHCHECK --interval=5s CMD ls /tmp/healthcheck || exit 1

ENTRYPOINT [ "/bin/bash"]

CMD [ "/opt/sunet/start.sh"]