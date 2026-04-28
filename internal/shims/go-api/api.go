package api

import (
	"net/http"
	"path"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
)

type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type Response[T any] struct {
	Success bool   `json:"success"`
	Data    T      `json:"data,omitempty"`
	Error   *Error `json:"error,omitempty"`
}

func OK[T any](data T) Response[T] {
	return Response[T]{Success: true, Data: data}
}

func Fail(code, message string) Response[any] {
	return Response[any]{
		Success: false,
		Error:   &Error{Code: code, Message: message},
	}
}

type RouteDescription struct {
	Method      string
	Path        string
	Summary     string
	Description string
	Tags        []string
	RequestBody any
	Response    any
}

type RouteGroup interface {
	Name() string
	BasePath() string
	RegisterRoutes(*gin.RouterGroup)
}

type DescribableGroup interface {
	Describe() []RouteDescription
}

type ToolDescriptor struct {
	Name         string
	Description  string
	Group        string
	InputSchema  map[string]any
	OutputSchema map[string]any
}

type ToolBridge struct {
	basePath string
	mu       sync.RWMutex
	tools    []ToolDescriptor
	handlers map[string]func(*gin.Context)
}

func NewToolBridge(basePath ...string) *ToolBridge {
	base := "/tools"
	if len(basePath) > 0 && strings.TrimSpace(basePath[0]) != "" {
		base = basePath[0]
	}
	if !strings.HasPrefix(base, "/") {
		base = "/" + base
	}
	return &ToolBridge{
		basePath: path.Clean(base),
		handlers: make(map[string]func(*gin.Context)),
	}
}

func (b *ToolBridge) Name() string { return "tools" }

func (b *ToolBridge) BasePath() string { return b.basePath }

func (b *ToolBridge) Add(desc ToolDescriptor, handler func(*gin.Context)) {
	if desc.Name == "" {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	b.tools = append(b.tools, desc)
	b.handlers[desc.Name] = handler
}

func (b *ToolBridge) Tools() []ToolDescriptor {
	b.mu.RLock()
	defer b.mu.RUnlock()
	out := make([]ToolDescriptor, len(b.tools))
	copy(out, b.tools)
	return out
}

func (b *ToolBridge) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("", b.listTools)
	rg.GET("/", b.listTools)
	for _, tool := range b.Tools() {
		name := tool.Name
		rg.POST("/"+name, func(c *gin.Context) {
			b.mu.RLock()
			handler := b.handlers[name]
			b.mu.RUnlock()
			if handler == nil {
				c.JSON(http.StatusInternalServerError, Fail("tool_unavailable", "tool handler unavailable"))
				return
			}
			handler(c)
		})
	}
}

func (b *ToolBridge) Describe() []RouteDescription {
	tools := b.Tools()
	descs := make([]RouteDescription, 0, len(tools)+1)
	descs = append(descs, RouteDescription{
		Method:      http.MethodGet,
		Path:        "/",
		Summary:     "List bridged tools",
		Description: "List MCP tools exposed through the REST bridge.",
		Tags:        []string{"tools"},
	})
	for _, tool := range tools {
		group := tool.Group
		if group == "" {
			group = "tools"
		}
		descs = append(descs, RouteDescription{
			Method:      http.MethodPost,
			Path:        "/" + tool.Name,
			Summary:     tool.Name,
			Description: tool.Description,
			Tags:        []string{group},
			RequestBody: tool.InputSchema,
			Response:    tool.OutputSchema,
		})
	}
	return descs
}

func (b *ToolBridge) listTools(c *gin.Context) {
	c.JSON(http.StatusOK, OK(b.Tools()))
}

type Engine struct {
	router  *gin.Engine
	groups  []RouteGroup
	swagger *swaggerInfo
}

type swaggerInfo struct {
	Title       string
	Description string
	Version     string
}

type Option func(*Engine)

func WithSwagger(title, description, version string) Option {
	return func(e *Engine) {
		e.swagger = &swaggerInfo{Title: title, Description: description, Version: version}
	}
}

func New(opts ...Option) (*Engine, error) {
	e := &Engine{router: gin.New()}
	for _, opt := range opts {
		if opt != nil {
			opt(e)
		}
	}
	e.router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	if e.swagger != nil {
		e.router.GET("/swagger/doc.json", func(c *gin.Context) {
			c.JSON(http.StatusOK, e.swaggerDocument())
		})
	}
	return e, nil
}

func (e *Engine) Register(group RouteGroup) {
	if group == nil {
		return
	}
	e.groups = append(e.groups, group)
	group.RegisterRoutes(e.router.Group(group.BasePath()))
}

func (e *Engine) Handler() http.Handler { return e.router }

func (e *Engine) swaggerDocument() map[string]any {
	info := map[string]any{}
	if e.swagger != nil {
		info["title"] = e.swagger.Title
		info["description"] = e.swagger.Description
		info["version"] = e.swagger.Version
	}
	paths := map[string]any{}
	for _, group := range e.groups {
		describable, ok := group.(DescribableGroup)
		if !ok {
			continue
		}
		base := group.BasePath()
		for _, desc := range describable.Describe() {
			fullPath := joinRoute(base, desc.Path)
			method := strings.ToLower(desc.Method)
			if method == "" {
				method = "get"
			}
			item, _ := paths[fullPath].(map[string]any)
			if item == nil {
				item = map[string]any{}
				paths[fullPath] = item
			}
			item[method] = map[string]any{
				"summary":     desc.Summary,
				"description": desc.Description,
				"tags":        desc.Tags,
				"responses": map[string]any{
					"200": map[string]any{"description": "OK"},
				},
			}
		}
	}
	return map[string]any{
		"openapi": "3.0.0",
		"info":    info,
		"paths":   paths,
	}
}

func joinRoute(base, rel string) string {
	if rel == "" || rel == "/" {
		return path.Clean(base)
	}
	return path.Clean(strings.TrimRight(base, "/") + "/" + strings.TrimLeft(rel, "/"))
}
