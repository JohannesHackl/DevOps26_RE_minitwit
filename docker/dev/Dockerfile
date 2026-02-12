##courtesy to jheise for providing the original starting point from which this Dockerfile has developed: https://github.com/jheise/ubuntu-golang/blob/master/Dockerfile
FROM ubuntu:24.04

ARG GO_VERSION
ENV GO_VERSION=${GO_VERSION}

#Installation phase
RUN apt-get update
RUN apt-get install -y wget git gcc

RUN wget -P /tmp "https://dl.google.com/go/go${GO_VERSION}.linux-amd64.tar.gz"

RUN tar -C /usr/local -xzf "/tmp/go${GO_VERSION}.linux-amd64.tar.gz"

# Post install clean up
RUN rm "/tmp/go${GO_VERSION}.linux-amd64.tar.gz"
RUN apt-get clean && rm -rf /var/lib/apt/lists/*

# Setup runtime environement
ENV GOPATH=/go
ENV PATH=$GOPATH/bin:/usr/local/go/bin:$PATH
RUN mkdir -p "$GOPATH/src" "$GOPATH/bin" && chmod -R 777 "$GOPATH"

# Script for compiling the go project.
RUN printf ' #!/bin/bash\n\
set -e\n\
cd $GOPATH/src\n\
echo "Downloading dependencies..."\n\
go mod download\n\
echo "Building minitwit..."\n\
go build -o bin/minitwit ./...\n\
echo "Build complete: Installed in $GOPATH/bin"\n' > /usr/local/bin/gobuild && \
chmod +x /usr/local/bin/gobuild

WORKDIR $GOPATH/src

EXPOSE 8080

CMD ["/bin/bash"]
