package middlewares

import (
	"net/http"

	"github.com/rs/zerolog/log"
)

// Middleware is a function that wraps an http.HandlerFunc
type Middleware func(http.HandlerFunc) http.HandlerFunc

// ApplyMiddlewares chains all middleware functions to a single handler
func ApplyMiddlewares(h http.HandlerFunc, authDisabled bool) http.HandlerFunc {
	middlewares := []Middleware{
		loggingMiddleware,
	}

	// If authentication is not disabled, add the authentication middleware
	if !authDisabled {
		middlewares = append(middlewares, authMiddleware()...)
	}

	return chainMiddleware(h, middlewares...)
}

// loggingMiddleware logs the request method and path
func loggingMiddleware(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Info().Msgf("%s %s", r.Method, r.URL.Path)
		h.ServeHTTP(w, r) // call ServeHTTP on the original handler
	}
}

// authMiddleware chains the authentication and authorization middleware functions
func authMiddleware() []Middleware {
	// The list of middleware functions to be applied to the request
	// Note: The order of the middleware functions is important as it
	//       defines the order in which they are applied
	return []Middleware{
		authenticationMiddleware,
		authorizationMiddleware,
	}
}

// chainMiddleware chains multiple middleware functions to a single handler
func chainMiddleware(handler http.HandlerFunc, middlewares ...Middleware) http.HandlerFunc {
	// loop in reverse to preserve middleware order
	for i := len(middlewares) - 1; i >= 0; i-- {
		handler = middlewares[i](handler)
	}
	return handler
}
