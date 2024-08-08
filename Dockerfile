FROM golang:1.23rc2-alpine3.20

WORKDIR /app
COPY . /app

RUN go build ./cmd/dex-http-server/main.go

