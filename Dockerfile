FROM golang:1.25-bookworm AS builder

WORKDIR /app


RUN apt-get update && apt-get install -y gcc libc6-dev && rm -rf /var/lib/apt/lists/*

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=1 GOOS=linux go build -o minitwit minitwit.go

FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y libc6 ca-certificates && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY --from=builder /app/minitwit .
COPY --from=builder /app/templates/ ./templates/
COPY --from=builder /app/static/ ./static/
COPY --from=builder /app/schema.sql .

EXPOSE 8080

CMD ["./minitwit"]