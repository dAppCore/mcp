// SPDX-License-Identifier: EUPL-1.2

package brain

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"dappco.re/go/mcp/pkg/mcp/ide"
	"github.com/gin-gonic/gin"
)

// routerForProvider mounts a BrainProvider's routes under /api/brain.
func routerForProvider(p *BrainProvider) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	p.RegisterRoutes(router.Group("/api/brain"))
	return router
}

// disconnectedBridge returns a Bridge with no live connection: Send always
// errors, exercising the bridge_error (500) branch of each handler.
func disconnectedBridge() *ide.Bridge {
	return ide.NewBridge(nil, ide.DefaultConfig())
}

func doReq(router *gin.Engine, method, path, body string) *httptest.ResponseRecorder {
	rr := httptest.NewRecorder()
	var req *http.Request
	if body == "" {
		req = httptest.NewRequest(method, path, nil)
	} else {
		req = httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	}
	router.ServeHTTP(rr, req)
	return rr
}

func TestProviderHandlers_remember_Ugly_NilBridge503(t *testing.T) {
	router := routerForProvider(NewProvider(nil, nil))
	rr := doReq(router, http.MethodPost, "/api/brain/remember", `{"content":"x","type":"bug"}`)
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 with nil bridge, got %d (%s)", rr.Code, rr.Body.String())
	}
}

func TestProviderHandlers_remember_Bad_MalformedJSON400(t *testing.T) {
	router := routerForProvider(NewProvider(disconnectedBridge(), nil))
	rr := doReq(router, http.MethodPost, "/api/brain/remember", `{not valid json`)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for malformed JSON, got %d (%s)", rr.Code, rr.Body.String())
	}
}

func TestProviderHandlers_remember_Good_BridgeNotConnected500(t *testing.T) {
	router := routerForProvider(NewProvider(disconnectedBridge(), nil))
	rr := doReq(router, http.MethodPost, "/api/brain/remember", `{"content":"hello","type":"note","org":"core"}`)
	// Valid input parses, then Send fails because the bridge has no connection.
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 bridge_error, got %d (%s)", rr.Code, rr.Body.String())
	}
}

func TestProviderHandlers_recall_Ugly_NilBridge503(t *testing.T) {
	router := routerForProvider(NewProvider(nil, nil))
	rr := doReq(router, http.MethodPost, "/api/brain/recall", `{"query":"x"}`)
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", rr.Code)
	}
}

func TestProviderHandlers_recall_Bad_MalformedJSON400(t *testing.T) {
	router := routerForProvider(NewProvider(disconnectedBridge(), nil))
	rr := doReq(router, http.MethodPost, "/api/brain/recall", `[`)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestProviderHandlers_recall_Good_BridgeNotConnected500(t *testing.T) {
	router := routerForProvider(NewProvider(disconnectedBridge(), nil))
	rr := doReq(router, http.MethodPost, "/api/brain/recall", `{"query":"what do we know","top_k":5}`)
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d (%s)", rr.Code, rr.Body.String())
	}
}

func TestProviderHandlers_forget_Ugly_NilBridge503(t *testing.T) {
	router := routerForProvider(NewProvider(nil, nil))
	rr := doReq(router, http.MethodPost, "/api/brain/forget", `{"id":"m1"}`)
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", rr.Code)
	}
}

func TestProviderHandlers_forget_Bad_MalformedJSON400(t *testing.T) {
	router := routerForProvider(NewProvider(disconnectedBridge(), nil))
	rr := doReq(router, http.MethodPost, "/api/brain/forget", `{`)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestProviderHandlers_forget_Good_BridgeNotConnected500(t *testing.T) {
	router := routerForProvider(NewProvider(disconnectedBridge(), nil))
	rr := doReq(router, http.MethodPost, "/api/brain/forget", `{"id":"mem-123","reason":"stale"}`)
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d (%s)", rr.Code, rr.Body.String())
	}
}

func TestProviderHandlers_list_Ugly_NilBridge503(t *testing.T) {
	router := routerForProvider(NewProvider(nil, nil))
	rr := doReq(router, http.MethodGet, "/api/brain/list?project=core/mcp", "")
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", rr.Code)
	}
}

func TestProviderHandlers_list_Good_BridgeNotConnected500(t *testing.T) {
	router := routerForProvider(NewProvider(disconnectedBridge(), nil))
	rr := doReq(router, http.MethodGet, "/api/brain/list?project=core/mcp&org=core&type=note&limit=10", "")
	// GET handler reads query params then Send fails on the dead bridge.
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d (%s)", rr.Code, rr.Body.String())
	}
}
