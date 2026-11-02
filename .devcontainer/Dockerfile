FROM mcr.microsoft.com/devcontainers/base:trixie

ARG GO_VERSION=1.27.3

RUN curl -fsSL "https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz" | tar -C /usr/local -xz

ENV GOPATH=/home/vscode/go
ENV PATH=/usr/local/go/bin:${GOPATH}/bin:${PATH}

USER vscode
RUN go install golang.org/x/tools/gopls@latest \
    && go install github.com/go-delve/delve/cmd/dlv@latest \
    && go install honnef.co/go/tools/cmd/staticcheck@latest

USER root
