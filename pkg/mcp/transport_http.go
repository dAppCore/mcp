// SPDX-License-Identifier: EUPL-1.2

package mcp

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"strings"
	"time"

	core "dappco.re/go/core"
	coreerr "dappco.re/go/core/log"
	api "dappco.re/go/core/api"
	"github.com/gin-gonic/gin"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// DefaultHTTPAddr is the default address for the MCP HTTP server.
//
//	svc.ServeHTTP(ctx, DefaultHTTPAddr) // "127.0.0.1:9101"
const DefaultHTTPAddr = "127.0.0.1:9101"

// ServeHTTP starts the MCP server with Streamable HTTP transport.
// Supports Bearer token authentication via MCP_AUTH_TOKEN env var.
// If no token is set, authentication is disabled (local development mode).
//
//	// Local development (no auth):
//	svc.ServeHTTP(ctx, "127.0.0.1:9101")
//
//	// Production (with auth):
//	os.Setenv("MCP_AUTH_TOKEN", "sk-abc123")
//	svc.ServeHTTP(ctx, "0.0.0.0:9101")
//
// Endpoint /mcp: GET (SSE stream), POST (JSON-RPC), DELETE (terminate session).
//
// Additional endpoints:
//   - POST /mcp/auth: exchange API token for JWT
//   - /v1/tools/<tool_name>: auto-mounted REST bridge for MCP tools
//   - /health: unauthenticated health endpoint
//   - /.well-known/mcp-servers.json: MCP portal discovery
func (s *Service) ServeHTTP(ctx context.Context, addr string) error {
	if addr == "" {
		addr = DefaultHTTPAddr
	}

	authToken := core.Env("MCP_AUTH_TOKEN")

	handler := mcp.NewStreamableHTTPHandler(
		func(r *http.Request) *mcp.Server {
			return s.server
		},
		&mcp.StreamableHTTPOptions{
			SessionTimeout: 30 * time.Minute,
		},
	)

	toolBridge := api.NewToolBridge("/v1/tools")
	BridgeToAPI(s, toolBridge)
	toolEngine := gin.New()
	toolBridge.RegisterRoutes(toolEngine.Group("/v1/tools"))
	toolHandler := withAuth(authToken, toolEngine)

	mux := http.NewServeMux()
	mux.Handle("/mcp", withAuth(authToken, handler))
	mux.Handle("/v1/tools", toolHandler)
	mux.Handle("/v1/tools/", toolHandler)
	mux.HandleFunc("/mcp/auth", func(w http.ResponseWriter, r *http.Request) {
		serveMCPAuthExchange(w, r, authToken)
	})
	mux.HandleFunc("/.well-known/mcp-servers.json", handleMCPDiscovery)

	// Health check (no auth)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"status": "ok"})
	})

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return coreerr.E("mcp.ServeHTTP", "failed to listen on "+addr, err)
	}
	defer listener.Close()

	diagPrintf("MCP HTTP server listening on %s\n", addr)

	server := &http.Server{Handler: mux}

	// Graceful shutdown on context cancellation
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()

	if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
		return coreerr.E("mcp.ServeHTTP", "server error", err)
	}
	return nil
}

type mcpAuthExchangeRequest struct {
	Token        string   `json:"token"`
	Workspace    string   `json:"workspace"`
	Entitlements []string `json:"entitlements"`
	Sub          string   `json:"sub"`
}

type mcpAuthExchangeResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int64  `json:"expires_in"`
	ExpiresAt   int64  `json:"expires_at"`
}

type mcpDiscoveryResponse struct {
	Servers []mcpDiscoveryServer `json:"servers"`
}

type mcpDiscoveryServer struct {
	ID             string         `json:"id"`
	Name           string         `json:"name"`
	Description    string         `json:"description"`
	Connection     map[string]any `json:"connection"`
	Capabilities   []string       `json:"capabilities"`
	UseWhen        []string       `json:"use_when"`
	RelatedServers []string       `json:"related_servers"`
}

