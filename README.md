# MiniTwit — Go/Gin Rewrite
A rewrite of the original Python/Flask MiniTwit application using Go and Gin.

## Tech Stack

Language: Go
Web Framework: Gin
Database: SQLite (via go-sqlite3)
Session Management: gin-contrib/sessions
Password Hashing: bcrypt
Containerization: Docker

DevOps26_RE_minitwit/
├── minitwit.go        # Main application file
├── schema.sql         # Database schema
├── Dockerfile         # Docker build instructions
├── docker-compose.yml # Docker Compose configuration
├── .dockerignore      # Files excluded from Docker build
├── templates/         # HTML templates
│   ├── layout.html
│   ├── timeline.html
│   ├── login.html
│   └── register.html
├── static/            # Static files
│   └── style.css
└── tmp/               # SQLite database (auto-created)
    └── minitwit.db

## Run without Docker
go run minitwit.go

## Run with Docker Compose
docker compose up
The app will be available at http://localhost:8080.