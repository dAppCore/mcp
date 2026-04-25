// SPDX-License-Identifier: EUPL-1.2

package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	core "dappco.re/go/core"
)

func TestClientRemember_Good_SendsOrgAndAuth(t *testing.T) {
	var gotBody map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/v1/brain/remember" {
			t.Fatalf("expected /v1/brain/remember, got %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Fatalf("expected bearer token, got %q", r.Header.Get("Authorization"))
		}
		gotBody = readRequestBody(t, r)
		writeJSON(t, w, http.StatusOK, map[string]any{"id": "mem-1"})
	}))
	defer server.Close()

	c := New(Options{
		URL:         server.URL,
		Key:         "test-key",
		Org:         "core",
		AgentID:     "codex",
		HTTPClient:  server.Client(),
		MaxAttempts: 1,
	})
	result, err := c.Remember(context.Background(), RememberInput{
		Content: "remember org",
		Type:    "decision",
		Project: "mcp",
	})
	if err != nil {
		t.Fatalf("Remember failed: %v", err)
	}
	if result["id"] != "mem-1" {
		t.Fatalf("expected id mem-1, got %v", result["id"])
	}
	if gotBody["org"] != "core" {
		t.Fatalf("expected org=core, got %v", gotBody["org"])
	}
	if gotBody["project"] != "mcp" {
		t.Fatalf("expected project=mcp, got %v", gotBody["project"])
	}
	if gotBody["agent_id"] != "codex" {
		t.Fatalf("expected agent_id=codex, got %v", gotBody["agent_id"])
	}
}

func TestClientList_Good_SendsOrgURLParam(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/v1/brain/list" {
			t.Fatalf("expected /v1/brain/list, got %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("org"); got != "core" {
			t.Fatalf("expected org=core, got %q", got)
		}
		if got := r.URL.Query().Get("project"); got != "mcp" {
			t.Fatalf("expected project=mcp, got %q", got)
		}
		if got := r.URL.Query().Get("limit"); got != "50" {
			t.Fatalf("expected default limit=50, got %q", got)
		}
		writeJSON(t, w, http.StatusOK, map[string]any{"memories": []any{}})
	}))
	defer server.Close()

	c := New(Options{URL: server.URL, Key: "test-key", Org: "core", HTTPClient: server.Client(), MaxAttempts: 1})
	if _, err := c.List(context.Background(), ListInput{Project: "mcp"}); err != nil {
		t.Fatalf("List failed: %v", err)
	}
}

func TestClientCall_Good_Retries503ThenSucceeds(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts == 1 {
			writeJSON(t, w, http.StatusServiceUnavailable, map[string]any{"error": "down"})
			return
		}
		writeJSON(t, w, http.StatusOK, map[string]any{"memories": []any{}})
	}))
	defer server.Close()

	c := New(Options{
		URL:         server.URL,
		Key:         "test-key",
		HTTPClient:  server.Client(),
		MaxAttempts: 3,
		BaseDelay:   time.Nanosecond,
	})
	if _, err := c.Recall(context.Background(), RecallInput{Query: "retry"}); err != nil {
		t.Fatalf("Recall failed after retry: %v", err)
	}
	if attempts != 2 {
		t.Fatalf("expected 2 attempts, got %d", attempts)
	}
}

func TestClientCall_Bad_DoesNotRetry400(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		writeJSON(t, w, http.StatusBadRequest, map[string]any{"error": "bad request"})
	}))
	defer server.Close()

	c := New(Options{
		URL:         server.URL,
		Key:         "test-key",
		HTTPClient:  server.Client(),
		MaxAttempts: 3,
		BaseDelay:   time.Nanosecond,
	})
	if _, err := c.Recall(context.Background(), RecallInput{Query: "bad"}); err == nil {
		t.Fatal("expected 400 error")
	}
	if attempts != 1 {
		t.Fatalf("expected one attempt for 400, got %d", attempts)
	}
}

