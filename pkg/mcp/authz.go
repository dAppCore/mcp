// SPDX-License-Identifier: EUPL-1.2

package mcp

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"github.com/goccy/go-json"
	"reflect"
	"strconv"
	"time"

	core "dappco.re/go"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	// authTokenPrefix is the prefix used by HTTP Authorization headers.
	authTokenPrefix = "Bearer "
	// authDefaultJWTTTL is the default validity duration for minted JWTs.
	authDefaultJWTTTL = time.Hour
	// authJWTSecretEnv is the HMAC secret used for JWT signing and verification.
	authJWTSecretEnv = "MCP_JWT_SECRET"
	// authJWTTTLSecondsEnv allows overriding token lifetime.
	authJWTTTLSecondsEnv = "MCP_JWT_TTL_SECONDS"
)

// authClaims is the compact claim payload stored inside our internal JWTs.
type authClaims struct {
	Workspace    string   `json:"workspace,omitempty"`
	Entitlements []string `json:"entitlements,omitempty"`
	Subject      string   `json:"sub,omitempty"`
	Issuer       string   `json:"iss,omitempty"`
	IssuedAt     int64    `json:"iat,omitempty"`
	ExpiresAt    int64    `json:"exp,omitempty"`
}

type authContextKey struct{}

func withAuthClaims(ctx context.Context, claims *authClaims) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return context.WithValue(ctx, authContextKey{}, claims)
}

func claimsFromContext(ctx context.Context) *authClaims {
	if ctx == nil {
		return nil
	}
	if c := ctx.Value(authContextKey{}); c != nil {
		if cl, ok := c.(*authClaims); ok {
			return cl
		}
	}
	return nil
}

// authConfig holds token verification options derived from environment.
type authConfig struct {
	apiToken string
	secret   []byte
	ttl      time.Duration
}

func currentAuthConfig(apiToken string) authConfig {
	cfg := authConfig{
		apiToken: apiToken,
		secret:   []byte(core.Env(authJWTSecretEnv)),
		ttl:      authDefaultJWTTTL,
	}
	if len(cfg.secret) == 0 {
		cfg.secret = []byte(apiToken)
	}
	if ttlRaw := core.Trim(core.Env(authJWTTTLSecondsEnv)); ttlRaw != "" {
		if ttlVal, err := strconv.Atoi(ttlRaw); err == nil && ttlVal > 0 {
			cfg.ttl = time.Duration(ttlVal) * time.Second
		}
	}
	return cfg
}

func extractBearerToken(raw string) string {
	raw = core.Trim(raw)
	if core.HasPrefix(raw, authTokenPrefix) {
		return core.Trim(core.TrimPrefix(raw, authTokenPrefix))
	}
	return ""
}

func parseAuthClaims(authToken, apiToken string) (
	*authClaims,
	error,
) {
	cfg := currentAuthConfig(apiToken)
	if cfg.apiToken == "" {
		return nil, nil
	}
	tkn := extractBearerToken(authToken)
	if tkn == "" {
		return nil, core.E("", "missing bearer token", nil)
	}

	if subtle.ConstantTimeCompare([]byte(tkn), []byte(cfg.apiToken)) == 1 {
		return &authClaims{
			Subject:  "api-key",
			IssuedAt: time.Now().Unix(),
		}, nil
	}

	if len(cfg.secret) == 0 {
		return nil, core.E("", "jwt secret is not configured", nil)
	}

	parts := core.Split(tkn, ".")
	if len(parts) != 3 {
		return nil, core.E("", "invalid token format", nil)
	}

	headerJSON, err := decodeJWTSection(parts[0])
	if err != nil {
		return nil, err
	}
	var header map[string]any
	if err := json.Unmarshal(headerJSON, &header); err != nil {
		return nil, err
	}
	if alg, _ := header["alg"].(string); alg != "" && alg != "HS256" {
		return nil, core.E("", core.Sprintf("unsupported jwt algorithm: %s", alg), nil)
	}

	signatureBase := parts[0] + "." + parts[1]
	mac := hmac.New(sha256.New, cfg.secret)
	mac.Write([]byte(signatureBase))
	expectedSig := mac.Sum(nil)
	actualSig, err := decodeJWTSection(parts[2])
	if err != nil {
		return nil, err
	}
	if !hmac.Equal(expectedSig, actualSig) {
		return nil, core.E("", "invalid token signature", nil)
	}

	payloadJSON, err := decodeJWTSection(parts[1])
	if err != nil {
		return nil, err
	}
	var claims authClaims
	if err := json.Unmarshal(payloadJSON, &claims); err != nil {
		return nil, err
	}

	now := time.Now().Unix()
	if claims.ExpiresAt > 0 && claims.ExpiresAt < now {
		return nil, core.E("", "token has expired", nil)
	}

	return &claims, nil
}

func decodeJWTSection(value string) (
	[]byte,
	error,
) {
	raw, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return nil, err
	}
	return raw, nil
}

func encodeJWTSection(value []byte) string {
	return base64.RawURLEncoding.EncodeToString(value)
}

func mintJWTToken(rawClaims authClaims, cfg authConfig) (
	string,
	error,
) {
	now := time.Now().Unix()
	if rawClaims.IssuedAt == 0 {
		rawClaims.IssuedAt = now
	}
	if rawClaims.ExpiresAt == 0 {
		rawClaims.ExpiresAt = now + int64(cfg.ttl.Seconds())
	}
	header := map[string]string{
		"alg": "HS256",
		"typ": "JWT",
	}
	headerJSON, err := json.Marshal(header)
	if err != nil {
		return "", err
	}
	payloadJSON, err := json.Marshal(rawClaims)
	if err != nil {
		return "", err
	}
	signingInput := encodeJWTSection(headerJSON) + "." + encodeJWTSection(payloadJSON)
	mac := hmac.New(sha256.New, cfg.secret)
	mac.Write([]byte(signingInput))
	signature := mac.Sum(nil)

	return signingInput + "." + encodeJWTSection(signature), nil
}

