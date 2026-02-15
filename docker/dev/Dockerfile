FROM ubuntu:24.04

ARG GO_VERSION=1.24.0
RUN apt-get update && apt-get install -y \
    curl \
    gcc \
    libc6-dev \
    git \
    && rm -rf /var/lib/apt/lists/*

# detect architecture automatically
RUN ARCH=$(dpkg --print-architecture) && \
    curl -OL https://go.dev/dl/go${GO_VERSION}.linux-${ARCH}.tar.gz && \
    tar -C /usr/local -xzf go${GO_VERSION}.linux-${ARCH}.tar.gz && \
    rm go${GO_VERSION}.linux-${ARCH}.tar.gz

ENV PATH=$PATH:/usr/local/go/bin
ENV GOPATH=/go

RUN mkdir -p /go/src /go/bin

RUN echo '#!/bin/bash\nCGO_ENABLED=1 go build -o /go/bin/minitwit /go/src/minitwit.go' \
    > /usr/local/bin/gobuild && chmod +x /usr/local/bin/gobuild

WORKDIR /go/src