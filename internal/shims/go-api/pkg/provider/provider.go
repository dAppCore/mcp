package provider

import (
	"dappco.re/go/api"
	"github.com/gin-gonic/gin"
)

type Provider interface {
	Name() string
	BasePath() string
	RegisterRoutes(*gin.RouterGroup)
}

type Streamable interface {
	Channels() []string
}

type Describable interface {
	Describe() []api.RouteDescription
}

type ElementSpec struct {
	Tag    string `json:"tag"`
	Source string `json:"source"`
}

type Renderable interface {
	Element() ElementSpec
}
