# DevOps26_RE_minitwit

This repository contains the Go implementation of the MiniTwit application, designed for the ITU DevOps course. The project is fully containerized for development and automated for cloud deployment on DigitalOcean.
## 🚀 Public Access

The application is deployed and reachable at the following endpoints:

    MiniTwit Application URL: http://164.92.186.201:5001 (Web UI)

    Simulator API Endpoint: http://164.92.186.201:5001 (API root for simulator)

## 🛠 Deployment Guide (Task 2)

We have automated the provisioning of the entire infrastructure (Database and Web servers) using Vagrant and DigitalOcean. Follow these steps to deploy a fresh release from scratch.
1. Prerequisites & Secrets

Before running the deployment script, ensure you have the following secrets configured as environment variables on your host machine:
Variable	Description
DIGITAL_OCEAN_TOKEN	Your DigitalOcean Personal Access Token.
SSH_KEY_NAME	The name of the SSH key already registered in your DigitalOcean account.
Bash

export DIGITAL_OCEAN_TOKEN="your_actual_token_here"
export SSH_KEY_NAME="your_key_name"

2. One-Command Deployment

To create the VMs, configure the network, set up the PostgreSQL database, and deploy the Go application, run:
Bash

# Clone the repository
git clone git@github.com:<your_id>/DevOps26_RE_minitwit.git
cd DevOps26_RE_minitwit

# Provision the cloud infrastructure
vagrant up --provider=digital_ocean

Note on Provisioning: The Vagrantfile is scripted to automatically:

    Spin up a dbserver (PostgreSQL).

    Spin up a webserver (Go App).

    Synchronize the source code and compile the binary.

    Handle process management (automatically kills old versions and restarts).

## 💻 Local Development

For local development, we provide a pre-configured Docker environment to ensure toolchain consistency.
Development Container

The container runs Ubuntu 24.04. You can customize the Go version using --build-args GO_VERSION=.

Quick Start with develop.sh:

    ./develop.sh build: Builds the development Docker image.

    ./develop.sh run: Runs the container and drops you into an interactive terminal.

## Inside the Container

    /go/src: Contains the synchronized source code.

    /go/src/bin: Contains the compiled binary.

    Command gobuild: A built-in alias to compile the project immediately to the /bin folder.

## 📂 Project Structure

    /src: Go source code and application logic.

    /src/bin: Compiled binaries and schema.sql.

    /docker: Dockerfiles for development and production.

    Vagrantfile: Infrastructure as Code (IaC) for cloud deployment.