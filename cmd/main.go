package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"flag"
	"fmt"
	"net/http"
	"strings"

	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"nwneisen/dex-http-server/gen/go/api"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/grpclog"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

var (
	// command-line options:
	// gRPC server endpoint
	grpcServerEndpoint = flag.String("grpc-server", "dex:5557", "gRPC server endpoint")
	// HTTP server port
	port              = flag.String("http-port", "8080", "HTTP server port")
	dexGrpcCertSecret = flag.String("dex-grpc-cert-secret", "", "Namepsace/name for the Dex gRPC cert secret name. e.g 'mke/dex-grpc.tls'")

	version, commit, date = "", "", "" // These are always injected at build time
)

func run() error {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var err error

	var clientTLSConfig credentials.TransportCredentials
	if *dexGrpcCertSecret == "" {
		log.Debug().Msg("No Dex gRPC cert secret name provided, using insecure connection")
		clientTLSConfig = insecure.NewCredentials()
	} else {
		log.Info().Msgf("Using Dex gRPC cert from secret: %s", *dexGrpcCertSecret)
		clientTLSConfig, err = getDexGrpcCredentials(ctx, *dexGrpcCertSecret)
	}

	// Register gRPC server endpoint
	// Note: Make sure the gRPC server is running properly and accessible
	mux := runtime.NewServeMux()
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(clientTLSConfig),
	}

	if err = api.RegisterDexHandlerFromEndpoint(ctx, mux, *grpcServerEndpoint, opts); err != nil {
		return fmt.Errorf("failed to register gRPC server endpoint: %w", err)
	}

	log.Info().Msgf("Registered gRPC server endpoint: %s", *grpcServerEndpoint)

	// Start HTTP server (and proxy calls to gRPC server endpoint)
	log.Info().Msgf("Running HTTP server on %s", *port)
	return http.ListenAndServe(":"+*port, mux)
}

func getDexGrpcCredentials(ctx context.Context, secretName string) (credentials.TransportCredentials, error) {
	namespace, name, err := splitNamespaceName(secretName)
	if err != nil {
		return nil, fmt.Errorf("failed to parse secret name: %w", err)
	}

	// Create k8s client
	log.Info().Msg("Creating k8s client")
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get k8s config: %w", err)
	}
	// creates the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create k8s clientset: %w", err)
	}

	log.Info().Msg("Retrieving gRPC secret")
	secret, err := clientset.CoreV1().Secrets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get secret: %w", err)
	}

	cPool := x509.NewCertPool()
	if !cPool.AppendCertsFromPEM(secret.Data["ca.crt"]) {
		return nil, fmt.Errorf("unable to parse CA crt from secret %s", secretName)
	}

	clientCert, err := tls.X509KeyPair(secret.Data["tls.crt"], secret.Data["tls.key"])
	if err != nil {
		return nil, fmt.Errorf("invalid client crt data from ")
	}

	clientTLSConfig := &tls.Config{
		RootCAs:      cPool,
		Certificates: []tls.Certificate{clientCert},
	}

	return credentials.NewTLS(clientTLSConfig), nil
}

func splitNamespaceName(s string) (string, string, error) {
	parts := strings.Split(s, "/")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid namespace/name format: %s", s)
	}
	return parts[0], parts[1], nil
}

func main() {
	flag.Parse()

	log.Info().Msg("Starting dex-http-server")
	log.Info().Msgf("Version: %s", version)
	log.Info().Msgf("Commit: %s", commit)
	log.Info().Msgf("Date: %s", date)

	if err := run(); err != nil {
		grpclog.Fatal(err)
	}
}
