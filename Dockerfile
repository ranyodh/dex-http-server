FROM golang:1.23rc2-alpine3.20

WORKDIR /app
COPY . /app

RUN go build -o bin/dex-http-server ./cmd/dex-http-server/main.go

ENTRYPOINT ["/app/bin/dex-http-server"]
