FROM nats:alpine

RUN set -eux; \
    apk add --no-cache dpkg wget ca-certificates; \
    wget -O /tmp/nats-0.4.0-amd64.deb https://github.com/nats-io/natscli/releases/download/v0.4.0/nats-0.4.0-amd64.deb; \
    echo "fdc0897c08b41f1587838fbcbe3bb84cc7da5dce0aa9afe34fd8da7a8197b202  /tmp/nats-0.4.0-amd64.deb" | sha256sum -c -; \
    dpkg --force-architecture -i /tmp/nats-0.4.0-amd64.deb; \
    rm -f /tmp/nats-0.4.0-amd64.deb; \
    apk del dpkg wget

