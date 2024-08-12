FROM golang:1.23rc2-alpine3.20

# Set environment variables
ENV HTTP_SERVER_PORT 8080
ENV GRPC_SERVER_ADDRESS dex:5557


WORKDIR /app
COPY . /app

RUN apk add make git

RUN make build

ENTRYPOINT /app/bin/dex-http-server --http-port=${HTTP_SERVER_PORT} --grpc-server=${GRPC_SERVER_ADDRESS}