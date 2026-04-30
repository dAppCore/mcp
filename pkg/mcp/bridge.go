// SPDX-License-Identifier: EUPL-1.2

package mcp

import (
	"net/http"

	core "dappco.re/go"
	"github.com/gin-gonic/gin"
)

// maxBodySize is the maximum request body size accepted by bridged tool endpoints.
const maxBodySize = 10 << 20 // 10 MB

type restError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type restResponse struct {
	Success bool       `json:"success"`
	Data    any        `json:"data,omitempty"`
	Error   *restError `json:"error,omitempty"`
}

func restOK(data any) restResponse {
	return restResponse{Success: true, Data: data}
}

func restFail(code, message string) restResponse {
	return restResponse{
		Success: false,
		Error:   &restError{Code: code, Message: message},
	}
}

// BridgeToAPI mounts recorded MCP tools as Gin POST endpoints.
// The historical name is retained for callers, but this bridge is owned by
// mcp and does not depend on the go-api gateway package. Each route reads a
// JSON body, dispatches to the tool's RESTHandler, and wraps the result in a
// small local response envelope.
//
//	router := gin.New()
//	mcp.BridgeToAPI(svc, router.Group("/v1/tools"))
func BridgeToAPI(svc *Service, rg *gin.RouterGroup) {
	if svc == nil || rg == nil {
		return
	}

	for rec := range svc.ToolsSeq() {
		name := rec.Name
		handler := rec.RESTHandler

		rg.POST("/"+name, func(c *gin.Context) {
			var body []byte
			if c.Request.Body != nil {
				c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBodySize)
				r := core.ReadAll(c.Request.Body)
				if !r.OK {
					if err, ok := r.Value.(error); ok {
						var maxBytesErr *http.MaxBytesError
						if core.As(err, &maxBytesErr) || core.Contains(err.Error(), "request body too large") {
							c.JSON(http.StatusRequestEntityTooLarge, restFail("request_too_large", "Request body exceeds 10 MB limit"))
							return
						}
					}
					c.JSON(http.StatusBadRequest, restFail("invalid_request", "Failed to read request body"))
					return
				}
				body = []byte(r.Value.(string))
			}

			result, err := handler(c.Request.Context(), body)
			if err != nil {
				// Body present + error = likely bad input (malformed JSON).
				// No body + error = tool execution failure.
				if core.Is(err, errInvalidRESTInput) {
					c.JSON(http.StatusBadRequest, restFail("invalid_input", "Malformed JSON in request body"))
					return
				}
				c.JSON(http.StatusInternalServerError, restFail("tool_error", "Tool execution failed"))
				return
			}

			c.JSON(http.StatusOK, restOK(result))
		})
	}
}
