// SPDX-License-Identifier: EUPL-1.2

package mcp

import (
	"context"
	"crypto/subtle"
	"net"
	"net/http"
	"os"
	"time"

	coreerr "forge.lthn.ai/core/go-log"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// DefaultHTTPAddr is the default address for the MCP HTTP server.
const DefaultHTTPAddr = "127.0.0.1:9101"

// ServeHTTP starts the MCP server with Streamable HTTP transport.
// Supports Bearer token authentication via MCP_AUTH_TOKEN env var.
// If no token is set, authentication is disabled (local development mode).
//
// The server exposes a single endpoint at /mcp that handles:
//   - GET:    Open SSE stream for server-to-client notifications
//   - POST:   Send JSON-RPC messages (tool calls, etc.)
//   - DELETE: Terminate session
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
// If token is empty, authentication is disabled (passthrough).
func withAuth(token string, next http.Handler) http.Handler {
	if token == "" {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if len(auth) < 7 || auth[:7] != "Bearer " {
			http.Error(w, `{"error":"missing Bearer token"}`, http.StatusUnauthorized)
			return
		}
		provided := auth[7:]
		if subtle.ConstantTimeCompare([]byte(provided), []byte(token)) != 1 {
			http.Error(w, `{"error":"invalid token"}`, http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}