// withAuth wraps an http.Handler with Bearer token authentication.
// If token is empty, authentication is disabled for local development.
func withAuth(token string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if core.Trim(token) == "" {
			next.ServeHTTP(w, r)
			return
		}

		claims, err := parseAuthClaims(r.Header.Get("Authorization"), token)
		if err != nil {
			http.Error(w, `{"error":"invalid token"}`, http.StatusUnauthorized)
			return
		}
		if claims != nil {
			r = r.WithContext(withAuthClaims(r.Context(), claims))
		}
		next.ServeHTTP(w, r)
	})
}

func serveMCPAuthExchange(w http.ResponseWriter, r *http.Request, apiToken string) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	apiToken = core.Trim(apiToken)
	if apiToken == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(api.Fail("unauthorized", "authentication is not configured"))
		return
	}

	var req mcpAuthExchangeRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 10<<20)).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(api.Fail("invalid_request", "invalid JSON payload"))
		return
	}

	providedToken := strings.TrimSpace(extractBearerToken(r.Header.Get("Authorization")))
	if providedToken == "" {
		providedToken = strings.TrimSpace(req.Token)
	}
	if providedToken == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(api.Fail("invalid_request", "missing token"))
		return
	}

	if _, err := parseAuthClaims("Bearer "+providedToken, apiToken); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(api.Fail("unauthorized", "invalid API token"))
		return
	}

	cfg := currentAuthConfig(apiToken)
	now := time.Now()
	claims := authClaims{
		Workspace:    strings.TrimSpace(req.Workspace),
		Entitlements: dedupeEntitlements(req.Entitlements),
		Subject:      core.Trim(req.Sub),
		IssuedAt:     now.Unix(),
		ExpiresAt:     now.Unix() + int64(cfg.ttl.Seconds()),
	}

	minted, err := mintJWTToken(claims, cfg)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(api.Fail("token_error", "failed to mint token"))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(mcpAuthExchangeResponse{
		AccessToken: minted,
		TokenType:   "Bearer",
		ExpiresIn:   int64(cfg.ttl.Seconds()),
		ExpiresAt:   claims.ExpiresAt,
	})
}

func dedupeEntitlements(entitlements []string) []string {
	if len(entitlements) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(entitlements))
	out := make([]string, 0, len(entitlements))
	for _, ent := range entitlements {
		e := strings.TrimSpace(ent)
		if e == "" {
			continue
		}
		if _, ok := seen[e]; ok {
			continue
		}
		seen[e] = struct{}{}
		out = append(out, e)
	}
	return out
}

func handleMCPDiscovery(w http.ResponseWriter, r *http.Request) {
	resp := mcpDiscoveryResponse{
		Servers: []mcpDiscoveryServer{
			{
				ID:          "core-agent",
				Name:        "Core Agent",
				Description: "Dispatch agents, manage workspaces, search OpenBrain",
				Connection: map[string]any{
					"type":    "stdio",
					"command": "core-agent",
					"args":    []string{"mcp"},
				},
				Capabilities: []string{"tools", "resources"},
				UseWhen: []string{
					"Need to dispatch work to Codex/Claude/Gemini",
					"Need workspace status",
					"Need semantic search",
				},
				RelatedServers: []string{"core-mcp"},
			},
			{
				ID:          "core-mcp",
				Name:        "Core MCP",
				Description: "File ops, process and build tools, RAG search, webview, dashboards — the agent-facing MCP framework.",
				Connection: map[string]any{
					"type":    "stdio",
					"command": "core-mcp",
				},
				Capabilities: []string{"tools", "resources", "logging"},
				UseWhen: []string{
					"Need to read/write files inside a workspace",
					"Need to start or monitor processes",
					"Need to run RAG queries or index documents",
					"Need to render or update an embedded dashboard view",
				},
				RelatedServers: []string{"core-agent"},
			},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(api.Fail("server_error", "failed to encode discovery payload"))
	}
}
