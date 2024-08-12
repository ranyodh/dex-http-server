package main

import (
	"context"
	"flag"
	"net/http"
	"nwneisen/dex-http-server/gen/go/api"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/grpclog"
)

var (
	// command-line options:
	// gRPC server endpoint
	grpcServerEndpoint = flag.String("grpc-server", "localhost:5557", "gRPC server endpoint")
	// HTTP server port
	port = flag.String("http-port", "8080", "HTTP server port")
)

func run() error {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Register gRPC server endpoint
	// Note: Make sure the gRPC server is running properly and accessible
	mux := runtime.NewServeMux()
	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	err := api.RegisterDexHandlerFromEndpoint(ctx, mux, *grpcServerEndpoint, opts)
	if err != nil {
		return err
	}
	log.Info().Msgf("Registered gRPC server endpoint: %s", *grpcServerEndpoint)

	// Start HTTP server (and proxy calls to gRPC server endpoint)
	log.Info().Msgf("Starting HTTP server on %s", *port)
	return http.ListenAndServe(":"+*port, mux)
}

func main() {
	flag.Parse()

	if err := run(); err != nil {
		grpclog.Fatal(err)
	}
}
