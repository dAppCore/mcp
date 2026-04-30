// SPDX-License-Identifier: EUPL-1.2

package brain

import (
	"context"
	"github.com/goccy/go-json"
	"testing"
	"time"

	"dappco.re/go/mcp/pkg/mcp/ide"
	"dappco.re/go/ws"
)

type recordingNotifier struct {
	channel string
	data    any
}

func (r *recordingNotifier) ChannelSend(_ context.Context, channel string, data any) {
	r.channel = channel
	r.data = data
}

// --- Nil bridge tests (headless mode) ---

func TestBrainRemember_Bad_NilBridge(t *testing.T) {
	sub := New(nil)
	_, _, err := sub.brainRemember(context.Background(), nil, RememberInput{
		Content: "test memory",
		Type:    "observation",
	})
	if err == nil {
		t.Error("expected error when bridge is nil")
	}
}

func TestBrainRecall_Bad_NilBridge(t *testing.T) {
	sub := New(nil)
	_, _, err := sub.brainRecall(context.Background(), nil, RecallInput{
		Query: "how does scoring work?",
	})
	if err == nil {
		t.Error("expected error when bridge is nil")
	}
}

func TestBrainForget_Bad_NilBridge(t *testing.T) {
	sub := New(nil)
	_, _, err := sub.brainForget(context.Background(), nil, ForgetInput{
		ID: "550e8400-e29b-41d4-a716-446655440000",
	})
	if err == nil {
		t.Error("expected error when bridge is nil")
	}
}

func TestBrainList_Bad_NilBridge(t *testing.T) {
	sub := New(nil)
	_, _, err := sub.brainList(context.Background(), nil, ListInput{
		Project: "eaas",
	})
	if err == nil {
		t.Error("expected error when bridge is nil")
	}
}

// --- Subsystem interface tests ---

func TestSubsystem_Good_Name(t *testing.T) {
	sub := New(nil)
	if sub.Name() != "brain" {
		t.Errorf("expected Name() = 'brain', got %q", sub.Name())
	}
}

func TestSubsystem_Good_ShutdownNoop(t *testing.T) {
	sub := New(nil)
	if err := sub.Shutdown(context.Background()); err != nil {
		t.Errorf("Shutdown failed: %v", err)
	}
}

func TestSubsystem_Good_BridgeRecallNotification(t *testing.T) {
	sub := New(nil)
	notifier := &recordingNotifier{}
	sub.notifier = notifier

	sub.handleBridgeMessage(ide.BridgeMessage{
		Type: "brain_recall",
		Data: map[string]any{
			"query":   "how does scoring work?",
			"org":     "core",
			"project": "eaas",
			"memories": []any{
				map[string]any{"id": "m1"},
				map[string]any{"id": "m2"},
			},
		},
	})

	if notifier.channel != "brain.recall.complete" {
		t.Fatalf("expected brain.recall.complete, got %q", notifier.channel)
	}

	payload, ok := notifier.data.(map[string]any)
	if !ok {
		t.Fatalf("expected payload map, got %T", notifier.data)
	}
	if payload["count"] != 2 {
		t.Fatalf("expected count 2, got %v", payload["count"])
	}
	if payload["query"] != "how does scoring work?" {
		t.Fatalf("expected query to be forwarded, got %v", payload["query"])
	}
	if payload["org"] != "core" {
		t.Fatalf("expected org to be forwarded, got %v", payload["org"])
	}
}

// --- Struct round-trip tests ---

func TestRememberInput_Good_RoundTrip(t *testing.T) {
	in := RememberInput{
		Content:    "LEM scoring was blind to negative emotions",
		Type:       "bug",
		Tags:       []string{"scoring", "lem"},
		Org:        "core",
		Project:    "eaas",
		Confidence: 0.95,
		Supersedes: "550e8400-e29b-41d4-a716-446655440000",
		ExpiresIn:  24,
	}
	data, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	var out RememberInput
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if out.Content != in.Content || out.Type != in.Type {
		t.Errorf("round-trip mismatch: content or type")
	}
	if len(out.Tags) != 2 || out.Tags[0] != "scoring" {
		t.Errorf("round-trip mismatch: tags")
	}
	if out.Org != "core" {
		t.Errorf("round-trip mismatch: org %q != core", out.Org)
	}
	if out.Confidence != 0.95 {
		t.Errorf("round-trip mismatch: confidence %f != 0.95", out.Confidence)
	}
}

