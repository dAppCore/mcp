package client

import (
	"github.com/goccy/go-json"
	"net/http"
	"net/http/httptest"
	"time"

	core "dappco.re/go"
)

// moved helpers from ax7_triplets_test.go
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

func brainClientForTest(t *T, handler http.HandlerFunc) (*Client, *httptest.Server) {
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

func writeJSONForTest(t *T, w http.ResponseWriter, status int, payload any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		t.Fatalf("encode response: %v", err)
	}
}

func localFSForTest(t *T) *localCoreFS {
	t.Helper()
	r := core.PathEvalSymlinks(t.TempDir())
	AssertTrue(t, r.OK)
	root := r.Value.(string)
	return &localCoreFS{fs: (&core.Fs{}).New(root)}
}
