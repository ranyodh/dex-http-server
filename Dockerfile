FROM golang:1.23rc2-alpine3.20

WORKDIR /app
COPY . /app

RUN apk add make git

RUN make build

ENTRYPOINT ["/app/bin/dex-http-server"]
