FROM golang:1.23rc2-alpine3.20

# Set environment variables
ENV HTTP_SERVER_PORT 8080
ENV GRPC_SERVER_ADDRESS dex:5557


WORKDIR /app
COPY . /app

RUN go build -o bin/dex-http-server ./cmd/dex-http-server/main.go

ENTRYPOINT /app/bin/dex-http-server --http-port=${HTTP_SERVER_PORT} --grpc-server=${GRPC_SERVER_ADDRESS}