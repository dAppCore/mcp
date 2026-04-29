package api_test

import (
	"net/http"
	"net/http/httptest"

	. "dappco.re/go"
	api "dappco.re/go/api"
	"github.com/gin-gonic/gin"
)

// moved AX-7 triplet TestApi_OK_Good
func TestApi_OK_Good(t *T) {
	resp := api.OK(map[string]string{"id": "1"})
	AssertTrue(t, resp.Success)
	AssertEqual(t, "1", resp.Data["id"])
	AssertNil(t, resp.Error)
}

// moved AX-7 triplet TestApi_OK_Bad
func TestApi_OK_Bad(t *T) {
	resp := api.OK[any](nil)
	AssertTrue(t, resp.Success)
	AssertNil(t, resp.Data)
}

// moved AX-7 triplet TestApi_OK_Ugly
func TestApi_OK_Ugly(t *T) {
	resp := api.OK(map[string]any{})
	AssertTrue(t, resp.Success)
	AssertEqual(t, 0, len(resp.Data))
	AssertNil(t, resp.Error)
}

// moved AX-7 triplet TestApi_Fail_Good
func TestApi_Fail_Good(t *T) {
	resp := api.Fail("invalid", "bad input")
	AssertFalse(t, resp.Success)
	AssertEqual(t, "invalid", resp.Error.Code)
}

// moved AX-7 triplet TestApi_Fail_Bad
func TestApi_Fail_Bad(t *T) {
	resp := api.Fail("", "")
	AssertFalse(t, resp.Success)
	AssertEqual(t, "", resp.Error.Code)
	AssertEqual(t, "", resp.Error.Message)
}

// moved AX-7 triplet TestApi_Fail_Ugly
func TestApi_Fail_Ugly(t *T) {
	resp := api.Fail("x/y", "line\nbreak")
	AssertFalse(t, resp.Success)
	AssertContains(t, resp.Error.Message, "line")
}

// moved AX-7 triplet TestApi_NewToolBridge_Good
func TestApi_NewToolBridge_Good(t *T) {
	bridge := api.NewToolBridge("/v1/tools")
	AssertEqual(t, "tools", bridge.Name())
	AssertEqual(t, "/v1/tools", bridge.BasePath())
}

// moved AX-7 triplet TestApi_NewToolBridge_Bad
func TestApi_NewToolBridge_Bad(t *T) {
	bridge := api.NewToolBridge("")
	AssertEqual(t, "tools", bridge.Name())
	AssertEqual(t, "/tools", bridge.BasePath())
}

// moved AX-7 triplet TestApi_NewToolBridge_Ugly
func TestApi_NewToolBridge_Ugly(t *T) {
	bridge := api.NewToolBridge("tools/../tools")
	AssertEqual(t, "tools", bridge.Name())
	AssertEqual(t, "/tools", bridge.BasePath())
}

// moved AX-7 triplet TestApi_ToolBridge_Name_Good
func TestApi_ToolBridge_Name_Good(t *T) {
	bridge := api.NewToolBridge()
	AssertNotNil(t, bridge)
	AssertEqual(t, "tools", bridge.Name())
}

// moved AX-7 triplet TestApi_ToolBridge_Name_Bad
func TestApi_ToolBridge_Name_Bad(t *T) {
	bridge := api.NewToolBridge("")
	AssertEqual(t, "/tools", bridge.BasePath())
	AssertEqual(t, "tools", bridge.Name())
}

// moved AX-7 triplet TestApi_ToolBridge_Name_Ugly
func TestApi_ToolBridge_Name_Ugly(t *T) {
	var bridge *api.ToolBridge
	AssertEqual(t, "tools", bridge.Name())
	AssertNil(t, bridge)
}

// moved AX-7 triplet TestApi_ToolBridge_BasePath_Good
func TestApi_ToolBridge_BasePath_Good(t *T) {
	bridge := api.NewToolBridge("custom")
	AssertEqual(t, "tools", bridge.Name())
	AssertEqual(t, "/custom", bridge.BasePath())
}

// moved AX-7 triplet TestApi_ToolBridge_BasePath_Bad
func TestApi_ToolBridge_BasePath_Bad(t *T) {
	var bridge *api.ToolBridge
	AssertNil(t, bridge)
	AssertPanics(t, func() { _ = bridge.BasePath() })
}

// moved AX-7 triplet TestApi_ToolBridge_BasePath_Ugly
func TestApi_ToolBridge_BasePath_Ugly(t *T) {
	bridge := api.NewToolBridge("")
	AssertEqual(t, "tools", bridge.Name())
	AssertEqual(t, "/tools", bridge.BasePath())
}

