##courtesy to jheise for providing the original starting point from which this Dockerfile has developed: https://github.com/jheise/ubuntu-golang/blob/master/Dockerfile
FROM ubuntu:24.04

ARG GO_VERSION
ENV GO_VERSION=${GO_VERSION}

# Installation phase
RUN apt-get update && apt-get install -y \
    wget \
    git \
    gcc \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

# Dynamically detect architecture and download the correct Go binary
RUN set -eux; \
    arch="$(dpkg --print-architecture)"; \
    case "${arch}" in \
        amd64) export GO_ARCH='amd64' ;; \
        arm64) export GO_ARCH='arm64' ;; \
        armhf) export GO_ARCH='armv6l' ;; \
        i386)  export GO_ARCH='386' ;; \
        *) echo "Unsupported architecture: ${arch}"; exit 1 ;; \
    esac; \
    wget -O /tmp/go.tar.gz "https://dl.google.com/go/go${GO_VERSION}.linux-${GO_ARCH}.tar.gz"; \
    tar -C /usr/local -xzf /tmp/go.tar.gz; \
    rm /tmp/go.tar.gz

# Setup Go environment
ENV GOPATH=/go
ENV PATH=$GOPATH/bin:/usr/local/go/bin:$PATH
RUN mkdir -p "$GOPATH/bin" && chmod -R 777 "$GOPATH"

# Create /app directory and set permissions for development
RUN mkdir -p /app && chmod -R 777 /app

# Script for compiling the go project.
RUN printf '#!/bin/bash\n\
set -e\n\
cd /app\n\
echo "Downloading dependencies..."\n\
go mod download\n\
echo "Building minitwit..."\n\
go build -o minitwit .\n\
echo "Build complete."\n' > /usr/local/bin/gobuild && \
chmod +x /usr/local/bin/gobuild

# Set the working directory to /app
WORKDIR /app


EXPOSE 5001

CMD ["/bin/bash"]