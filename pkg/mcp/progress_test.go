// SPDX-License-Identifier: EUPL-1.2

package mcp

import (
	"context"
	"testing"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestProgressTokenFromRequest_Good_ExtractsMetaToken(t *testing.T) {
	req := &sdkmcp.CallToolRequest{Params: &sdkmcp.CallToolParamsRaw{}}
	req.Params.SetProgressToken("dispatch-123")

	if got := ProgressTokenFromRequest(req); got != "dispatch-123" {
		t.Fatalf("expected progress token dispatch-123, got %v", got)
	}
}

func TestProgressTokenFromRequest_Good_NilSafe(t *testing.T) {
	if got := ProgressTokenFromRequest(nil); got != nil {
		t.Fatalf("expected nil token from nil request, got %v", got)
	}
	req := &sdkmcp.CallToolRequest{}
	if got := ProgressTokenFromRequest(req); got != nil {
		t.Fatalf("expected nil token from request without params, got %v", got)
	}
}

func TestSendProgressNotification_Good_NoopsWithoutSession(t *testing.T) {
	req := &sdkmcp.CallToolRequest{Params: &sdkmcp.CallToolParamsRaw{}}
	req.Params.SetProgressToken("process-1")

	if err := SendProgressNotification(context.Background(), req, 1, 2, "started"); err != nil {
		t.Fatalf("expected no-op without session, got error: %v", err)
	}

	notifier := NewProgressNotifier(context.Background(), req)
	if err := notifier.Send(2, 2, "done"); err != nil {
		t.Fatalf("expected no-op notifier without session, got error: %v", err)
	}
}
