// SPDX-License-Identifier: EUPL-1.2

package brain

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	brainclient "dappco.re/go/mcp/pkg/mcp/brain/client"
)

const testInsecureBrainEnv = "CORE_BRAIN_INSECURE"

// newTestDirect creates a DirectSubsystem pointing at a test server.
func newTestDirect(t *testing.T, url string) *DirectSubsystem {
	t.Helper()
	t.Setenv(testInsecureBrainEnv, "true")

	return &DirectSubsystem{
		apiClient: brainclient.New(brainclient.Options{
			URL:         url,
			Key:         "test-key",
			HTTPClient:  http.DefaultClient,
			MaxAttempts: 1,
			BaseDelay:   time.Nanosecond,
		}),
	}
}

// --- DirectSubsystem interface tests ---

func TestDirectSubsystem_Good_Name(t *testing.T) {
	s := &DirectSubsystem{}
	if s.Name() != "brain" {
		t.Errorf("expected Name() = 'brain', got %q", s.Name())
	}
}

func TestDirectSubsystem_Good_Shutdown(t *testing.T) {
	s := &DirectSubsystem{}
	if err := s.Shutdown(context.Background()); err != nil {
		t.Errorf("Shutdown failed: %v", err)
	}
}

// --- apiCall tests ---

func TestApiCall_Good_PostWithBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("missing or wrong Authorization header")
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("missing Content-Type header")
		}
		w.WriteHeader(200)
		json.NewEncoder(w).Encode(map[string]any{"id": "mem-123", "success": true})
	}))
	defer srv.Close()

	s := newTestDirect(t, srv.URL)
	result, err := s.apiCall(context.Background(), "POST", "/v1/brain/remember", map[string]string{"content": "test"})
	if err != nil {
		t.Fatalf("apiCall failed: %v", err)
	}
	if result["id"] != "mem-123" {
		t.Errorf("expected id=mem-123, got %v", result["id"])
	}
}

func TestApiCall_Good_GetNilBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET, got %s", r.Method)
		}
		w.WriteHeader(200)
		json.NewEncoder(w).Encode(map[string]any{"status": "ok"})
	}))
	defer srv.Close()

	s := newTestDirect(t, srv.URL)
	result, err := s.apiCall(context.Background(), "GET", "/status", nil)
	if err != nil {
		t.Fatalf("apiCall failed: %v", err)
	}
	if result["status"] != "ok" {
		t.Errorf("expected status=ok, got %v", result["status"])
	}
}

func TestApiCall_Bad_NoApiKey(t *testing.T) {
	s := &DirectSubsystem{apiClient: brainclient.New(brainclient.Options{
		URL:         "http://example.test",
		Key:         "",
		HTTPClient:  http.DefaultClient,
		MaxAttempts: 1,
	})}
	_, err := s.apiCall(context.Background(), "GET", "/test", nil)
	if err == nil {
		t.Error("expected error when apiKey is empty")
	}
}

func TestApiCall_Bad_HttpError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		w.Write([]byte(`{"error":"internal server error"}`))
	}))
	defer srv.Close()

	s := newTestDirect(t, srv.URL)
	_, err := s.apiCall(context.Background(), "POST", "/fail", map[string]string{})
	if err == nil {
		t.Error("expected error on HTTP 500")
	}
}

func TestApiCall_Bad_InvalidJson(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("not json"))
	}))
	defer srv.Close()

	s := newTestDirect(t, srv.URL)
	_, err := s.apiCall(context.Background(), "GET", "/bad-json", nil)
	if err == nil {
		t.Error("expected error on invalid JSON response")
	}
}

func TestApiCall_Bad_Unreachable(t *testing.T) {
	s := &DirectSubsystem{
		apiClient: brainclient.New(brainclient.Options{
			URL:         "http://127.0.0.1:1", // nothing listening
			Key:         "key",
			HTTPClient:  http.DefaultClient,
			MaxAttempts: 1,
		}),
	}
	_, err := s.apiCall(context.Background(), "GET", "/test", nil)
	if err == nil {
		t.Error("expected error for unreachable server")
	}
}

// --- remember tool tests ---

func TestDirectRemember_Good(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		if body["content"] != "test memory" {
			t.Errorf("unexpected content: %v", body["content"])
		}
		if body["agent_id"] != "cladius" {
			t.Errorf("expected agent_id=cladius, got %v", body["agent_id"])
		}
		if body["org"] != "core" {
			t.Errorf("expected org=core, got %v", body["org"])
		}
		w.WriteHeader(200)
		json.NewEncoder(w).Encode(map[string]any{"id": "mem-456"})
	}))
	defer srv.Close()

	s := newTestDirect(t, srv.URL)
	_, out, err := s.remember(context.Background(), nil, RememberInput{
		Content: "test memory",
		Type:    "observation",
		Org:     "core",
		Project: "test-project",
	})
	if err != nil {
		t.Fatalf("remember failed: %v", err)
	}
	if !out.Success {
		t.Error("expected success=true")
	}
	if out.MemoryID != "mem-456" {
		t.Errorf("expected memoryId=mem-456, got %q", out.MemoryID)
	}
}

