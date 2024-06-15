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
COPY dockerfiles/validator_start.sh ./
#COPY ./developer_pki/pki /opt/sunet/pki

RUN python3 -mvenv /opt/sunet/venv
RUN /opt/sunet/venv/bin/pip install --upgrade pip wheel
RUN /opt/sunet/venv/bin/pip install --index-url https://pypi.sunet.se/simple -r /opt/sunet/requirements.txt

RUN mkdir -p /var/log/sunet && chmod 770 /var/log/sunet

EXPOSE 8080

#HEALTHCHECK --interval=27s CMD curl --max-time 20 http://localhost:8080/status/ | grep -q OK

ENTRYPOINT [ "/bin/bash"]

CMD [ "/opt/sunet/validator_start.sh"]