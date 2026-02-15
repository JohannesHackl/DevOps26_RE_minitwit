# Stage 1: Build the Go binary
FROM golang:1.26 AS builder

WORKDIR /app

# Copy dependency files first (better caching)
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source code
COPY . .

# Build the binary (CGO enabled for sqlite3)
RUN CGO_ENABLED=1 go build -o minitwit main.go

# Stage 2: Run the app in a small image
FROM debian:bookworm-slim

WORKDIR /app

# Install sqlite3 runtime dependency
RUN apt-get update && apt-get install -y libsqlite3-0 ca-certificates && rm -rf /var/lib/apt/lists/*

# Copy the binary and assets from the builder stage
COPY --from=builder /app/minitwit .
COPY --from=builder /app/templates ./templates
COPY --from=builder /app/static ./static
COPY --from=builder /app/schema.sql ./schema.sql

# Expose the port
EXPOSE 5000

# Run the app
CMD ["./minitwit"]