func TestDirectRemember_Bad_ApiError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(422)
		w.Write([]byte(`{"error":"validation failed"}`))
	}))
	defer srv.Close()

	s := newTestDirect(t, srv.URL)
	_, _, err := s.remember(context.Background(), nil, RememberInput{Content: "x", Type: "bug"})
	if err == nil {
		t.Error("expected error on API failure")
	}
}

// --- recall tool tests ---

func TestDirectRecall_Good(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		if body["query"] != "scoring algorithm" {
			t.Errorf("unexpected query: %v", body["query"])
		}
		if body["org"] != "core" {
			t.Errorf("expected org=core, got %v", body["org"])
		}
		w.WriteHeader(200)
		json.NewEncoder(w).Encode(map[string]any{
			"memories": []any{
				map[string]any{
					"id":         "mem-1",
					"content":    "scoring uses weighted average",
					"type":       "architecture",
					"org":        "core",
					"project":    "eaas",
					"agent_id":   "virgil",
					"score":      0.92,
					"created_at": "2026-03-01T00:00:00Z",
				},
			},
		})
	}))
	defer srv.Close()

	s := newTestDirect(t, srv.URL)
	_, out, err := s.recall(context.Background(), nil, RecallInput{
		Query:  "scoring algorithm",
		TopK:   5,
		Filter: RecallFilter{Org: "core", Project: "eaas"},
	})
	if err != nil {
		t.Fatalf("recall failed: %v", err)
	}
	if !out.Success || out.Count != 1 {
		t.Errorf("expected 1 memory, got %d", out.Count)
	}
	if out.Memories[0].ID != "mem-1" {
		t.Errorf("expected id=mem-1, got %q", out.Memories[0].ID)
	}
	if out.Memories[0].Org != "core" {
		t.Errorf("expected org=core, got %q", out.Memories[0].Org)
	}
	if out.Memories[0].Confidence != 0.92 {
		t.Errorf("expected score=0.92, got %f", out.Memories[0].Confidence)
	}
}

func TestDirectRecall_Good_DefaultTopK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		// TopK=0 should default to 10
		if topK, ok := body["top_k"].(float64); !ok || topK != 10 {
			t.Errorf("expected top_k=10, got %v", body["top_k"])
		}
		w.WriteHeader(200)
		json.NewEncoder(w).Encode(map[string]any{"memories": []any{}})
	}))
	defer srv.Close()

	s := newTestDirect(t, srv.URL)
	_, out, err := s.recall(context.Background(), nil, RecallInput{Query: "test"})
	if err != nil {
		t.Fatalf("recall failed: %v", err)
	}
	if !out.Success || out.Count != 0 {
		t.Errorf("expected empty result, got %d", out.Count)
	}
}

func TestDirectRecall_Bad_ApiError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		w.Write([]byte(`{"error":"internal"}`))
	}))
	defer srv.Close()

	s := newTestDirect(t, srv.URL)
	_, _, err := s.recall(context.Background(), nil, RecallInput{Query: "test"})
	if err == nil {
		t.Error("expected error on API failure")
	}
}

// --- forget tool tests ---

func TestDirectForget_Good(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/v1/brain/forget/mem-789" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(200)
		json.NewEncoder(w).Encode(map[string]any{"success": true})
	}))
	defer srv.Close()

	s := newTestDirect(t, srv.URL)
	_, out, err := s.forget(context.Background(), nil, ForgetInput{
		ID:     "mem-789",
		Reason: "outdated",
	})
	if err != nil {
		t.Fatalf("forget failed: %v", err)
	}
	if !out.Success || out.Forgotten != "mem-789" {
		t.Errorf("unexpected output: %+v", out)
	}
}

