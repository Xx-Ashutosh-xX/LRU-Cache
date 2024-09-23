# Dockerfile for Golang Backend
FROM golang:1.20-alpine

WORKDIR /app

COPY . .

RUN go mod download
RUN go build -o lru-cache .

EXPOSE 8080

CMD ["./lru-cache"]
