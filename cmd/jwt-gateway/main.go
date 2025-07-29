package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/shogotsuneto/dapr-actor-experiment/internal/auth"
)

type JWTProxy struct {
	jwtMiddleware *auth.JWTMiddleware
	daprProxy     *httputil.ReverseProxy
	daprTarget    *url.URL
}

func NewJWTProxy(daprEndpoint string, jwtMiddleware *auth.JWTMiddleware) (*JWTProxy, error) {
	target, err := url.Parse(daprEndpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Dapr endpoint: %w", err)
	}

	proxy := httputil.NewSingleHostReverseProxy(target)
	
	// Customize the proxy to handle errors and add logging
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		log.Printf("Proxy error: %v", err)
		http.Error(w, "Bad Gateway", http.StatusBadGateway)
	}

	return &JWTProxy{
		jwtMiddleware: jwtMiddleware,
		daprProxy:     proxy,
		daprTarget:    target,
	}, nil
}

func (p *JWTProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Health check endpoint - no JWT required
	if r.URL.Path == "/v1.0/healthz" || r.URL.Path == "/healthz" {
		p.daprProxy.ServeHTTP(w, r)
		return
	}

	// Non-actor endpoints - no JWT required
	if !strings.Contains(r.URL.Path, "/v1.0/actors/") {
		p.daprProxy.ServeHTTP(w, r)
		return
	}

	// Extract JWT token from Authorization header
	authHeader := r.Header.Get("Authorization")
	tokenString, err := auth.ExtractTokenFromAuthHeader(authHeader)
	if err != nil {
		log.Printf("JWT validation failed: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{
			"error": fmt.Sprintf("Unauthorized: %s", err.Error()),
		})
		return
	}

	// Validate JWT token
	claims, err := p.jwtMiddleware.ValidateTokenString(tokenString)
	if err != nil {
		log.Printf("JWT validation failed: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{
			"error": fmt.Sprintf("Unauthorized: %s", err.Error()),
		})
		return
	}

	// Add JWT information to headers for downstream services
	if sub, ok := claims["sub"]; ok {
		r.Header.Set("X-JWT-Subject", fmt.Sprintf("%v", sub))
	}
	if role, ok := claims["role"]; ok {
		r.Header.Set("X-JWT-Role", fmt.Sprintf("%v", role))
	}

	log.Printf("JWT validation successful for %s %s, subject: %v", r.Method, r.URL.Path, claims["sub"])

	// Forward the request to Dapr sidecar
	p.daprProxy.ServeHTTP(w, r)
}

func main() {
	// Configuration from environment variables
	proxyPort := os.Getenv("PROXY_PORT")
	if proxyPort == "" {
		proxyPort = "3500" // Default Dapr HTTP port
	}

	daprEndpoint := os.Getenv("DAPR_ENDPOINT")
	if daprEndpoint == "" {
		daprEndpoint = "http://localhost:3501" // Internal Dapr sidecar
	}

	jwksUrl := os.Getenv("JWKS_URL")
	if jwksUrl == "" {
		jwksUrl = "http://jwks-server:3000/.well-known/jwks.json"
	}

	jwtIssuer := os.Getenv("JWT_ISSUER")
	if jwtIssuer == "" {
		jwtIssuer = "http://localhost:3000"
	}

	jwtAudience := os.Getenv("JWT_AUDIENCE")
	if jwtAudience == "" {
		jwtAudience = "dev-api"
	}

	// Initialize JWT middleware
	jwtConfig := &auth.JWTConfig{
		JWKSUrl:  jwksUrl,
		Issuer:   jwtIssuer,
		Audience: jwtAudience,
	}

	jwtMiddleware, err := auth.NewJWTMiddleware(jwtConfig)
	if err != nil {
		log.Fatalf("Failed to initialize JWT middleware: %v", err)
	}
	defer jwtMiddleware.Close()

	// Create JWT proxy
	proxy, err := NewJWTProxy(daprEndpoint, jwtMiddleware)
	if err != nil {
		log.Fatalf("Failed to create JWT proxy: %v", err)
	}

	// Create HTTP server
	srv := &http.Server{
		Addr:         ":" + proxyPort,
		Handler:      proxy,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	log.Printf("Starting JWT Gateway on port %s", proxyPort)
	log.Printf("Forwarding to Dapr endpoint: %s", daprEndpoint)
	log.Printf("JWKS URL: %s", jwksUrl)
	log.Printf("Expected Issuer: %s", jwtIssuer)
	log.Printf("Expected Audience: %s", jwtAudience)
	log.Println("JWT validation will be applied to /v1.0/actors/* endpoints")

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Failed to start server: %v", err)
	}
}