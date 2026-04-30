// SPDX-License-Identifier: EUPL-1.2

package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"syscall"
	"testing"
	"time"

	core "dappco.re/go"
)

func TestClientRemember_Good_SendsOrgAndAuth(t *testing.T) {
	var gotBody map[string]any

	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

func TestClientCall_Good_BuildsRequestAgainstAPIURL(t *testing.T) {
	gotHost := ""
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/v1/brain/remember" {
			t.Fatalf("expected /v1/brain/remember, got %s", r.URL.Path)
		}
		gotHost = r.Host
		writeJSON(t, w, http.StatusOK, map[string]any{"id": "mem-1"})
	}))
	defer server.Close()

	c := New(Options{
		URL:         server.URL,
		Key:         "test-key",
		HTTPClient:  server.Client(),
		MaxAttempts: 1,
	})

	result, err := c.Call(context.Background(), http.MethodPost, "/v1/brain/remember", map[string]any{"content": "safe"})
	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}
	if result["id"] != "mem-1" {
		t.Fatalf("expected id mem-1, got %v", result["id"])
	}
	if gotHost != core.TrimPrefix(server.URL, "https://") {
		t.Fatalf("expected host %s, got %s", core.TrimPrefix(server.URL, "https://"), gotHost)
	}
}

func TestClientCall_Bad_RejectsAbsoluteRequestURL(t *testing.T) {
	for _, requestPath := range []string{"http://attacker.com/leak", "https://attacker.com/leak"} {
		t.Run(requestPath, func(t *testing.T) {
			calls := 0
			c := New(Options{
				URL: "https://brain.test",
				Key: "test-key",
				HTTPClient: &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
					calls++
					return nil, core.E("test", "unexpected HTTP request", nil)
				})},
				MaxAttempts: 1,
			})

			_, err := c.Call(context.Background(), http.MethodPost, requestPath, map[string]any{"content": "leak"})
			if err == nil {
				t.Fatal("expected absolute URL error")
			}
			if !core.Contains(err.Error(), "absolute request URL rejected") {
				t.Fatalf("expected absolute URL rejection, got %v", err)
			}
			if calls != 0 {
				t.Fatalf("expected no HTTP requests, got %d", calls)
			}
		})
	}
}

func TestClientNew_Bad_RejectsHTTPAPIURLWithoutInsecureEnv(t *testing.T) {
	t.Setenv(insecureBrainEnv, "")

	c := New(Options{URL: "http://internal/", Key: "test-key"})
	if c.configErr == nil {
		t.Fatal("expected insecure HTTP API URL to be rejected")
	}
	if !core.Contains(c.configErr.Error(), "API URL must use https unless CORE_BRAIN_INSECURE=true") {
		t.Fatalf("expected insecure API URL error, got %v", c.configErr)
	}
}

func TestClientNew_Good_AllowsHTTPAPIURLWithInsecureEnv(t *testing.T) {
	t.Setenv(insecureBrainEnv, "true")

	c := New(Options{URL: "http://internal/", Key: "test-key"})
	if c.configErr != nil {
		t.Fatalf("expected insecure HTTP API URL to be allowed, got %v", c.configErr)
	}
}

