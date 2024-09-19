package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"flag"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/coreos/go-oidc"
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

	server := http.Server{
		Addr:    ":" + *port,
		Handler: authMiddleware(mux),
	}

	// Start HTTP server (and proxy calls to gRPC server endpoint)
	log.Info().Msgf("Running HTTP server on %s", *port)
	return server.ListenAndServe()
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

type logResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rsp *logResponseWriter) WriteHeader(code int) {
	rsp.statusCode = code
	rsp.ResponseWriter.WriteHeader(code)
}

// Unwrap returns the original http.ResponseWriter. This is necessary
// to expose Flush() and Push() on the underlying response writer.
func (rsp *logResponseWriter) Unwrap() http.ResponseWriter {
	return rsp.ResponseWriter
}

func newLogResponseWriter(w http.ResponseWriter) *logResponseWriter {
	return &logResponseWriter{w, http.StatusOK}
}

// logRequestBody logs the request body when the response status code is not 200.
func logRequestBody(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lw := newLogResponseWriter(w)

		// Note that buffering the entire request body could consume a lot of memory.
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to read body: %v", err), http.StatusBadRequest)
			return
		}
		clonedR := r.Clone(r.Context())
		clonedR.Body = io.NopCloser(bytes.NewReader(body))

		h.ServeHTTP(lw, clonedR)

		if lw.statusCode != http.StatusOK {
			grpclog.Errorf("http error %+v request body %+v", lw.statusCode, string(body))
		}
	})
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

func authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, err := getBearerToken(r)
		if err != nil {
			log.Error().Err(err).Msg("failed to get bearer token")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		user, err := authorize(r.Context(), token)
		if err != nil {
			log.Error().Err(err).Msg("failed to authorize user")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// only allow if the user is in the "admin" group
		if user.email != "admin" {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}

type user struct {
	email  string
	groups []string
}

// authorize verifies a bearer token and pulls user information form the claims.
func authorize(ctx context.Context, bearerToken string) (*user, error) {
	// Create a remote key set that fetches keys from the provider's jwks_uri.
	// This is set manually because the issuer URL returned from the discovery endpoints is not a URL that can be resolved from the pod
	keySet := oidc.NewRemoteKeySet(ctx, "http://authentication-dex:5556/dex/keys")

	// Create an ID token parser that only trust ID tokens issued to "mke-dashboard"
	// The SkipIssuerCheck option is used to skip the issuer check because Dex's issuer URL is not a URL that can be resolved from the pod
	idTokenVerifier := oidc.NewVerifier("http://authentication-dex:5556/dex", keySet, &oidc.Config{ClientID: "mke-dashboard", SkipIssuerCheck: true})

	idToken, err := idTokenVerifier.Verify(ctx, bearerToken)
	if err != nil {
		return nil, fmt.Errorf("could not verify bearer token: %v", err)
	}
	// Extract custom claims.
	var claims struct {
		Email    string   `json:"email"`
		Verified bool     `json:"email_verified"`
		Groups   []string `json:"groups"`
	}
	if err := idToken.Claims(&claims); err != nil {
		return nil, fmt.Errorf("failed to parse claims: %v", err)
	}
	if !claims.Verified {
		return nil, fmt.Errorf("email (%q) in returned claims was not verified", claims.Email)
	}
	return &user{claims.Email, claims.Groups}, nil
}

func getBearerToken(r *http.Request) (string, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return "", fmt.Errorf("authorization header not found")
	}

	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		return "", fmt.Errorf("invalid authorization header format")
	}

	return parts[1], nil
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
