// SPDX-License-Identifier: EUPL-1.2

package mcp

import (
	"testing"
)

// TestAuthz_CurrentAuthConfig_Good asserts the JWT signing secret is read from
// MCP_JWT_SECRET, distinct from the API token.
func TestAuthz_CurrentAuthConfig_Good(t *testing.T) {
	t.Setenv("MCP_JWT_SECRET", "distinct-signing-key")

	cfg := currentAuthConfig("api-token-value")
	if string(cfg.secret) != "distinct-signing-key" {
		t.Fatalf("expected secret from MCP_JWT_SECRET, got %q", string(cfg.secret))
	}
	if string(cfg.secret) == cfg.apiToken {
		t.Fatal("signing secret must never equal the API token")
	}
}

// TestAuthz_CurrentAuthConfig_Bad asserts there is NO fallback to the API token
// when MCP_JWT_SECRET is unset — the conflation that let any token-holder
// self-mint tool:* claims is removed (S1.3).
func TestAuthz_CurrentAuthConfig_Bad(t *testing.T) {
	t.Setenv("MCP_JWT_SECRET", "")

	cfg := currentAuthConfig("api-token-value")
	if len(cfg.secret) != 0 {
		t.Fatalf("expected empty signing secret with no MCP_JWT_SECRET (no API-token fallback), got %q", string(cfg.secret))
	}
}

// TestAuthz_ParseAuthClaims_Bad_NoSelfMintWithAPIToken asserts S1.3: a JWT
// forged by signing with the API token (the old fallback secret) does NOT
// verify when a distinct MCP_JWT_SECRET is configured. A token-holder cannot
// self-mint entitlement claims.
func TestAuthz_ParseAuthClaims_Bad_NoSelfMintWithAPIToken(t *testing.T) {
	t.Setenv("MCP_JWT_SECRET", "distinct-signing-key")

	const apiToken = "api-token-value"

	// Attacker mints a JWT granting full tool access, signing with the API
	// token they hold (the pre-fix fallback secret).
	forgedCfg := authConfig{
		apiToken: apiToken,
		secret:   []byte(apiToken),
		ttl:      authDefaultJWTTTL,
	}
	forged, err := mintJWTToken(authClaims{
		Subject:      "attacker",
		Entitlements: []string{"tool:*"},
	}, forgedCfg)
	if err != nil {
		t.Fatalf("mint forged token: %v", err)
	}

	// The served transport verifies against the distinct MCP_JWT_SECRET, so
	// the forged signature must be rejected.
	claims, err := parseAuthClaims("Bearer "+forged, apiToken)
	if err == nil {
		t.Fatalf("expected forged self-minted JWT to be rejected, got claims=%+v", claims)
	}
}

// TestAuthz_ParseAuthClaims_Good_DistinctSecretRoundTrip asserts a JWT minted
// with the distinct MCP_JWT_SECRET verifies and carries its claims.
func TestAuthz_ParseAuthClaims_Good_DistinctSecretRoundTrip(t *testing.T) {
	t.Setenv("MCP_JWT_SECRET", "distinct-signing-key")

	const apiToken = "api-token-value"
	cfg := currentAuthConfig(apiToken)

	minted, err := mintJWTToken(authClaims{
		Subject:      "legit",
		Workspace:    "core/mcp",
		Entitlements: []string{"tool:dispatch"},
	}, cfg)
	if err != nil {
		t.Fatalf("mint token: %v", err)
	}

	claims, err := parseAuthClaims("Bearer "+minted, apiToken)
	if err != nil {
		t.Fatalf("expected minted token to verify, got: %v", err)
	}
	if claims == nil || claims.Subject != "legit" || claims.Workspace != "core/mcp" {
		t.Fatalf("unexpected claims: %+v", claims)
	}
}

// TestAuthz_ParseAuthClaims_Good_APITokenDirectMatch asserts the bearer
// credential itself (the configured API token) authenticates directly via
// constant-time compare — this is the per-request Authorization path.
func TestAuthz_ParseAuthClaims_Good_APITokenDirectMatch(t *testing.T) {
	t.Setenv("MCP_JWT_SECRET", "distinct-signing-key")

	const apiToken = "api-token-value"
	claims, err := parseAuthClaims("Bearer "+apiToken, apiToken)
	if err != nil {
		t.Fatalf("expected API token to authenticate, got: %v", err)
	}
	if claims == nil || claims.Subject != "api-key" {
		t.Fatalf("expected api-key subject, got: %+v", claims)
	}
}

// TestAuthz_ServedAuthConfigError_Good asserts no error when both the bearer
// token and a distinct signing secret are configured.
func TestAuthz_ServedAuthConfigError_Good(t *testing.T) {
	t.Setenv("MCP_AUTH_TOKEN", "test-secret-token")
	t.Setenv("MCP_JWT_SECRET", "distinct-signing-key")
	if err := servedAuthConfigError(); err != nil {
		t.Fatalf("expected no error with both env vars set, got: %v", err)
	}
}