func authClaimsFromToolRequest(ctx context.Context, req *mcp.CallToolRequest, apiToken string) (
	claims *authClaims,
	inTransport bool,
	err error,
) {
	cfg := currentAuthConfig(apiToken)
	if cfg.apiToken == "" {
		return nil, false, nil
	}
	if req != nil {
		extra := req.GetExtra()
		if extra == nil || extra.Header == nil {
			return nil, true, core.E("", "missing request auth metadata", nil)
		}
		raw := extra.Header.Get("Authorization")
		parsed, err := parseAuthClaims(raw, apiToken)
		if err != nil {
			return nil, true, err
		}
		return parsed, true, nil
	}

	if claims = claimsFromContext(ctx); claims != nil {
		return claims, true, nil
	}

	return nil, false, nil
}

func (s *Service) authorizeToolAccess(
	ctx context.Context,
	req *mcp.CallToolRequest,
	tool string,
	input any,
) (
	_ error, // result
) {
	apiToken := core.Env("MCP_AUTH_TOKEN")
	cfg := currentAuthConfig(apiToken)
	if cfg.apiToken == "" {
		return nil
	}

	claims, inTransport, err := authClaimsFromToolRequest(ctx, req, apiToken)
	if err != nil {
		return core.E("auth", "unauthorized", err)
	}
	if !inTransport {
		// Allow direct service method calls in-process, while still enforcing
		// transport requests where auth metadata is present.
		return nil
	}
	if claims == nil {
		return core.E("auth", "unauthorized", core.E("", "missing auth claims", nil))
	}
	if !claims.canRunTool(tool) {
		return core.E("auth", "forbidden", core.E("", "tool not allowed for token", nil))
	}
	if !claims.canAccessWorkspaceFromInput(input) {
		return core.E("auth", "forbidden", core.E("", "workspace scope mismatch", nil))
	}
	return nil
}

func (c *authClaims) canRunTool(tool string) bool {
	if c == nil {
		return false
	}
	if len(c.Entitlements) == 0 {
		return true
	}
	toolAllow := "tool:" + tool
	for _, e := range c.Entitlements {
		switch e {
		case "*", "tool:*", "tools:*":
			return true
		default:
			if e == tool {
				return true
			}
			if e == toolAllow || e == "tools:"+tool {
				return true
			}
		}
	}
	return false
}

func (c *authClaims) canAccessWorkspaceFromInput(input any) bool {
	if c == nil || c.Workspace == "" || c.Workspace == "*" {
		return true
	}
	target := inputWorkspaceFromValue(input)
	if target == "" {
		return true
	}
	return workspaceMatch(c.Workspace, target)
}

func workspaceMatch(claimed, target string) bool {
	if core.Trim(claimed) == "" {
		return true
	}
	if core.Trim(target) == "" {
		return true
	}
	if claimed == target {
		return true
	}
	if core.HasSuffix(claimed, "*") {
		prefix := core.TrimSuffix(claimed, "*")
		return core.HasPrefix(target, prefix)
	}
	return core.HasPrefix(target, claimed+"/")
}

func inputWorkspaceFromValue(input any) string {
	if input == nil {
		return ""
	}
	v := reflect.ValueOf(input)
	for v.Kind() == reflect.Pointer && !v.IsNil() {
		v = v.Elem()
	}
	if !v.IsValid() {
		return ""
	}

	switch v.Kind() {
	case reflect.Map:
		return workspaceFromMap(v)
	case reflect.Struct:
		return workspaceFromStruct(v)
	default:
		return ""
	}
}

func workspaceFromMap(v reflect.Value) string {
	if v.IsNil() {
		return ""
	}
	keyType := v.Type().Key()
	if keyType.Kind() != reflect.String {
		return ""
	}
	for _, key := range []string{
		"workspace",
		"repo",
		"repository",
		"project",
		"workspace_id",
	} {
		mapKey := reflect.ValueOf(key)
		if mapKey.Type() != keyType {
			if mapKey.Type().ConvertibleTo(keyType) {
				mapKey = mapKey.Convert(keyType)
			} else {
				continue
			}
		}
		if mapKey.IsValid() {
			raw := v.MapIndex(mapKey)
			if raw.IsValid() && raw.Kind() == reflect.String {
				return core.Trim(raw.String())
			}
		}
	}
	return ""
}

func workspaceFromStruct(v reflect.Value) string {
	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		f := v.Field(i)
		ft := t.Field(i)
		if !f.CanInterface() {
			continue
		}

		keys := []string{core.Lower(ft.Name)}
		if tag := ft.Tag.Get(`json`); tag != "" {
			keys = append(keys, core.Lower(core.Split(tag, ",")[0]))
		}
		for _, candidate := range keys {
			if candidate != "workspace" && candidate != "repo" && candidate != "repository" {
				continue
			}
			switch f.Kind() {
			case reflect.String:
				if s := core.Trim(f.String()); s != "" {
					return s
				}
			case reflect.Pointer:
				if f.IsNil() {
					continue
				}
				if f.Elem().Kind() == reflect.String {
					if s := core.Trim(f.Elem().String()); s != "" {
						return s
					}
				}
			}
		}
	}
	return ""
}