// moved AX-7 triplet TestApi_ToolBridge_Add_Good
func TestApi_ToolBridge_Add_Good(t *T) {
	bridge := api.NewToolBridge()
	bridge.Add(api.ToolDescriptor{Name: "echo", Description: "Echo"}, func(*gin.Context) {})
	AssertLen(t, bridge.Tools(), 1)
}

// moved AX-7 triplet TestApi_ToolBridge_Add_Bad
func TestApi_ToolBridge_Add_Bad(t *T) {
	bridge := api.NewToolBridge()
	bridge.Add(api.ToolDescriptor{}, func(*gin.Context) {})
	AssertLen(t, bridge.Tools(), 0)
}

// moved AX-7 triplet TestApi_ToolBridge_Add_Ugly
func TestApi_ToolBridge_Add_Ugly(t *T) {
	bridge := api.NewToolBridge()
	bridge.Add(api.ToolDescriptor{Name: "nil_handler"}, nil)
	AssertLen(t, bridge.Tools(), 1)
}

// moved AX-7 triplet TestApi_ToolBridge_Tools_Good
func TestApi_ToolBridge_Tools_Good(t *T) {
	bridge := api.NewToolBridge()
	bridge.Add(api.ToolDescriptor{Name: "echo"}, func(*gin.Context) {})
	AssertEqual(t, "echo", bridge.Tools()[0].Name)
}

// moved AX-7 triplet TestApi_ToolBridge_Tools_Bad
func TestApi_ToolBridge_Tools_Bad(t *T) {
	bridge := api.NewToolBridge()
	AssertLen(t, bridge.Tools(), 0)
	AssertEqual(t, "tools", bridge.Name())
}

// moved AX-7 triplet TestApi_ToolBridge_Tools_Ugly
func TestApi_ToolBridge_Tools_Ugly(t *T) {
	bridge := api.NewToolBridge()
	bridge.Add(api.ToolDescriptor{Name: "echo"}, func(*gin.Context) {})
	tools := bridge.Tools()
	tools[0].Name = "mutated"
	AssertEqual(t, "echo", bridge.Tools()[0].Name)
}

// moved AX-7 triplet TestApi_ToolBridge_RegisterRoutes_Good
func TestApi_ToolBridge_RegisterRoutes_Good(t *T) {
	gin.SetMode(gin.TestMode)
	bridge := api.NewToolBridge("/tools")
	bridge.Add(api.ToolDescriptor{Name: "echo"}, func(c *gin.Context) { c.JSON(http.StatusOK, api.OK("ok")) })
	router := gin.New()
	bridge.RegisterRoutes(router.Group("/tools"))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/tools/echo", nil)
	router.ServeHTTP(rr, req)
	AssertEqual(t, http.StatusOK, rr.Code)
}

// moved AX-7 triplet TestApi_ToolBridge_RegisterRoutes_Bad
func TestApi_ToolBridge_RegisterRoutes_Bad(t *T) {
	gin.SetMode(gin.TestMode)
	bridge := api.NewToolBridge("/tools")
	bridge.Add(api.ToolDescriptor{Name: "missing"}, nil)
	router := gin.New()
	bridge.RegisterRoutes(router.Group("/tools"))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/tools/missing", nil)
	router.ServeHTTP(rr, req)
	AssertEqual(t, http.StatusInternalServerError, rr.Code)
}

// moved AX-7 triplet TestApi_ToolBridge_RegisterRoutes_Ugly
func TestApi_ToolBridge_RegisterRoutes_Ugly(t *T) {
	gin.SetMode(gin.TestMode)
	bridge := api.NewToolBridge("/tools")
	router := gin.New()
	bridge.RegisterRoutes(router.Group("/tools"))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/tools/", nil)
	router.ServeHTTP(rr, req)
	AssertEqual(t, http.StatusOK, rr.Code)
}

// moved AX-7 triplet TestApi_ToolBridge_Describe_Good
func TestApi_ToolBridge_Describe_Good(t *T) {
	bridge := api.NewToolBridge()
	bridge.Add(api.ToolDescriptor{Name: "echo", Group: "demo", Description: "Echo"}, func(*gin.Context) {})
	descs := bridge.Describe()
	AssertLen(t, descs, 2)
	AssertEqual(t, "/echo", descs[1].Path)
}

