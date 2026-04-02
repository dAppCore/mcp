// SPDX-License-Identifier: EUPL-1.2

package mcp

import (
	"context"
	"crypto/subtle"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	coreerr "forge.lthn.ai/core/go-log"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// DefaultHTTPAddr is the default address for the MCP HTTP server.
//
//	svc.ServeHTTP(ctx, DefaultHTTPAddr) // "127.0.0.1:9101"
const DefaultHTTPAddr = "127.0.0.1:9101"

// ServeHTTP starts the MCP server with Streamable HTTP transport.
// Supports Bearer token authentication via MCP_AUTH_TOKEN env var.
// If no token is set, authentication is disabled (local development mode).
//
//	// Local development (no auth):
//	svc.ServeHTTP(ctx, "127.0.0.1:9101")
//
//	// Production (with auth):
//	os.Setenv("MCP_AUTH_TOKEN", "sk-abc123")
//	svc.ServeHTTP(ctx, "0.0.0.0:9101")
//
// Endpoint /mcp: GET (SSE stream), POST (JSON-RPC), DELETE (terminate session).
func (s *Service) ServeHTTP(ctx context.Context, addr string) error {
	if addr == "" {
		addr = DefaultHTTPAddr
	}

	authToken := os.Getenv("MCP_AUTH_TOKEN")

	handler := mcp.NewStreamableHTTPHandler(
		func(r *http.Request) *mcp.Server {
			return s.server
		},
		&mcp.StreamableHTTPOptions{
			SessionTimeout: 30 * time.Minute,
		},
	)

	mux := http.NewServeMux()
	mux.Handle("/mcp", withAuth(authToken, handler))

	// Health check (no auth)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return coreerr.E("mcp.ServeHTTP", "failed to listen on "+addr, err)
	}
	defer listener.Close()

	diagPrintf("MCP HTTP server listening on %s\n", addr)

	server := &http.Server{Handler: mux}

	// Graceful shutdown on context cancellation
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		server.Shutdown(shutdownCtx)
	}()

	if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
		return coreerr.E("mcp.ServeHTTP", "server error", err)
	}
	return nil
}

// withAuth wraps an http.Handler with Bearer token authentication.
// If token is empty, authentication is disabled for local development.
func withAuth(token string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.TrimSpace(token) == "" {
			next.ServeHTTP(w, r)
			return
		}

		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
			http.Error(w, `{"error":"missing Bearer token"}`, http.StatusUnauthorized)
			return
		}

		provided := strings.TrimSpace(strings.TrimPrefix(auth, "Bearer "))
		if len(provided) == 0 {
			http.Error(w, `{"error":"missing Bearer token"}`, http.StatusUnauthorized)
			return
		}

		if subtle.ConstantTimeCompare([]byte(provided), []byte(token)) != 1 {
			http.Error(w, `{"error":"invalid token"}`, http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}
