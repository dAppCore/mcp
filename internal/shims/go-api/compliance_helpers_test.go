package api_test

import (
	"net/http"

	. "dappco.re/go"
	api "dappco.re/go/api"
	"github.com/gin-gonic/gin"
)

// moved helpers from ax7_triplets_test.go
type apiGroupForTest struct {
	name string
	base string
}

func (g apiGroupForTest) Name() string     { return g.name }
func (g apiGroupForTest) BasePath() string { return g.base }
func (g apiGroupForTest) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/ping", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) })
}

func (g apiGroupForTest) Describe() []api.RouteDescription {
	return []api.RouteDescription{{Method: http.MethodGet, Path: "/ping", Summary: "Ping", Tags: []string{"ax7"}}}
}
