FROM debian:stable

ENV DEBIAN_FRONTEND=noninteractive

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
    procps \
    opensc-pkcs11 \
    opensc \
    usbutils \
    libpcsclite1 \
    libgdk-pixbuf2.0-0 \
    libgtk2.0-0 \
    libpango-1.0-0

RUN rm -rf /var/lib/apt/lists/*

WORKDIR /opt/sunet
COPY requirements.txt ./
COPY src ./src
COPY docker/validator/start.sh ./

RUN python3 -mvenv /opt/sunet/venv
RUN /opt/sunet/venv/bin/pip install --upgrade pip wheel
RUN /opt/sunet/venv/bin/pip install --index-url https://pypi.sunet.se/simple -r /opt/sunet/requirements.txt

RUN mkdir -p /var/log/sunet && chmod 770 /var/log/sunet

HEALTHCHECK --interval=5s CMD ls /tmp/healthcheck || exit 1

ENTRYPOINT [ "/bin/bash"]

CMD [ "/opt/sunet/start.sh"]