# DevOps26_RE_minitwit

This repository contains a Go implementation of the MiniTwit application, designed for the ITU DevOps course. The project features a containerized development environment and automated cloud deployment on DigitalOcean.

## 🚀 Public Access

The application is deployed and reachable at the following endpoints:

| Service | URL |
| :--- | :--- |
| **MiniTwit Web UI** | [http://164.92.186.201:5001](http://164.92.186.201:5001) |
| **Simulator API** | [http://164.92.186.201:5001/api](http://164.92.186.201:5001/api) |

---

## 🛠 Deployment Guide

Infrastructure provisioning and application deployment are automated using **Vagrant** and the **DigitalOcean** provider.

### 1. Prerequisites & Secrets
Before running the deployment, ensure the following environment variables are configured on your host machine:

| Variable | Description |
| :--- | :--- |
| `DIGITAL_OCEAN_TOKEN` | Your DigitalOcean Personal Access Token. |
| `SSH_KEY_NAME` | The name of the SSH key registered in your DigitalOcean account. |

```bash
export DIGITAL_OCEAN_TOKEN="your_actual_token_here"
export SSH_KEY_NAME="your_key_name"
```

### 2. One-Command Cloud Deployment
To provision the VMs (Database and Web servers) and deploy the latest release:

```bash
# Clone the repository
git clone git@github.com:<your_id>/DevOps26_RE_minitwit.git
cd DevOps26_RE_minitwit

# Provision the cloud infrastructure
vagrant up --provider=digital_ocean
```

> **Note**: If you modify the Go application code later, simply run `vagrant provision webserver` to automatically recompile, kill the old process, and restart the service.

---

## 🧪 Testing & Troubleshooting

### Run Simulator API Tests
Test the compatibility of your API with the provided Python simulator:
```bash
python3 minitwit_simulator.py "http://164.92.186.201:5001/api"
```

### Monitor Webserver Logs
To view real-time application logs (including `fmt.Printf` debug messages and Gin logs), SSH into the webserver:
```bash
vagrant ssh webserver

# View the last few lines and follow real-time output
tail -f /vagrant/src/bin/minitwit.log

# View the entire log history
cat /vagrant/src/bin/minitwit.log
```

---

## 💻 Local Development

We provide a pre-configured Docker environment using **Ubuntu 24.04** to ensure toolchain consistency.

### Development Commands
* `./develop.sh build`: Builds the development Docker image.
* `./develop.sh run`: Runs the container and opens an interactive terminal.

### Inside the Container
* `/go/src`: Synced source code directory.
* `/go/src/bin`: Destination for compiled binaries.
* **`gobuild`**: A built-in alias to compile the project immediately to the `/bin` folder.

---

## 📂 Project Structure

* `/src`: Go source code and core application logic.
* `/src/bin`: Compiled binaries and `schema.sql`.
* `/docker`: Dockerfiles for development and production.
* `Vagrantfile`: Infrastructure as Code (IaC) configuration.
