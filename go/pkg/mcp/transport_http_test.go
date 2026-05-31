// SPDX-License-Identifier: EUPL-1.2

package mcp

import (
	"context"
	"net"
	"net/http"
	"testing"
	"time"

	core "dappco.re/go"
)

func TestServeHTTP_Good_HealthEndpoint(t *testing.T) {
	t.Setenv("MCP_AUTH_TOKEN", "test-secret-token")
	t.Setenv("MCP_JWT_SECRET", "distinct-signing-key")

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

	resp, err := http.Get(core.Sprintf("http://%s/health", addr))
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
	t.Setenv("MCP_AUTH_TOKEN", "test-secret-token")
	t.Setenv("MCP_JWT_SECRET", "distinct-signing-key")

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
	resp, err := http.Get(core.Sprintf("http://%s/mcp", addr))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != 401 {
		t.Errorf("expected 401 without token, got %d", resp.StatusCode)
	}

	// Health endpoint should still work (no auth)
	resp, err = http.Get(core.Sprintf("http://%s/health", addr))
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

// TestServeHTTP_Bad_NoAuthTokenFailsClosed asserts S1.1: the served transport
// refuses to bind a listener when MCP_AUTH_TOKEN is absent. There is no
// unauthenticated open-listener mode — ServeHTTP returns an error and never
// opens a socket.
func TestServeHTTP_Bad_NoAuthTokenFailsClosed(t *testing.T) {
	t.Setenv("MCP_AUTH_TOKEN", "")
	t.Setenv("MCP_JWT_SECRET", "distinct-signing-key")

	s, err := New(Options{})
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to find free port: %v", err)
	}
	addr := listener.Addr().String()
	listener.Close()

	if err := s.ServeHTTP(context.Background(), addr); err == nil {
		t.Fatal("expected ServeHTTP to refuse to serve without MCP_AUTH_TOKEN (fail-closed), got nil error")
	}

	// The listener must never have opened — the port is free to rebind.
	probe, err := net.Listen("tcp", addr)
	if err != nil {
		t.Fatalf("expected addr %s free after fail-closed refusal, got: %v", addr, err)
	}
	probe.Close()
}

// TestServeHTTP_Bad_NoJWTSecretFailsClosed asserts S1.3: even with a bearer
// token configured, the served transport refuses to bind without a distinct
// MCP_JWT_SECRET. There is no API-token fallback for the signing key.
func TestServeHTTP_Bad_NoJWTSecretFailsClosed(t *testing.T) {
	t.Setenv("MCP_AUTH_TOKEN", "test-secret-token")
	t.Setenv("MCP_JWT_SECRET", "")

	s, err := New(Options{})
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to find free port: %v", err)
	}
	addr := listener.Addr().String()
	listener.Close()

	if err := s.ServeHTTP(context.Background(), addr); err == nil {
		t.Fatal("expected ServeHTTP to refuse to serve without MCP_JWT_SECRET (fail-closed), got nil error")
	}
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

func TestWithAuth_Good_EmptyConfiguredToken_DisablesAuth(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	})

	wrapped := withAuth("", handler)

	req, _ := http.NewRequest("GET", "/", nil)
	rr := &fakeResponseWriter{code: 200}
	wrapped.ServeHTTP(rr, req)
	if rr.code != 200 {
		t.Errorf("expected 200 with empty configured token, got %d", rr.code)
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
	t.Setenv("MCP_HTTP_ADDR", addr)
	t.Setenv("MCP_ADDR", "")
	t.Setenv("MCP_AUTH_TOKEN", "test-secret-token")
	t.Setenv("MCP_JWT_SECRET", "distinct-signing-key")

	errCh := make(chan error, 1)
	go func() {
		errCh <- s.Run(ctx)
	}()

	time.Sleep(100 * time.Millisecond)

	// Verify server is running
	resp, err := http.Get(core.Sprintf("http://%s/health", addr))
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

// moved AX-7 triplet TestTransportHttp_Service_ServeHTTP_Good
func TestTransportHttp_Service_ServeHTTP_Good(t *T) {
	t.Setenv("MCP_AUTH_TOKEN", "test-secret-token")
	t.Setenv("MCP_JWT_SECRET", "distinct-signing-key")
	svc := newServiceForTest(t, Options{})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := svc.ServeHTTP(ctx, "127.0.0.1:0")
	AssertNoError(t, err)
}

// moved AX-7 triplet TestTransportHttp_Service_ServeHTTP_Bad
func TestTransportHttp_Service_ServeHTTP_Bad(t *T) {
	t.Setenv("MCP_AUTH_TOKEN", "test-secret-token")
	t.Setenv("MCP_JWT_SECRET", "distinct-signing-key")
	svc := newServiceForTest(t, Options{})
	err := svc.ServeHTTP(context.Background(), "127.0.0.1:bad")
	AssertError(t, err)
}

// moved AX-7 triplet TestTransportHttp_Service_ServeHTTP_Ugly
func TestTransportHttp_Service_ServeHTTP_Ugly(t *T) {
	t.Setenv("MCP_AUTH_TOKEN", "test-secret-token")
	t.Setenv("MCP_JWT_SECRET", "distinct-signing-key")
	svc := newServiceForTest(t, Options{})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := svc.ServeHTTP(ctx, "")
	AssertNoError(t, err)
}
