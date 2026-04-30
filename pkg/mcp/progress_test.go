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

// moved AX-7 triplet TestProgress_NewProgressNotifier_Good
func TestProgress_NewProgressNotifier_Good(t *T) {
	req := &sdkmcp.CallToolRequest{}
	notifier := NewProgressNotifier(context.Background(), req)
	AssertNoError(t, notifier.Send(1, 2, "ok"))
}

// moved AX-7 triplet TestProgress_NewProgressNotifier_Bad
func TestProgress_NewProgressNotifier_Bad(t *T) {
	notifier := NewProgressNotifier(context.Background(), nil)
	AssertNoError(t, notifier.Send(1, 2, "ok"))
	AssertNil(t, notifier.req)
}

// moved AX-7 triplet TestProgress_NewProgressNotifier_Ugly
func TestProgress_NewProgressNotifier_Ugly(t *T) {
	notifier := NewProgressNotifier(nil, &sdkmcp.CallToolRequest{})
	AssertNoError(t, notifier.Send(-1, 0, ""))
	AssertNotNil(t, notifier.req)
}

// moved AX-7 triplet TestProgress_ProgressNotifier_Send_Good
func TestProgress_ProgressNotifier_Send_Good(t *T) {
	notifier := NewProgressNotifier(context.Background(), &sdkmcp.CallToolRequest{})
	err := notifier.Send(1, 2, "ok")
	AssertNoError(t, err)
}

// moved AX-7 triplet TestProgress_ProgressNotifier_Send_Bad
func TestProgress_ProgressNotifier_Send_Bad(t *T) {
	notifier := ProgressNotifier{}
	err := notifier.Send(1, 2, "ok")
	AssertNoError(t, err)
}

// moved AX-7 triplet TestProgress_ProgressNotifier_Send_Ugly
func TestProgress_ProgressNotifier_Send_Ugly(t *T) {
	notifier := NewProgressNotifier(nil, nil)
	err := notifier.Send(-1, 0, "")
	AssertNoError(t, err)
}

// moved AX-7 triplet TestProgress_ProgressTokenFromRequest_Good
func TestProgress_ProgressTokenFromRequest_Good(t *T) {
	req := &sdkmcp.CallToolRequest{Params: &sdkmcp.CallToolParamsRaw{}}
	req.Params.SetProgressToken("tok")
	AssertEqual(t, "tok", ProgressTokenFromRequest(req))
}

// moved AX-7 triplet TestProgress_ProgressTokenFromRequest_Bad
func TestProgress_ProgressTokenFromRequest_Bad(t *T) {
	var req *sdkmcp.CallToolRequest
	AssertNil(t, ProgressTokenFromRequest(req))
	AssertNil(t, ProgressTokenFromRequest(&sdkmcp.CallToolRequest{}))
}

// moved AX-7 triplet TestProgress_ProgressTokenFromRequest_Ugly
func TestProgress_ProgressTokenFromRequest_Ugly(t *T) {
	req := &sdkmcp.CallToolRequest{Params: &sdkmcp.CallToolParamsRaw{}}
	req.Params.SetProgressToken(42)
	AssertEqual(t, 42, ProgressTokenFromRequest(req))
}

// moved AX-7 triplet TestProgress_SendProgressNotification_Good
func TestProgress_SendProgressNotification_Good(t *T) {
	req := &sdkmcp.CallToolRequest{Params: &sdkmcp.CallToolParamsRaw{}}
	err := SendProgressNotification(context.Background(), req, 1, 2, "ok")
	AssertNoError(t, err)
}

// moved AX-7 triplet TestProgress_SendProgressNotification_Bad
func TestProgress_SendProgressNotification_Bad(t *T) {
	err := SendProgressNotification(context.Background(), nil, 1, 2, "ok")
	AssertNoError(t, err)
	AssertNil(t, ProgressTokenFromRequest(nil))
}

// moved AX-7 triplet TestProgress_SendProgressNotification_Ugly
func TestProgress_SendProgressNotification_Ugly(t *T) {
	req := &sdkmcp.CallToolRequest{Params: &sdkmcp.CallToolParamsRaw{}}
	req.Params.SetProgressToken("tok")
	err := SendProgressNotification(context.Background(), req, -1, 0, "")
	AssertNoError(t, err)
}
