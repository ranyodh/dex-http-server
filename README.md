## Setup

1. Install the deps

protobuf compiler

```bash
go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.28
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.2
```

buf cli

```bash
curl https://github.com/bufbuild/buf/releases/download/v1.35.1/buf-Linux-x86_64 -o buf
chmod +x buf
sudo mv buf /usr/local/bin
```

2. Generate the code

```bash
buf generate
```

3. Run the server

```bash
go run cmd/dex-http-server/main.go
```