func TestRememberOutput_Good_RoundTrip(t *testing.T) {
	in := RememberOutput{
		Success:   true,
		MemoryID:  "550e8400-e29b-41d4-a716-446655440000",
		Timestamp: time.Now().Truncate(time.Second),
	}
	data, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	var out RememberOutput
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if !out.Success || out.MemoryID != in.MemoryID {
		t.Errorf("round-trip mismatch: %+v != %+v", out, in)
	}
}

func TestRecallInput_Good_RoundTrip(t *testing.T) {
	in := RecallInput{
		Query: "how does verdict classification work?",
		TopK:  5,
		Filter: RecallFilter{
			Org:           "core",
			Project:       "eaas",
			MinConfidence: 0.5,
		},
	}
	data, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	var out RecallInput
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if out.Query != in.Query || out.TopK != 5 {
		t.Errorf("round-trip mismatch: query or topK")
	}
	if out.Filter.Org != "core" || out.Filter.Project != "eaas" || out.Filter.MinConfidence != 0.5 {
		t.Errorf("round-trip mismatch: filter")
	}
}

func TestMemory_Good_RoundTrip(t *testing.T) {
	in := Memory{
		ID:         "550e8400-e29b-41d4-a716-446655440000",
		AgentID:    "virgil",
		Type:       "decision",
		Content:    "Use Qdrant for vector search",
		Tags:       []string{"architecture", "openbrain"},
		Org:        "core",
		Project:    "php-agentic",
		Confidence: 0.9,
		CreatedAt:  "2026-03-03T12:00:00+00:00",
		UpdatedAt:  "2026-03-03T12:00:00+00:00",
	}
	data, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	var out Memory
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if out.ID != in.ID || out.AgentID != "virgil" || out.Type != "decision" || out.Org != "core" {
		t.Errorf("round-trip mismatch: %+v", out)
	}
}

func TestForgetInput_Good_RoundTrip(t *testing.T) {
	in := ForgetInput{
		ID:     "550e8400-e29b-41d4-a716-446655440000",
		Reason: "Superseded by new approach",
	}
	data, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	var out ForgetInput
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if out.ID != in.ID || out.Reason != in.Reason {
		t.Errorf("round-trip mismatch: %+v != %+v", out, in)
	}
}

func TestListInput_Good_RoundTrip(t *testing.T) {
	in := ListInput{
		Org:     "core",
		Project: "eaas",
		Type:    "decision",
		AgentID: "charon",
		Limit:   20,
	}
	data, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	var out ListInput
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if out.Org != "core" || out.Project != "eaas" || out.Type != "decision" || out.AgentID != "charon" || out.Limit != 20 {
		t.Errorf("round-trip mismatch: %+v", out)
	}
}

func TestListOutput_Good_RoundTrip(t *testing.T) {
	in := ListOutput{
		Success: true,
		Count:   2,
		Memories: []Memory{
			{ID: "id-1", AgentID: "virgil", Type: "decision", Content: "memory 1", Confidence: 0.9, CreatedAt: "2026-03-03T12:00:00+00:00", UpdatedAt: "2026-03-03T12:00:00+00:00"},
			{ID: "id-2", AgentID: "charon", Type: "bug", Content: "memory 2", Confidence: 0.8, CreatedAt: "2026-03-03T13:00:00+00:00", UpdatedAt: "2026-03-03T13:00:00+00:00"},
		},
	}
	data, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	var out ListOutput
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if !out.Success || out.Count != 2 || len(out.Memories) != 2 {
		t.Errorf("round-trip mismatch: %+v", out)
	}
}

// moved AX-7 triplet TestBrain_New_Good
func TestBrain_New_Good(t *T) {
	bridge := ide.NewBridge(ws.NewHub(), ide.DefaultConfig())
	sub := New(bridge)
	AssertEqual(t, bridge, sub.bridge)
	AssertEqual(t, "brain", sub.Name())
}

