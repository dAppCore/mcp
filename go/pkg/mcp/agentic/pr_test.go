// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestForgeCreatePR_Bad_InvalidJSONResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("{not-json"))
	}))
	defer srv.Close()

	s := &PrepSubsystem{
		forgeURL: srv.URL,
		client:   srv.Client(),
	}

	_, _, err := s.forgeCreatePR(context.Background(), "core", "demo", "agent/test", "main", "Fix bug", "body")
	if err == nil {
		t.Fatal("expected malformed PR response to fail")
	}
}