// moved AX-7 triplet TestApi_ToolBridge_Describe_Bad
func TestApi_ToolBridge_Describe_Bad(t *T) {
	descs := api.NewToolBridge().Describe()
	AssertLen(t, descs, 1)
	AssertEqual(t, "/", descs[0].Path)
}

// moved AX-7 triplet TestApi_ToolBridge_Describe_Ugly
func TestApi_ToolBridge_Describe_Ugly(t *T) {
	bridge := api.NewToolBridge()
	bridge.Add(api.ToolDescriptor{Name: "echo"}, func(*gin.Context) {})
	AssertEqual(t, []string{"tools"}, bridge.Describe()[1].Tags)
}

// moved AX-7 triplet TestApi_WithSwagger_Good
func TestApi_WithSwagger_Good(t *T) {
	engine, err := api.New(api.WithSwagger("Title", "Desc", "v1"))
	AssertNoError(t, err)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/swagger/doc.json", nil)
	engine.Handler().ServeHTTP(rr, req)
	AssertEqual(t, http.StatusOK, rr.Code)
}

// moved AX-7 triplet TestApi_WithSwagger_Bad
func TestApi_WithSwagger_Bad(t *T) {
	engine, err := api.New(api.WithSwagger("", "", ""))
	AssertNoError(t, err)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/swagger/doc.json", nil)
	engine.Handler().ServeHTTP(rr, req)
	AssertEqual(t, http.StatusOK, rr.Code)
}

// moved AX-7 triplet TestApi_WithSwagger_Ugly
func TestApi_WithSwagger_Ugly(t *T) {
	engine, err := api.New(nil, api.WithSwagger("T", "D", "V"))
	AssertNoError(t, err)
	AssertNotNil(t, engine.Handler())
}

// moved AX-7 triplet TestApi_New_Good
func TestApi_New_Good(t *T) {
	engine, err := api.New()
	AssertNoError(t, err)
	AssertNotNil(t, engine.Handler())
}

// moved AX-7 triplet TestApi_New_Bad
func TestApi_New_Bad(t *T) {
	AssertPanics(t, func() {
		_, _ = api.New(func(*api.Engine) { panic("bad option") })
	})
}

// moved AX-7 triplet TestApi_New_Ugly
func TestApi_New_Ugly(t *T) {
	engine, err := api.New(nil)
	AssertNoError(t, err)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	engine.Handler().ServeHTTP(rr, req)
	AssertEqual(t, http.StatusOK, rr.Code)
}

// moved AX-7 triplet TestApi_Engine_Register_Good
func TestApi_Engine_Register_Good(t *T) {
	engine, err := api.New()
	AssertNoError(t, err)
	engine.Register(apiGroupForTest{name: "ax7", base: "/ax7"})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ax7/ping", nil)
	engine.Handler().ServeHTTP(rr, req)
	AssertEqual(t, http.StatusOK, rr.Code)
}

// moved AX-7 triplet TestApi_Engine_Register_Bad
func TestApi_Engine_Register_Bad(t *T) {
	engine, err := api.New()
	AssertNoError(t, err)
	engine.Register(nil)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/missing", nil)
	engine.Handler().ServeHTTP(rr, req)
	AssertEqual(t, http.StatusNotFound, rr.Code)
}

// moved AX-7 triplet TestApi_Engine_Register_Ugly
func TestApi_Engine_Register_Ugly(t *T) {
	engine, err := api.New(api.WithSwagger("T", "D", "V"))
	AssertNoError(t, err)
	engine.Register(apiGroupForTest{name: "ax7", base: "ax7"})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/swagger/doc.json", nil)
	engine.Handler().ServeHTTP(rr, req)
	AssertEqual(t, http.StatusOK, rr.Code)
}

// moved AX-7 triplet TestApi_Engine_Handler_Good
func TestApi_Engine_Handler_Good(t *T) {
	engine, err := api.New()
	AssertNoError(t, err)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	engine.Handler().ServeHTTP(rr, req)
	AssertEqual(t, http.StatusOK, rr.Code)
}

// moved AX-7 triplet TestApi_Engine_Handler_Bad
func TestApi_Engine_Handler_Bad(t *T) {
	var engine *api.Engine
	AssertNil(t, engine)
	AssertPanics(t, func() { _ = engine.Handler() })
}

// moved AX-7 triplet TestApi_Engine_Handler_Ugly
func TestApi_Engine_Handler_Ugly(t *T) {
	engine, err := api.New()
	AssertNoError(t, err)
	AssertNotNil(t, engine.Handler())
	AssertNotNil(t, engine.Handler())
}
