#!/usr/bin/env bash

if [ "$1" = "build" ]; then
    if [ -f "./docker/dev/Dockerfile" ]; then
        echo "[INFO] Building docker image..."
        docker build --build-arg GO_VERSION=1.25.0 -t ubuntu-go ./docker/dev/
        exit 0
    fi
elif [ "$1" = "run" ]; then 
    if docker images | grep -o ubuntu-go; then
        echo "[INFO] Running ubuntu go developement environment"
        docker run -it -p 5001:5001 -v ./src:/go/src ubuntu-go
        exit 0
    else
        echo "[ERROR] Image ubuntu-go not found. Run '$0 build' first"
        exit 1
    fi
else
    echo "Usage: $0 {build|run}"
    exit 1
fi

