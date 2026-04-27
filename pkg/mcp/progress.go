// SPDX-License-Identifier: EUPL-1.2

package mcp

import (
	"context"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

// ProgressTokenFromRequest extracts _meta.progressToken from an MCP tool call.
func ProgressTokenFromRequest(req *sdkmcp.CallToolRequest) any {
	if req == nil || req.Params == nil {
		return nil
	}
	return req.Params.GetProgressToken()
}

// SendProgressNotification emits notifications/progress when the caller supplied
// _meta.progressToken. Calls without a token or MCP session are no-ops.
func SendProgressNotification(ctx context.Context, req *sdkmcp.CallToolRequest, progress float64, total float64, message string) error {
	token := ProgressTokenFromRequest(req)
	if req == nil || req.Session == nil || token == nil {
		return nil
	}
	return req.Session.NotifyProgress(ctx, &sdkmcp.ProgressNotificationParams{
		ProgressToken: token,
		Progress:      progress,
		Total:         total,
		Message:       message,
	})
}

// ProgressNotifier caches the request progress token for multi-step tools.
type ProgressNotifier struct {
	ctx   context.Context
	req   *sdkmcp.CallToolRequest
	token any
}

// NewProgressNotifier prepares repeated notifications for a single tool call.
func NewProgressNotifier(ctx context.Context, req *sdkmcp.CallToolRequest) ProgressNotifier {
	return ProgressNotifier{
		ctx:   ctx,
		req:   req,
		token: ProgressTokenFromRequest(req),
	}
}

// Send emits a progress notification when the tool call includes a token.
func (n ProgressNotifier) Send(progress float64, total float64, message string) error {
	if n.req == nil || n.req.Session == nil || n.token == nil {
		return nil
	}
	return n.req.Session.NotifyProgress(n.ctx, &sdkmcp.ProgressNotificationParams{
		ProgressToken: n.token,
		Progress:      progress,
		Total:         total,
		Message:       message,
	})
}
