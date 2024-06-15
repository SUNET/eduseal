FROM debian:stable

ENV DEBIAN_FRONTEND noninteractive

RUN apt-get update && apt-get install -y \
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
    libgdk-pixbuf2.0-0 \
    libgtk2.0-0 \
    softhsm2 \
    libsnappy-dev \
    libpango-1.0-0

RUN rm -rf /var/lib/apt/lists/*

WORKDIR /opt/sunet
COPY requirements.txt ./
COPY src ./src
COPY docker/sealer/softhsm/grpc/start.sh ./
COPY docker/sealer/softhsm/import_dev_pki_into_hsm.sh ./
COPY docker/sealer/softhsm/private.der ./pki/private.der
COPY docker/sealer/softhsm/example.crt ./pki/example.crt

RUN python3 -mvenv /opt/sunet/venv
RUN /opt/sunet/venv/bin/pip install --upgrade pip wheel
RUN /opt/sunet/venv/bin/pip install --index-url https://pypi.sunet.se/simple -r /opt/sunet/requirements.txt

RUN mkdir -p /var/log/sunet && chmod 770 /var/log/sunet

EXPOSE 8080

ENTRYPOINT [ "/bin/bash"]

CMD [ "/opt/sunet/start.sh"]