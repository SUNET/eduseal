FROM nats:alpine

RUN apk add dpkg

RUN apk add wget

RUN wget https://github.com/nats-io/natscli/releases/download/v0.4.0/nats-0.4.0-amd64.deb

RUN echo "fdc0897c08b41f1587838fbcbe3bb84cc7da5dce0aa9afe34fd8da7a8197b202 nats-0.4.0-amd64.deb" | sha256sum -c -

RUN dpkg  --force-architecture -i nats-0.4.0-amd64.deb
