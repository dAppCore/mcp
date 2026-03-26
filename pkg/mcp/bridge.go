// SPDX-License-Identifier: EUPL-1.2

package mcp

import (
	"net/http"

	core "dappco.re/go/core"
	"github.com/gin-gonic/gin"

	api "forge.lthn.ai/core/api"
)

// maxBodySize is the maximum request body size accepted by bridged tool endpoints.
const maxBodySize = 10 << 20 // 10 MB

// BridgeToAPI populates a go-api ToolBridge from recorded MCP tools.
// Each tool becomes a POST endpoint that reads a JSON body, dispatches
// to the tool's RESTHandler (which knows the concrete input type), and
// wraps the result in the standard api.Response envelope.
//
//	bridge := api.NewToolBridge()
//	mcp.BridgeToAPI(svc, bridge)
//	bridge.Mount(router, "/v1/tools")
func BridgeToAPI(svc *Service, bridge *api.ToolBridge) {
	for rec := range svc.ToolsSeq() {
		desc := api.ToolDescriptor{
			Name:         rec.Name,
			Description:  rec.Description,
			Group:        rec.Group,
			InputSchema:  rec.InputSchema,
			OutputSchema: rec.OutputSchema,
		}

		// Capture the handler for the closure.
		handler := rec.RESTHandler

		bridge.Add(desc, func(c *gin.Context) {
			var body []byte
			if c.Request.Body != nil {
				r := core.ReadAll(c.Request.Body)
				if !r.OK {
					c.JSON(http.StatusBadRequest, api.Fail("invalid_request", "Failed to read request body"))
					return
				}
				body = []byte(r.Value.(string))
			}

			result, err := handler(c.Request.Context(), body)
			if err != nil {
				// Body present + error = likely bad input (malformed JSON).
				// No body + error = tool execution failure.
				if len(body) > 0 && core.Contains(err.Error(), "unmarshal") {
					c.JSON(http.StatusBadRequest, api.Fail("invalid_input", "Malformed JSON in request body"))
					return
				}
				c.JSON(http.StatusInternalServerError, api.Fail("tool_error", "Tool execution failed"))
				return
			}

			c.JSON(http.StatusOK, api.OK(result))
		})
	}
}