// moved AX-7 triplet TestBrain_New_Bad
func TestBrain_New_Bad(t *T) {
	sub := New(nil)
	AssertNil(t, sub.bridge)
	AssertEqual(t, "brain", sub.Name())
}

// moved AX-7 triplet TestBrain_New_Ugly
func TestBrain_New_Ugly(t *T) {
	sub := New(ide.NewBridge(nil, ide.DefaultConfig()))
	AssertNotNil(t, sub.bridge)
	AssertNoError(t, sub.Shutdown(context.Background()))
}

// moved AX-7 triplet TestBrain_Subsystem_Name_Good
func TestBrain_Subsystem_Name_Good(t *T) {
	sub := New(nil)
	AssertEqual(t, "brain", sub.Name())
	AssertNil(t, sub.bridge)
}

// moved AX-7 triplet TestBrain_Subsystem_Name_Bad
func TestBrain_Subsystem_Name_Bad(t *T) {
	var sub *Subsystem
	AssertEqual(t, "brain", sub.Name())
	AssertNil(t, sub)
}

// moved AX-7 triplet TestBrain_Subsystem_Name_Ugly
func TestBrain_Subsystem_Name_Ugly(t *T) {
	sub := &Subsystem{}
	AssertEqual(t, "brain", sub.Name())
	AssertNil(t, sub.notifier)
}

// moved AX-7 triplet TestBrain_Subsystem_RegisterTools_Good
func TestBrain_Subsystem_RegisterTools_Good(t *T) {
	svc := brainMCPServiceForTest(t)
	New(nil).RegisterTools(svc)
	AssertTrue(t, len(svc.Tools()) > 0)
}

// moved AX-7 triplet TestBrain_Subsystem_RegisterTools_Bad
func TestBrain_Subsystem_RegisterTools_Bad(t *T) {
	sub := New(nil)
	AssertPanics(t, func() { sub.RegisterTools(nil) })
	AssertEqual(t, "brain", sub.Name())
}

// moved AX-7 triplet TestBrain_Subsystem_RegisterTools_Ugly
func TestBrain_Subsystem_RegisterTools_Ugly(t *T) {
	svc := brainMCPServiceForTest(t)
	(&Subsystem{}).RegisterTools(svc)
	AssertTrue(t, len(svc.Tools()) > 0)
}

// moved AX-7 triplet TestBrain_Subsystem_SetNotifier_Good
func TestBrain_Subsystem_SetNotifier_Good(t *T) {
	sub := New(nil)
	notifier := &recordingNotifier{}
	sub.SetNotifier(notifier)
	AssertEqual(t, notifier, sub.notifier)
}

// moved AX-7 triplet TestBrain_Subsystem_SetNotifier_Bad
func TestBrain_Subsystem_SetNotifier_Bad(t *T) {
	sub := New(nil)
	sub.SetNotifier(nil)
	AssertNil(t, sub.notifier)
}

// moved AX-7 triplet TestBrain_Subsystem_SetNotifier_Ugly
func TestBrain_Subsystem_SetNotifier_Ugly(t *T) {
	sub := &Subsystem{}
	sub.SetNotifier(&recordingNotifier{})
	AssertNotNil(t, sub.notifier)
}

// moved AX-7 triplet TestBrain_Subsystem_Shutdown_Good
func TestBrain_Subsystem_Shutdown_Good(t *T) {
	sub := New(nil)
	err := sub.Shutdown(context.Background())
	AssertNoError(t, err)
}

// moved AX-7 triplet TestBrain_Subsystem_Shutdown_Bad
func TestBrain_Subsystem_Shutdown_Bad(t *T) {
	sub := New(nil)
	err := sub.Shutdown(nil)
	AssertNoError(t, err)
}

// moved AX-7 triplet TestBrain_Subsystem_Shutdown_Ugly
func TestBrain_Subsystem_Shutdown_Ugly(t *T) {
	var sub *Subsystem
	err := sub.Shutdown(context.Background())
	AssertNoError(t, err)
}