func TestDirectForget_Good_EmitsChannel(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		json.NewEncoder(w).Encode(map[string]any{"success": true})
	}))
	defer srv.Close()

	var gotChannel string
	var gotPayload map[string]any

	s := newTestDirect(t, srv.URL)
	s.onChannel = func(_ context.Context, channel string, data any) {
		gotChannel = channel
		if payload, ok := data.(map[string]any); ok {
			gotPayload = payload
		}
	}

	_, out, err := s.forget(context.Background(), nil, ForgetInput{
		ID:     "mem-789",
		Reason: "outdated",
	})
	if err != nil {
		t.Fatalf("forget failed: %v", err)
	}
	if !out.Success {
		t.Fatal("expected success=true")
	}
	if gotChannel != "brain.forget.complete" {
		t.Fatalf("expected brain.forget.complete, got %q", gotChannel)
	}
	if gotPayload == nil {
		t.Fatal("expected channel payload")
	}
	if gotPayload["id"] != "mem-789" {
		t.Fatalf("expected id=mem-789, got %v", gotPayload["id"])
	}
	if gotPayload["reason"] != "outdated" {
		t.Fatalf("expected reason=outdated, got %v", gotPayload["reason"])
	}
}

func TestDirectForget_Bad_ApiError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
		w.Write([]byte(`{"error":"not found"}`))
	}))
	defer srv.Close()

	s := newTestDirect(t, srv.URL)
	_, _, err := s.forget(context.Background(), nil, ForgetInput{ID: "nonexistent"})
	if err == nil {
		t.Error("expected error on 404")
	}
}

// --- list tool tests ---

func TestDirectList_Good(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if got := r.URL.Query().Get("project"); got != "eaas" {
			t.Errorf("expected project=eaas, got %q", got)
		}
		if got := r.URL.Query().Get("org"); got != "core" {
			t.Errorf("expected org=core, got %q", got)
		}
		if got := r.URL.Query().Get("type"); got != "decision" {
			t.Errorf("expected type=decision, got %q", got)
		}
		if got := r.URL.Query().Get("agent_id"); got != "virgil" {
			t.Errorf("expected agent_id=virgil, got %q", got)
		}
		if got := r.URL.Query().Get("limit"); got != "20" {
			t.Errorf("expected limit=20, got %q", got)
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"memories": []any{
				map[string]any{
					"id":         "mem-1",
					"content":    "use qdrant",
					"type":       "decision",
					"org":        "core",
					"project":    "eaas",
					"agent_id":   "virgil",
					"score":      0.88,
					"created_at": "2026-03-01T00:00:00Z",
				},
			},
		})
	}))
	defer srv.Close()

	s := newTestDirect(t, srv.URL)
	_, out, err := s.list(context.Background(), nil, ListInput{
		Org:     "core",
		Project: "eaas",
		Type:    "decision",
		AgentID: "virgil",
		Limit:   20,
	})
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if !out.Success || out.Count != 1 {
		t.Fatalf("expected 1 memory, got %+v", out)
	}
	if out.Memories[0].ID != "mem-1" {
		t.Errorf("expected id=mem-1, got %q", out.Memories[0].ID)
	}
	if out.Memories[0].Confidence != 0.88 {
		t.Errorf("expected score=0.88, got %f", out.Memories[0].Confidence)
	}
	if out.Memories[0].Org != "core" {
		t.Errorf("expected org=core, got %q", out.Memories[0].Org)
	}
}

func TestDirectList_Good_EmitsAgentIDChannelPayload(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{"memories": []any{}})
	}))
	defer srv.Close()

	var gotChannel string
	var gotPayload map[string]any

	s := newTestDirect(t, srv.URL)
	s.onChannel = func(_ context.Context, channel string, data any) {
		gotChannel = channel
		if payload, ok := data.(map[string]any); ok {
			gotPayload = payload
		}
	}

	_, out, err := s.list(context.Background(), nil, ListInput{
		Org:     "core",
		Project: "eaas",
		Type:    "decision",
		AgentID: "virgil",
		Limit:   20,
	})
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if !out.Success {
		t.Fatal("expected list success")
	}
	if gotChannel != "brain.list.complete" {
		t.Fatalf("expected brain.list.complete, got %q", gotChannel)
	}
	if gotPayload == nil {
		t.Fatal("expected channel payload")
	}
	if gotPayload["agent_id"] != "virgil" {
		t.Fatalf("expected agent_id=virgil, got %v", gotPayload["agent_id"])
	}
	if gotPayload["project"] != "eaas" {
		t.Fatalf("expected project=eaas, got %v", gotPayload["project"])
	}
	if gotPayload["org"] != "core" {
		t.Fatalf("expected org=core, got %v", gotPayload["org"])
	}
}

func TestDirectList_Good_DefaultLimit(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("limit"); got != "50" {
			t.Errorf("expected limit=50, got %q", got)
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{"memories": []any{}})
	}))
	defer srv.Close()

	s := newTestDirect(t, srv.URL)
	_, out, err := s.list(context.Background(), nil, ListInput{})
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if !out.Success || out.Count != 0 {
		t.Fatalf("expected empty list, got %+v", out)
	}
}
