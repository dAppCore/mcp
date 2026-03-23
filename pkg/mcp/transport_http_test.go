// SPDX-License-Identifier: EUPL-1.2

package mcp

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"testing"
	"time"
)

func TestServeHTTP_Good_HealthEndpoint(t *testing.T) {
	s, err := New(Options{})
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Get a free port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to find free port: %v", err)
	}
	addr := listener.Addr().String()
	listener.Close()

	errCh := make(chan error, 1)
	go func() {
		errCh <- s.ServeHTTP(ctx, addr)
	}()

	// Wait for server to start
	time.Sleep(100 * time.Millisecond)

	resp, err := http.Get(fmt.Sprintf("http://%s/health", addr))
	if err != nil {
		t.Fatalf("health check failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	cancel()
	<-errCh
}

func TestServeHTTP_Good_DefaultAddr(t *testing.T) {
	if DefaultHTTPAddr != "127.0.0.1:9101" {
		t.Errorf("expected default HTTP addr 127.0.0.1:9101, got %s", DefaultHTTPAddr)
	}
}

func TestServeHTTP_Good_AuthRequired(t *testing.T) {
	os.Setenv("MCP_AUTH_TOKEN", "test-secret-token")
	defer os.Unsetenv("MCP_AUTH_TOKEN")

	s, err := New(Options{})
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to find free port: %v", err)
	}
	addr := listener.Addr().String()
	listener.Close()

	errCh := make(chan error, 1)
	go func() {
		errCh <- s.ServeHTTP(ctx, addr)
	}()

	time.Sleep(100 * time.Millisecond)

	// Request without token should be rejected
	resp, err := http.Get(fmt.Sprintf("http://%s/mcp", addr))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != 401 {
		t.Errorf("expected 401 without token, got %d", resp.StatusCode)
	}

	// Health endpoint should still work (no auth)
	resp, err = http.Get(fmt.Sprintf("http://%s/health", addr))
	if err != nil {
		t.Fatalf("health check failed: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Errorf("expected 200 for health, got %d", resp.StatusCode)
	}

	cancel()
	<-errCh
}

func TestWithAuth_Good_ValidToken(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	})

	wrapped := withAuth("my-token", handler)

	// Valid token
	req, _ := http.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer my-token")
	rr := &fakeResponseWriter{code: 200}
	wrapped.ServeHTTP(rr, req)
	if rr.code != 200 {
		t.Errorf("expected 200 with valid token, got %d", rr.code)
	}
}

func TestWithAuth_Bad_InvalidToken(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	})

	wrapped := withAuth("my-token", handler)

	// Wrong token
	req, _ := http.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer wrong-token")
	rr := &fakeResponseWriter{code: 200}
	wrapped.ServeHTTP(rr, req)
	if rr.code != 401 {
		t.Errorf("expected 401 with wrong token, got %d", rr.code)
	}
}

func TestWithAuth_Bad_MissingToken(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	})

	wrapped := withAuth("my-token", handler)

	// No Authorization header
	req, _ := http.NewRequest("GET", "/", nil)
	rr := &fakeResponseWriter{code: 200}
	wrapped.ServeHTTP(rr, req)
	if rr.code != 401 {
		t.Errorf("expected 401 with missing token, got %d", rr.code)
	}
}

func TestWithAuth_Bad_EmptyConfiguredToken(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	})

	// Empty token now requires explicit configuration
	wrapped := withAuth("", handler)

	req, _ := http.NewRequest("GET", "/", nil)
	rr := &fakeResponseWriter{code: 200}
	wrapped.ServeHTTP(rr, req)
	if rr.code != 401 {
		t.Errorf("expected 401 with empty configured token, got %d", rr.code)
	}
}

func TestWithAuth_Bad_NonBearerToken(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	})

	wrapped := withAuth("my-token", handler)

	req, _ := http.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Token my-token")
	rr := &fakeResponseWriter{code: 200}
	wrapped.ServeHTTP(rr, req)
	if rr.code != 401 {
		t.Errorf("expected 401 with non-Bearer auth, got %d", rr.code)
	}
}

func TestRun_Good_HTTPTrigger(t *testing.T) {
	s, err := New(Options{})
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Find a free port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to find free port: %v", err)
	}
	addr := listener.Addr().String()
	listener.Close()

	// MCP_HTTP_ADDR takes priority over MCP_ADDR
	os.Setenv("MCP_HTTP_ADDR", addr)
	os.Setenv("MCP_ADDR", "")
	defer os.Unsetenv("MCP_HTTP_ADDR")
	defer os.Unsetenv("MCP_ADDR")

	errCh := make(chan error, 1)
	go func() {
		errCh <- s.Run(ctx)
	}()

	time.Sleep(100 * time.Millisecond)

	// Verify server is running
	resp, err := http.Get(fmt.Sprintf("http://%s/health", addr))
	if err != nil {
		t.Fatalf("health check failed: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	cancel()
	<-errCh
}

// fakeResponseWriter is a minimal http.ResponseWriter for unit testing withAuth.
type fakeResponseWriter struct {
	code int
	hdr  http.Header
}

func (f *fakeResponseWriter) Header() http.Header {
	if f.hdr == nil {
		f.hdr = make(http.Header)
	}
	return f.hdr
}

func (f *fakeResponseWriter) Write(b []byte) (int, error) { return len(b), nil }
func (f *fakeResponseWriter) WriteHeader(code int)        { f.code = code }