func TestClientCall_Good_Retries503ThenSucceeds(t *testing.T) {
	attempts := 0
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

func TestClientCall_Good_Retries408ThenSucceeds(t *testing.T) {
	attempts := 0
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts == 1 {
			writeJSON(t, w, http.StatusRequestTimeout, map[string]any{"error": "timeout"})
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

func TestClientCall_Good_Retries429ThenSucceeds(t *testing.T) {
	attempts := 0
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts == 1 {
			writeJSON(t, w, http.StatusTooManyRequests, map[string]any{"error": "rate limited"})
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

func TestClientCall_Good_Retries429UsingRetryAfterSeconds(t *testing.T) {
	attempts := 0
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts == 1 {
			w.Header().Set("Retry-After", "2")
			writeJSON(t, w, http.StatusTooManyRequests, map[string]any{"error": "rate limited"})
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
	sleeps := []time.Duration{}
	c.sleepFunc = func(ctx context.Context, delay time.Duration) error {
		sleeps = append(sleeps, delay)
		return nil
	}

	if _, err := c.Recall(context.Background(), RecallInput{Query: "retry"}); err != nil {
		t.Fatalf("Recall failed after retry: %v", err)
	}
	if attempts != 2 {
		t.Fatalf("expected 2 attempts, got %d", attempts)
	}
	if len(sleeps) != 1 {
		t.Fatalf("expected one retry sleep, got %d", len(sleeps))
	}
	if sleeps[0] != 2*time.Second {
		t.Fatalf("expected Retry-After sleep of 2s, got %v", sleeps[0])
	}
}

func TestClientSleep_Good_AppliesJitterAcrossClients(t *testing.T) {
	ctx := context.Background()
	c1 := New(Options{URL: "https://brain.test", Key: "test-key", BaseDelay: 10 * time.Second})
	c2 := New(Options{URL: "https://brain.test", Key: "test-key", BaseDelay: 10 * time.Second})

	var delay1 time.Duration
	var delay2 time.Duration
	c1.sleepFunc = func(ctx context.Context, delay time.Duration) error {
		delay1 = delay
		return nil
	}
	c2.sleepFunc = func(ctx context.Context, delay time.Duration) error {
		delay2 = delay
		return nil
	}

	for i := 0; i < 10; i++ {
		if err := c1.sleep(ctx, 3); err != nil {
			t.Fatalf("first client sleep failed: %v", err)
		}
		if err := c2.sleep(ctx, 3); err != nil {
			t.Fatalf("second client sleep failed: %v", err)
		}
		if delay1 < 0 || delay1 > maxBackoffDelay {
			t.Fatalf("first client delay out of range: %v", delay1)
		}
		if delay2 < 0 || delay2 > maxBackoffDelay {
			t.Fatalf("second client delay out of range: %v", delay2)
		}
		if delay1 != delay2 {
			return
		}
	}
	t.Fatalf("expected jitter to produce different delays for two clients, both got %v", delay1)
}

func TestJitteredBackoffDelay_Good_CapsHighAttempt(t *testing.T) {
	if limit := backoffDelayLimit(defaultBaseDelay, 20); limit != maxBackoffDelay {
		t.Fatalf("expected high-attempt backoff limit %v, got %v", maxBackoffDelay, limit)
	}
	for i := 0; i < 10; i++ {
		if delay := jitteredBackoffDelay(defaultBaseDelay, 20); delay < 0 || delay > maxBackoffDelay {
			t.Fatalf("expected high-attempt jitter <= %v, got %v", maxBackoffDelay, delay)
		}
	}
}

func TestJitteredBackoffDelay_Good_UsesFullJitterRange(t *testing.T) {
	limit := 800 * time.Millisecond
	if got := backoffDelayLimit(100*time.Millisecond, 3); got != limit {
		t.Fatalf("expected attempt 3 backoff limit %v, got %v", limit, got)
	}
	for i := 0; i < 10; i++ {
		if delay := jitteredBackoffDelay(100*time.Millisecond, 3); delay < 0 || delay > limit {
			t.Fatalf("expected jitter in [0, %v], got %v", limit, delay)
		}
	}
}

func TestClientCall_Good_Retries429WithPastRetryAfterDateWithoutNegativeSleep(t *testing.T) {
	attempts := 0
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts == 1 {
			w.Header().Set("Retry-After", "Wed, 21 Oct 2015 07:28:00 GMT")
			writeJSON(t, w, http.StatusTooManyRequests, map[string]any{"error": "rate limited"})
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
	sleeps := []time.Duration{}
	c.sleepFunc = func(ctx context.Context, delay time.Duration) error {
		sleeps = append(sleeps, delay)
		return nil
	}

	if _, err := c.Recall(context.Background(), RecallInput{Query: "retry"}); err != nil {
		t.Fatalf("Recall failed after retry: %v", err)
	}
	if attempts != 2 {
		t.Fatalf("expected 2 attempts, got %d", attempts)
	}
	if len(sleeps) != 1 {
		t.Fatalf("expected one retry sleep, got %d", len(sleeps))
	}
	if sleeps[0] != 0 {
		t.Fatalf("expected past Retry-After date to sleep zero, got %v", sleeps[0])
	}
}

func TestClientCall_Good_CapsRetryAfterDelay(t *testing.T) {
	attempts := 0
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts == 1 {
			w.Header().Set("Retry-After", "9999")
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
	sleeps := []time.Duration{}
	c.sleepFunc = func(ctx context.Context, delay time.Duration) error {
		sleeps = append(sleeps, delay)
		return nil
	}

	if _, err := c.Recall(context.Background(), RecallInput{Query: "retry"}); err != nil {
		t.Fatalf("Recall failed after retry: %v", err)
	}
	if attempts != 2 {
		t.Fatalf("expected 2 attempts, got %d", attempts)
	}
	if len(sleeps) != 1 {
		t.Fatalf("expected one retry sleep, got %d", len(sleeps))
	}
	if sleeps[0] != maxRetryAfterDelay {
		t.Fatalf("expected capped Retry-After sleep of %v, got %v", maxRetryAfterDelay, sleeps[0])
	}
}

func TestClientCall_Bad_DoesNotRetry400(t *testing.T) {
	attempts := 0
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	path := core.PathJoin(home, ".claude", "brain.key")
	if r := core.MkdirAll(core.PathDir(path), 0o755); !r.OK {
		t.Fatalf("create fixture dir: %v", r.Value)
	}
	if r := core.WriteFile(path, []byte("old-key\n"), 0o644); !r.OK {
		t.Fatalf("write fixture: %v", r.Value)
	}
	t.Setenv("HOME", home)

	if err := WriteBrainKey("test-key"); err != nil {
		t.Fatalf("WriteBrainKey failed: %v", err)
	}
	statResult := core.Stat(path)
	if !statResult.OK {
		t.Fatalf("stat brain key: %v", statResult.Value)
	}
	info := statResult.Value.(core.FsFileInfo)
	if got := info.Mode().Perm(); got != brainKeyFileMode {
		t.Fatalf("expected brain.key mode %v, got %v", brainKeyFileMode, got)
	}
	readResult := core.ReadFile(path)
	if !readResult.OK {
		t.Fatalf("read brain key: %v", readResult.Value)
	}
	data := readResult.Value.([]byte)
	if got := string(data); got != "test-key\n" {
		t.Fatalf("expected written key, got %q", got)
	}
}

func TestBrainKeyFile_Bad_RejectsInsecurePermissions(t *testing.T) {
	path := core.PathJoin(t.TempDir(), "brain.key")
	if r := core.WriteFile(path, []byte("test-key\n"), brainKeyFileMode); !r.OK {
		t.Fatalf("write fixture: %v", r.Value)
	}
	if err := syscall.Chmod(path, 0o644); err != nil {
		t.Fatalf("chmod fixture: %v", err)
	}

	if _, err := readBrainKeyFile(path); err == nil {
		t.Fatal("expected insecure permissions error")
	} else if !core.Contains(err.Error(), "brain.key has insecure permissions, expected 0600") {
		t.Fatalf("expected insecure permissions error, got %v", err)
	}
	statResult := core.Stat(path)
	if !statResult.OK {
		t.Fatalf("stat brain key: %v", statResult.Value)
	}
	info := statResult.Value.(core.FsFileInfo)
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

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return fn(request)
}

// moved AX-7 triplet TestClient_New_Good
func TestClient_New_Good(t *T) {
	c := New(Options{URL: DefaultURL, Key: "test-key", Org: "core", AgentID: "codex"})
	AssertEqual(t, DefaultURL, c.apiURL)
	AssertEqual(t, "codex", c.agentID)
}

// moved AX-7 triplet TestClient_New_Bad
func TestClient_New_Bad(t *T) {
	c := New(Options{URL: "://bad", Key: "test-key"})
	AssertNotNil(t, c.configErr)
	AssertContains(t, c.configErr.Error(), "invalid API URL")
}

// moved AX-7 triplet TestClient_New_Ugly
func TestClient_New_Ugly(t *T) {
	c := New(Options{})
	AssertEqual(t, DefaultURL, c.apiURL)
	AssertEqual(t, defaultAgentID, c.agentID)
}

// moved AX-7 triplet TestClient_NewFromEnvironment_Good
func TestClient_NewFromEnvironment_Good(t *T) {
	t.Setenv("CORE_BRAIN_KEY", "env-key")
	t.Setenv("CORE_BRAIN_URL", DefaultURL)
	c := NewFromEnvironment()
	AssertEqual(t, "env-key", c.apiKey)
	AssertEqual(t, DefaultURL, c.apiURL)
}

// moved AX-7 triplet TestClient_NewFromEnvironment_Bad
func TestClient_NewFromEnvironment_Bad(t *T) {
	t.Setenv("CORE_BRAIN_KEY", "env-key")
	t.Setenv("CORE_BRAIN_URL", "://bad")
	c := NewFromEnvironment()
	AssertNotNil(t, c.configErr)
	AssertContains(t, c.configErr.Error(), "invalid API URL")
}

// moved AX-7 triplet TestClient_NewFromEnvironment_Ugly
func TestClient_NewFromEnvironment_Ugly(t *T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("CORE_BRAIN_KEY", "")
	AssertNoError(t, WriteBrainKey("file-key"))
	c := NewFromEnvironment()
	AssertEqual(t, "file-key", c.apiKey)
}

// moved AX-7 triplet TestClient_WriteBrainKey_Good
func TestClient_WriteBrainKey_Good(t *T) {
	t.Setenv("HOME", t.TempDir())
	AssertNoError(t, WriteBrainKey("test-key"))
	got, err := readBrainKeyFile(brainKeyPath(core.Env("HOME")))
	AssertNoError(t, err)
	AssertEqual(t, "test-key", got)
}

// moved AX-7 triplet TestClient_WriteBrainKey_Bad
func TestClient_WriteBrainKey_Bad(t *T) {
	t.Setenv("HOME", "")
	err := WriteBrainKey("test-key")
	AssertError(t, err)
	AssertContains(t, err.Error(), "HOME not set")
}

// moved AX-7 triplet TestClient_WriteBrainKey_Ugly
func TestClient_WriteBrainKey_Ugly(t *T) {
	t.Setenv("HOME", t.TempDir())
	AssertNoError(t, WriteBrainKey("  test-key  "))
	got, err := readBrainKeyFile(brainKeyPath(core.Env("HOME")))
	AssertNoError(t, err)
	AssertEqual(t, "test-key", got)
}

// moved AX-7 triplet TestClient_NewCircuitBreaker_Good
func TestClient_NewCircuitBreaker_Good(t *T) {
	breaker := NewCircuitBreaker(CircuitBreakerOptions{FailureThreshold: 2, SuccessThreshold: 2, Cooldown: time.Second})
	AssertEqual(t, CircuitClosed, breaker.State())
	AssertEqual(t, 2, breaker.failureThreshold)
}

// moved AX-7 triplet TestClient_NewCircuitBreaker_Bad
func TestClient_NewCircuitBreaker_Bad(t *T) {
	breaker := NewCircuitBreaker(CircuitBreakerOptions{})
	AssertEqual(t, defaultFailureThreshold, breaker.failureThreshold)
	AssertEqual(t, CircuitClosed, breaker.State())
}

// moved AX-7 triplet TestClient_NewCircuitBreaker_Ugly
func TestClient_NewCircuitBreaker_Ugly(t *T) {
	breaker := NewCircuitBreaker(CircuitBreakerOptions{FailureThreshold: -1, SuccessThreshold: -1, Cooldown: -1})
	AssertEqual(t, defaultSuccessThreshold, breaker.successThreshold)
	AssertEqual(t, defaultCircuitCooldown, breaker.cooldown)
}

// moved AX-7 triplet TestClient_CircuitBreaker_State_Good
func TestClient_CircuitBreaker_State_Good(t *T) {
	breaker := NewCircuitBreaker(CircuitBreakerOptions{})
	state := breaker.State()
	AssertEqual(t, CircuitClosed, state)
}

// moved AX-7 triplet TestClient_CircuitBreaker_State_Bad
func TestClient_CircuitBreaker_State_Bad(t *T) {
	var breaker *CircuitBreaker
	state := breaker.State()
	AssertEqual(t, CircuitClosed, state)
}

// moved AX-7 triplet TestClient_CircuitBreaker_State_Ugly
func TestClient_CircuitBreaker_State_Ugly(t *T) {
	breaker := NewCircuitBreaker(CircuitBreakerOptions{Cooldown: time.Nanosecond})
	breaker.state = CircuitOpen
	breaker.openedAt = time.Now().Add(-time.Second)
	AssertEqual(t, CircuitHalfOpen, breaker.State())
}

// moved AX-7 triplet TestClient_Client_Remember_Good
func TestClient_Client_Remember_Good(t *T) {
	c, server := brainClientForTest(t, func(w http.ResponseWriter, r *http.Request) {
		AssertEqual(t, http.MethodPost, r.Method)
		AssertEqual(t, "/v1/brain/remember", r.URL.Path)
		writeJSONForTest(t, w, http.StatusOK, map[string]any{"id": "mem-1"})
	})
	defer server.Close()
	result, err := c.Remember(context.Background(), RememberInput{Content: "remember", Type: "decision"})
	AssertNoError(t, err)
	AssertEqual(t, "mem-1", result["id"])
}

// moved AX-7 triplet TestClient_Client_Remember_Bad
func TestClient_Client_Remember_Bad(t *T) {
	c := New(Options{URL: DefaultURL})
	result, err := c.Remember(context.Background(), RememberInput{Content: "remember"})
	AssertError(t, err)
	AssertNil(t, result)
}

// moved AX-7 triplet TestClient_Client_Remember_Ugly
func TestClient_Client_Remember_Ugly(t *T) {
	c, server := brainClientForTest(t, func(w http.ResponseWriter, r *http.Request) {
		AssertContains(t, r.Header.Get("Authorization"), "Bearer")
		writeJSONForTest(t, w, http.StatusOK, map[string]any{"id": "mem-empty"})
	})
	defer server.Close()
	result, err := c.Remember(context.Background(), RememberInput{})
	AssertNoError(t, err)
	AssertEqual(t, "mem-empty", result["id"])
}

// moved AX-7 triplet TestClient_Client_Recall_Good
func TestClient_Client_Recall_Good(t *T) {
	c, server := brainClientForTest(t, func(w http.ResponseWriter, r *http.Request) {
		AssertEqual(t, "/v1/brain/recall", r.URL.Path)
		writeJSONForTest(t, w, http.StatusOK, map[string]any{"memories": []any{}})
	})
	defer server.Close()
	result, err := c.Recall(context.Background(), RecallInput{Query: "query"})
	AssertNoError(t, err)
	AssertNotNil(t, result["memories"])
}

// moved AX-7 triplet TestClient_Client_Recall_Bad
func TestClient_Client_Recall_Bad(t *T) {
	c := New(Options{URL: DefaultURL})
	result, err := c.Recall(context.Background(), RecallInput{Query: "query"})
	AssertError(t, err)
	AssertNil(t, result)
}

// moved AX-7 triplet TestClient_Client_Recall_Ugly
func TestClient_Client_Recall_Ugly(t *T) {
	c, server := brainClientForTest(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSONForTest(t, w, http.StatusOK, map[string]any{"memories": []any{map[string]any{"id": "m1"}}})
	})
	defer server.Close()
	result, err := c.Recall(context.Background(), RecallInput{})
	AssertNoError(t, err)
	AssertLen(t, result["memories"], 1)
}

// moved AX-7 triplet TestClient_Client_Forget_Good
func TestClient_Client_Forget_Good(t *T) {
	c, server := brainClientForTest(t, func(w http.ResponseWriter, r *http.Request) {
		AssertEqual(t, http.MethodDelete, r.Method)
		AssertEqual(t, "/v1/brain/forget/mem-1", r.URL.Path)
		writeJSONForTest(t, w, http.StatusOK, map[string]any{"forgotten": "mem-1"})
	})
	defer server.Close()
	result, err := c.Forget(context.Background(), ForgetInput{ID: "mem-1"})
	AssertNoError(t, err)
	AssertEqual(t, "mem-1", result["forgotten"])
}

// moved AX-7 triplet TestClient_Client_Forget_Bad
func TestClient_Client_Forget_Bad(t *T) {
	c := New(Options{URL: DefaultURL})
	result, err := c.Forget(context.Background(), ForgetInput{ID: "mem-1"})
	AssertError(t, err)
	AssertNil(t, result)
}

// moved AX-7 triplet TestClient_Client_Forget_Ugly
func TestClient_Client_Forget_Ugly(t *T) {
	c, server := brainClientForTest(t, func(w http.ResponseWriter, r *http.Request) {
		AssertTrue(t, core.HasPrefix(r.URL.Path, "/v1/brain/forget/"))
		writeJSONForTest(t, w, http.StatusOK, map[string]any{"forgotten": ""})
	})
	defer server.Close()
	result, err := c.Forget(context.Background(), ForgetInput{})
	AssertNoError(t, err)
	AssertEqual(t, "", result["forgotten"])
}

// moved AX-7 triplet TestClient_Client_List_Good
func TestClient_Client_List_Good(t *T) {
	c, server := brainClientForTest(t, func(w http.ResponseWriter, r *http.Request) {
		AssertEqual(t, "core", r.URL.Query().Get("org"))
		AssertEqual(t, "50", r.URL.Query().Get("limit"))
		writeJSONForTest(t, w, http.StatusOK, map[string]any{"memories": []any{}})
	})
	defer server.Close()
	result, err := c.List(context.Background(), ListInput{})
	AssertNoError(t, err)
	AssertNotNil(t, result["memories"])
}

// moved AX-7 triplet TestClient_Client_List_Bad
func TestClient_Client_List_Bad(t *T) {
	c := New(Options{URL: DefaultURL})
	result, err := c.List(context.Background(), ListInput{})
	AssertError(t, err)
	AssertNil(t, result)
}

// moved AX-7 triplet TestClient_Client_List_Ugly
func TestClient_Client_List_Ugly(t *T) {
	c, server := brainClientForTest(t, func(w http.ResponseWriter, r *http.Request) {
		AssertEqual(t, "-1", r.URL.Query().Get("limit"))
		writeJSONForTest(t, w, http.StatusOK, map[string]any{"memories": []any{}})
	})
	defer server.Close()
	result, err := c.List(context.Background(), ListInput{Limit: -1})
	AssertNoError(t, err)
	AssertNotNil(t, result["memories"])
}

// moved AX-7 triplet TestClient_Client_Call_Good
func TestClient_Client_Call_Good(t *T) {
	c, server := brainClientForTest(t, func(w http.ResponseWriter, r *http.Request) {
		AssertEqual(t, "/v1/brain/status", r.URL.Path)
		writeJSONForTest(t, w, http.StatusOK, map[string]any{"ok": true})
	})
	defer server.Close()
	result, err := c.Call(context.Background(), http.MethodGet, "/v1/brain/status", nil)
	AssertNoError(t, err)
	AssertEqual(t, true, result["ok"])
}

// moved AX-7 triplet TestClient_Client_Call_Bad
func TestClient_Client_Call_Bad(t *T) {
	c := New(Options{URL: DefaultURL, Key: "test-key"})
	result, err := c.Call(context.Background(), http.MethodGet, "https://attacker.test/leak", nil)
	AssertError(t, err)
	AssertNil(t, result)
}

// moved AX-7 triplet TestClient_Client_Call_Ugly
func TestClient_Client_Call_Ugly(t *T) {
	c, server := brainClientForTest(t, func(w http.ResponseWriter, r *http.Request) {
		AssertEqual(t, "/relative", r.URL.Path)
		writeJSONForTest(t, w, http.StatusOK, map[string]any{"relative": true})
	})
	defer server.Close()
	result, err := c.Call(context.Background(), http.MethodGet, "relative", nil)
	AssertNoError(t, err)
	AssertEqual(t, true, result["relative"])
}
