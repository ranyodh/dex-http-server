FROM golang:1.23 AS builder
ARG TARGETOS
ARG TARGETARCH

WORKDIR /app
COPY . /app
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -a -o dex-http-server cmd/main.go

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=builder /app/dex-http-server .
USER 65532:65532

ENTRYPOINT ["/dex-http-server"]
