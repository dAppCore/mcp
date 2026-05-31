// SPDX-License-Identifier: EUPL-1.2

package mcp

import (
	"context"
	"testing"
)

func TestAuthzHelpers_claimsFromContext_Good_Bad_Ugly(t *testing.T) {
	// Good: claims stored via withAuthClaims round-trip back out.
	want := &authClaims{Workspace: "core/mcp", Subject: "cladius"}
	ctx := withAuthClaims(context.Background(), want)
	if got := claimsFromContext(ctx); got != want {
		t.Fatalf("claimsFromContext = %v, want %v", got, want)
	}

	// Bad: a context with no claims yields nil.
	if got := claimsFromContext(context.Background()); got != nil {
		t.Fatalf("expected nil claims for bare context, got %v", got)
	}

	// Ugly: a nil context must not panic and yields nil.
	if got := claimsFromContext(nil); got != nil {
		t.Fatalf("expected nil claims for nil context, got %v", got)
	}
}

func TestAuthzHelpers_withAuthClaims_Ugly_NilContext(t *testing.T) {
	// withAuthClaims on a nil context returns a usable background context but does
	// NOT carry the claims (it bails before WithValue). The contract here is
	// "do not panic on nil"; claims simply are not retrievable.
	ctx := withAuthClaims(nil, &authClaims{Subject: "x"})
	if ctx == nil {
		t.Fatal("expected non-nil context")
	}
	if claimsFromContext(ctx) != nil {
		t.Fatal("expected no claims to be carried on a recovered nil context")
	}
}

func TestAuthzHelpers_authClaimsFromToolRequest_Good_NoTokenDisablesAuth(t *testing.T) {
	// With an empty API token, auth is disabled: no claims, not in-transport, no error.
	claims, inTransport, err := authClaimsFromToolRequest(context.Background(), nil, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if claims != nil || inTransport {
		t.Fatalf("expected disabled auth (nil, false), got (%v, %v)", claims, inTransport)
	}
}

func TestAuthzHelpers_authClaimsFromToolRequest_Good_ContextClaims(t *testing.T) {
	// nil request + claims present in context → returns them, in-transport.
	want := &authClaims{Workspace: "core/mcp"}
	ctx := withAuthClaims(context.Background(), want)

	claims, inTransport, err := authClaimsFromToolRequest(ctx, nil, "an-api-token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !inTransport {
		t.Fatal("expected inTransport true when claims supplied via context")
	}
	if claims != want {
		t.Fatalf("claims = %v, want %v", claims, want)
	}
}

func TestAuthzHelpers_authClaimsFromToolRequest_Ugly_NoClaimsWithToken(t *testing.T) {
	// API token set, nil request, no context claims → not in-transport, no error
	// (in-process direct call path).
	claims, inTransport, err := authClaimsFromToolRequest(context.Background(), nil, "an-api-token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if claims != nil || inTransport {
		t.Fatalf("expected (nil, false) for direct in-process call, got (%v, %v)", claims, inTransport)
	}
}

type wsStructInput struct {
	Repo  string `json:"repo"`
	Other string `json:"other"`
}

type wsPtrInput struct {
	Workspace *string `json:"workspace"`
}

func TestAuthzHelpers_inputWorkspaceFromValue_Good_Struct(t *testing.T) {
	if got := inputWorkspaceFromValue(wsStructInput{Repo: "core/mcp"}); got != "core/mcp" {
		t.Fatalf("struct repo = %q, want core/mcp", got)
	}
	// Pointer to struct is dereferenced.
	if got := inputWorkspaceFromValue(&wsStructInput{Repo: "core/api"}); got != "core/api" {
		t.Fatalf("ptr struct repo = %q, want core/api", got)
	}
	// Pointer-string field.
	ws := "core/desktop"
	if got := inputWorkspaceFromValue(wsPtrInput{Workspace: &ws}); got != "core/desktop" {
		t.Fatalf("ptr-string workspace = %q, want core/desktop", got)
	}
}

func TestAuthzHelpers_inputWorkspaceFromValue_Good_Map(t *testing.T) {
	if got := inputWorkspaceFromValue(map[string]any{"workspace": "core/lthn"}); got != "core/lthn" {
		t.Fatalf("map workspace = %q", got)
	}
	// repository alias.
	if got := inputWorkspaceFromValue(map[string]string{"repository": "core/go"}); got != "core/go" {
		t.Fatalf("map repository = %q", got)
	}
}

func TestAuthzHelpers_inputWorkspaceFromValue_Ugly_NoMatch(t *testing.T) {
	if got := inputWorkspaceFromValue(nil); got != "" {
		t.Fatalf("nil input = %q, want empty", got)
	}
	if got := inputWorkspaceFromValue(wsStructInput{Other: "irrelevant"}); got != "" {
		t.Fatalf("struct with no ws field = %q, want empty", got)
	}
	// A nil pointer-string field must not panic and yields empty.
	if got := inputWorkspaceFromValue(wsPtrInput{Workspace: nil}); got != "" {
		t.Fatalf("nil ptr workspace = %q, want empty", got)
	}
	// Non-struct, non-map kinds yield empty.
	if got := inputWorkspaceFromValue(42); got != "" {
		t.Fatalf("int input = %q, want empty", got)
	}
}