// TestAuthz_ServedAuthConfigError_Bad asserts a startup error when either the
// bearer token or the distinct signing secret is missing.
func TestAuthz_ServedAuthConfigError_Bad(t *testing.T) {
	t.Setenv("MCP_AUTH_TOKEN", "")
	t.Setenv("MCP_JWT_SECRET", "distinct-signing-key")
	if err := servedAuthConfigError(); err == nil {
		t.Fatal("expected error with empty MCP_AUTH_TOKEN")
	}

	t.Setenv("MCP_AUTH_TOKEN", "test-secret-token")
	t.Setenv("MCP_JWT_SECRET", "")
	if err := servedAuthConfigError(); err == nil {
		t.Fatal("expected error with empty MCP_JWT_SECRET")
	}
}

// TestAuthz_ServedAuthConfigError_Ugly asserts whitespace-only env values are
// treated as unset (trimmed), so they cannot smuggle past the fail-closed gate.
func TestAuthz_ServedAuthConfigError_Ugly(t *testing.T) {
	t.Setenv("MCP_AUTH_TOKEN", "   ")
	t.Setenv("MCP_JWT_SECRET", "distinct-signing-key")
	if err := servedAuthConfigError(); err == nil {
		t.Fatal("expected whitespace-only MCP_AUTH_TOKEN to be rejected")
	}
}

// TestAuthz_CanRunTool_Good asserts per-session entitlement scoping allows a
// tool the claims explicitly grant, and the wildcard grants all (S1.4).
func TestAuthz_CanRunTool_Good(t *testing.T) {
	scoped := &authClaims{Entitlements: []string{"tool:dispatch"}}
	if !scoped.canRunTool("dispatch") {
		t.Error("expected explicitly-entitled tool to be allowed")
	}
	wildcard := &authClaims{Entitlements: []string{"tool:*"}}
	if !wildcard.canRunTool("anything") {
		t.Error("expected wildcard entitlement to allow any tool")
	}
}

// TestAuthz_CanRunTool_Bad asserts a session cannot run a tool outside its
// entitlement set — one session's authority does not leak to another's tools.
func TestAuthz_CanRunTool_Bad(t *testing.T) {
	scoped := &authClaims{Entitlements: []string{"tool:dispatch"}}
	if scoped.canRunTool("brain.forget") {
		t.Error("expected un-entitled tool to be rejected")
	}
}

// TestAuthz_CanRunTool_Ugly asserts a nil claim set is closed, while an empty
// entitlement list is the unscoped (full-access) bearer shape.
func TestAuthz_CanRunTool_Ugly(t *testing.T) {
	var nilClaims *authClaims
	if nilClaims.canRunTool("dispatch") {
		t.Error("expected nil claims to deny all tools")
	}
	unscoped := &authClaims{}
	if !unscoped.canRunTool("dispatch") {
		t.Error("expected empty entitlement list to be unscoped (full access)")
	}
}

// TestAuthz_CanAccessWorkspace_Good asserts a session scoped to a workspace can
// reach inputs within that workspace (exact match and prefix wildcard).
func TestAuthz_CanAccessWorkspace_Good(t *testing.T) {
	exact := &authClaims{Workspace: "core/mcp"}
	if !exact.canAccessWorkspaceFromInput(map[string]any{"workspace": "core/mcp"}) {
		t.Error("expected exact workspace match to be allowed")
	}
	wild := &authClaims{Workspace: "core/*"}
	if !wild.canAccessWorkspaceFromInput(map[string]any{"workspace": "core/agent"}) {
		t.Error("expected prefix wildcard workspace to be allowed")
	}
}

// TestAuthz_CanAccessWorkspace_Bad asserts a session scoped to one workspace
// cannot act on a different workspace's input.
func TestAuthz_CanAccessWorkspace_Bad(t *testing.T) {
	scoped := &authClaims{Workspace: "core/mcp"}
	if scoped.canAccessWorkspaceFromInput(map[string]any{"workspace": "other/repo"}) {
		t.Error("expected mismatched workspace to be rejected")
	}
}

// TestAuthz_CanAccessWorkspace_Ugly asserts an unscoped (empty or "*") claim
// reaches any workspace, and inputs with no workspace field are not blocked.
func TestAuthz_CanAccessWorkspace_Ugly(t *testing.T) {
	unscoped := &authClaims{}
	if !unscoped.canAccessWorkspaceFromInput(map[string]any{"workspace": "anything"}) {
		t.Error("expected unscoped claim to reach any workspace")
	}
	star := &authClaims{Workspace: "*"}
	if !star.canAccessWorkspaceFromInput(map[string]any{"workspace": "anything"}) {
		t.Error("expected * workspace to reach any workspace")
	}
	scoped := &authClaims{Workspace: "core/mcp"}
	if !scoped.canAccessWorkspaceFromInput(map[string]any{"unrelated": "x"}) {
		t.Error("expected input with no workspace field to be allowed (no target to scope against)")
	}
}
