# DevOps26_RE_minitwit

This repository contains a Go implementation of the MiniTwit application, designed for the ITU DevOps course. The project features a containerized development environment and automated cloud deployment on DigitalOcean.

## 🚀 Public Access

The application is deployed and reachable at the following endpoints:

| Service | URL |
| :--- | :--- |
| **MiniTwit Web UI** | [http://164.92.186.201:5001](http://164.92.186.201:5001) |
| **Simulator API** | [http://164.92.186.201:5001/api](http://164.92.186.201:5001/api) |

---

## 💻 Local Development Quickstart

To get the project running locally for the first time, follow these steps to set up your environment:

### 1. Initialize Local Files
We use template files to prevent local configuration or binary data from cluttering the version control.

```bash
# 1. Create your local working database from the template
cp tmp/minitwit.db.example tmp/minitwit.db

# 2. Create the DB IP configuration file at root 
# (Set to 127.0.0.1 for local SQLite or your remote DB server IP)
echo "127.0.0.1" > db_ip.txt
```

### 2. Run with Docker (Recommended)
Following modern DevOps practices, we recommend using a single Dockerfile with **multi-stage builds** to handle both development and production.

* **Build image**: `./develop.sh build`
* **Enter Container**: `./develop.sh run`
* **Inside the container**: The project root is synced to the container's workspace. You can run `go run main.go` directly.

### 3. Using Makefile (Shortcuts)
You can use the following commands for quick task execution:
* `make run`: Starts the Go application locally.
* `make build`: Compiles the Go binary.
* `make test-sim`: Runs the Python simulator against your local instance.

---

## � Docker Architecture

The project uses a **decoupled Docker setup** with three separate Dockerfiles and dedicated Docker Compose configurations:

### Dockerfiles

| File | Purpose | Used By |
| :--- | :--- | :--- |
| **Dockerfile-app** | Production Go application container | docker-compose-app.yaml (webserver) |
| **Dockerfile-db** | PostgreSQL database container | docker-compose-db.yaml (dbserver) |
| **Dockerfile-test** | Testing environment (Go + Python) | docker-compose-tests.yaml (local testing) |

### Docker Compose Files

- **docker-compose-db.yaml**: Database server only (persistent volume for data)
- **docker-compose-app.yaml**: Application server with remote database connectivity
- **docker-compose-tests.yaml**: Full stack (db + app + test runner) for local development

### Local Testing (All-in-One)

For local development, run the complete stack:
```bash
docker-compose -f docker-compose-tests.yaml up --build
```

This starts:
- PostgreSQL database container
- Go application container
- Python test simulator container

---

## �🛠 Cloud Deployment (DigitalOcean)

Infrastructure provisioning and application deployment are automated using **Vagrant** with the DigitalOcean provider.

### 1. Prerequisites
Ensure the following environment variables are configured on your host machine:

| Variable | Description |
| :--- | :--- |
| `DIGITAL_OCEAN_TOKEN` | Your DigitalOcean Personal Access Token. |
| `SSH_KEY_NAME` | The name of the SSH key registered in your DigitalOcean account. |

```bash
export DIGITAL_OCEAN_TOKEN="your_actual_token_here"
export SSH_KEY_NAME="your_key_name"
```

### 2. Provisioning
To provision the Database and Web servers and deploy the latest code:
```bash
vagrant up --provider=digital_ocean
```

---

## 🧪 Testing & Troubleshooting

### Run Simulator API Tests
Test your API compatibility using the provided Python simulator:
```bash
# Ensure you are at the project root
python3 test/minitwit_simulator.py "http://localhost:5001/api"
python3 test/minitwit_simulator.py "http://164.92.186.201:5001/api"
```

### Monitor Webserver Logs
To view real-time application logs, SSH into the webserver:
```bash
vagrant ssh webserver
tail -f /var/log/minitwit.log
```

---

## 📂 Project Structure

```text
.
├── db/              # Database schema and initialization scripts (schema.sql is here)
├── docker/          # Dockerfiles (Multi-stage build strategy)
├── static/          # Static assets (CSS, Images, JS)
├── templates/       # HTML templates for the Gin framework
├── test/            # Python simulator and test scenario CSVs
├── tmp/             # Local DB templates (Real DB and legacy folder are ignored)
├── simulator_api.go          # Application entry point
├── minitwit.go          # Application entry point
├── Makefile         # Shortcuts for common tasks
└── Vagrantfile      # Infrastructure as Code (IaC) configuration
└── develop.sh    # for local developemnt
```