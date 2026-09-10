package handlers

import (
	"fmt"
	"net"
	"net/http"
	"strings"

	"github.com/niickoh/api-notificaciones-go/services"
)

// Middleware representa una función middleware HTTP.
type Middleware func(http.Handler) http.Handler

// Chain aplica middlewares en orden.
func Chain(handler http.Handler, middlewares ...Middleware) http.Handler {
	for i := len(middlewares) - 1; i >= 0; i-- {
		handler = middlewares[i](handler)
	}
	return handler
}

// RecoverMiddleware evita crashes y fuerza respuestas JSON controladas.
func RecoverMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if recovered := recover(); recovered != nil {
				WriteError(w, http.StatusInternalServerError, "Error interno del servidor")
			}
		}()

		next.ServeHTTP(w, r)
	})
}

// CORSMiddleware habilita CORS para la API.
func CORSMiddleware(origin string) Middleware {
	if origin == "" {
		origin = "*"
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-API-Key")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// AuthMiddleware valida JWT en todos los endpoints protegidos.
func AuthMiddleware(jwtService *services.JWTService) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if err := jwtService.ValidateBearerHeader(r.Header.Get("Authorization")); err != nil {
				WriteError(w, http.StatusUnauthorized, "JWT inválido")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RateLimitMiddleware aplica límites por IP y API key/token.
func RateLimitMiddleware(ipLimiter, apiKeyLimiter *services.FixedWindowLimiter) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !ipLimiter.Allow(clientIP(r)) {
				WriteError(w, http.StatusTooManyRequests, "Rate limit excedido")
				return
			}

			apiKey := strings.TrimSpace(r.Header.Get("X-API-Key"))
			if apiKey == "" {
				apiKey = strings.TrimSpace(r.Header.Get("Authorization"))
			}
			if apiKey != "" && !apiKeyLimiter.Allow(apiKey) {
				WriteError(w, http.StatusTooManyRequests, "Rate limit excedido")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func clientIP(r *http.Request) string {
	if forwarded := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); forwarded != "" {
		parts := strings.Split(forwarded, ",")
		return strings.TrimSpace(parts[0])
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}

	if r.RemoteAddr != "" {
		return r.RemoteAddr
	}

	return fmt.Sprintf("unknown-%p", r)
}