func TestClientCall_Bad_Continuous503OpensCircuit(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		writeJSON(t, w, http.StatusServiceUnavailable, map[string]any{"error": "down"})
	}))
	defer server.Close()

	breaker := NewCircuitBreaker(CircuitBreakerOptions{
		FailureThreshold: 3,
		SuccessThreshold: 1,
		Cooldown:         time.Hour,
	})
	c := New(Options{
		URL:            server.URL,
		Key:            "test-key",
		HTTPClient:     server.Client(),
		MaxAttempts:    3,
		BaseDelay:      time.Nanosecond,
		CircuitBreaker: breaker,
	})

	if _, err := c.Recall(context.Background(), RecallInput{Query: "down"}); err == nil {
		t.Fatal("expected 503 error")
	}
	if breaker.State() != CircuitOpen {
		t.Fatalf("expected circuit open, got %s", breaker.State())
	}
	if _, err := c.Recall(context.Background(), RecallInput{Query: "down"}); !core.Is(err, ErrCircuitOpen) {
		t.Fatalf("expected ErrCircuitOpen, got %v", err)
	}
	if attempts != 3 {
		t.Fatalf("expected no network attempt after circuit open, got %d attempts", attempts)
	}
}

func TestClientCall_Bad_ContextCancellation(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		writeJSON(t, w, http.StatusOK, map[string]any{"ok": true})
	}))
	defer server.Close()

	c := New(Options{URL: server.URL, Key: "test-key", HTTPClient: server.Client(), MaxAttempts: 3})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := c.Recall(ctx, RecallInput{Query: "cancelled"}); !core.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
	if attempts != 0 {
		t.Fatalf("expected cancelled request to avoid network, got %d attempts", attempts)
	}
}

func TestWriteBrainKey_Good_Uses0600(t *testing.T) {
	home := t.TempDir()
	path := filepath.Join(home, ".claude", "brain.key")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("create fixture dir: %v", err)
	}
	if err := os.WriteFile(path, []byte("old-key\n"), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	t.Setenv("HOME", home)

	if err := WriteBrainKey("test-key"); err != nil {
		t.Fatalf("WriteBrainKey failed: %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat brain key: %v", err)
	}
	if got := info.Mode().Perm(); got != brainKeyFileMode {
		t.Fatalf("expected brain.key mode %v, got %v", brainKeyFileMode, got)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read brain key: %v", err)
	}
	if got := string(data); got != "test-key\n" {
		t.Fatalf("expected written key, got %q", got)
	}
}

func TestBrainKeyFile_Bad_RejectsInsecurePermissions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "brain.key")
	if err := os.WriteFile(path, []byte("test-key\n"), brainKeyFileMode); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	if err := os.Chmod(path, 0o644); err != nil {
		t.Fatalf("chmod fixture: %v", err)
	}

	if _, err := readBrainKeyFile(path); err == nil {
		t.Fatal("expected insecure permissions error")
	} else if !strings.Contains(err.Error(), "brain.key has insecure permissions, expected 0600") {
		t.Fatalf("expected insecure permissions error, got %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat brain key: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o644 {
		t.Fatalf("read should not chmod brain.key, got mode %v", got)
	}
}

func readRequestBody(t *testing.T, r *http.Request) map[string]any {
	t.Helper()

	readResult := core.ReadAll(r.Body)
	if !readResult.OK {
		t.Fatalf("failed to read body: %v", readResult.Value)
	}
	body := map[string]any{}
	if decodeResult := core.JSONUnmarshalString(readResult.Value.(string), &body); !decodeResult.OK {
		t.Fatalf("failed to decode body: %v", decodeResult.Value)
	}
	return body
}

func writeJSON(t *testing.T, w http.ResponseWriter, status int, payload any) {
	t.Helper()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if _, err := w.Write([]byte(core.JSONMarshalString(payload))); err != nil {
		t.Fatalf("failed to write response: %v", err)
	}
}
