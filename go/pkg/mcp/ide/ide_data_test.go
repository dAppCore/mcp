// SPDX-License-Identifier: EUPL-1.2

package ide

import (
	"testing"
)

func TestIdeData_buildInfosFromData_Good_Bad(t *testing.T) {
	data := map[string]any{
		"builds": []any{
			map[string]any{"id": "b1", "repo": "core/mcp", "status": "running"},
			map[string]any{"id": "b2", "repo": "core/api", "status": "passed"},
			map[string]any{"no_id": true}, // skipped: no id
		},
	}
	got := buildInfosFromData(data)
	if len(got) != 2 {
		t.Fatalf("expected 2 builds, got %d", len(got))
	}
	if got[0].ID != "b1" || got[0].Repo != "core/mcp" {
		t.Fatalf("unexpected first build: %+v", got[0])
	}

	// Wrong shape → empty slice (non-nil).
	if out := buildInfosFromData("not a map"); len(out) != 0 {
		t.Fatalf("expected empty for non-map, got %d", len(out))
	}
	if out := buildInfosFromData(map[string]any{"builds": "wrong"}); len(out) != 0 {
		t.Fatalf("expected empty for non-slice builds, got %d", len(out))
	}
}

func TestIdeData_buildLogsFromData_Good_Variants(t *testing.T) {
	// []any lines.
	id, lines := buildLogsFromData(map[string]any{
		"buildId": "b1",
		"lines":   []any{"line one", "line two"},
	})
	if id != "b1" || len(lines) != 2 {
		t.Fatalf("[]any lines: id=%q lines=%v", id, lines)
	}

	// []string lines + "id" fallback key.
	id, lines = buildLogsFromData(map[string]any{
		"id":    "b2",
		"lines": []string{"a", "b", "c"},
	})
	if id != "b2" || len(lines) != 3 {
		t.Fatalf("[]string lines: id=%q lines=%v", id, lines)
	}

	// "output" fallback.
	id, lines = buildLogsFromData(map[string]any{
		"buildId": "b3",
		"output":  "single blob",
	})
	if id != "b3" || len(lines) != 1 || lines[0] != "single blob" {
		t.Fatalf("output fallback: id=%q lines=%v", id, lines)
	}

	// buildLinesFromData is a thin wrapper.
	if l := buildLinesFromData(map[string]any{"lines": []any{"x"}}); len(l) != 1 {
		t.Fatalf("buildLinesFromData = %v", l)
	}
}

func TestIdeData_buildLogsFromData_Ugly_WrongShape(t *testing.T) {
	id, lines := buildLogsFromData(42)
	if id != "" || len(lines) != 0 {
		t.Fatalf("expected empty for non-map, got id=%q lines=%v", id, lines)
	}
}

func TestIdeData_sessionsFromData_Good_Bad(t *testing.T) {
	got := sessionsFromData(map[string]any{
		"sessions": []any{
			map[string]any{"id": "s1", "name": "alpha", "status": "active"},
			map[string]any{"id": "s2"},   // status defaults to "unknown"
			map[string]any{"name": "no"}, // skipped: no id
		},
	})
	if len(got) != 2 {
		t.Fatalf("expected 2 sessions, got %d", len(got))
	}
	if got[1].Status != "unknown" {
		t.Fatalf("expected default status unknown, got %q", got[1].Status)
	}

	if out := sessionsFromData("nope"); len(out) != 0 {
		t.Fatalf("expected empty for non-map, got %d", len(out))
	}
}

func TestIdeData_sessionFromData_Bad_NoID(t *testing.T) {
	if _, ok := sessionFromData(map[string]any{"name": "x"}); ok {
		t.Fatal("expected ok=false when id missing")
	}
	if _, ok := sessionFromData("not a map"); ok {
		t.Fatal("expected ok=false for non-map")
	}
}

func TestIdeData_chatHistoryFromData_Good_Bad(t *testing.T) {
	sessionID, msgs := chatHistoryFromData(map[string]any{
		"sessionId": "s1",
		"messages": []any{
			map[string]any{"role": "user", "content": "hi"},
			map[string]any{"role": "assistant", "content": "hello"},
			map[string]any{}, // skipped: empty role+content
		},
	})
	if sessionID != "s1" || len(msgs) != 2 {
		t.Fatalf("history: sessionID=%q msgs=%d", sessionID, len(msgs))
	}

	// session_id fallback key + no messages.
	sessionID, msgs = chatHistoryFromData(map[string]any{"session_id": "s9"})
	if sessionID != "s9" || len(msgs) != 0 {
		t.Fatalf("fallback: sessionID=%q msgs=%d", sessionID, len(msgs))
	}

	// Wrong shape.
	if sid, m := chatHistoryFromData(nil); sid != "" || len(m) != 0 {
		t.Fatalf("expected empty for nil, got sid=%q m=%d", sid, len(m))
	}
}

func TestIdeData_chatMessageFromData_Bad_Empty(t *testing.T) {
	if _, ok := chatMessageFromData(map[string]any{}); ok {
		t.Fatal("expected ok=false for empty role+content")
	}
	if _, ok := chatMessageFromData("nope"); ok {
		t.Fatal("expected ok=false for non-map")
	}
	// content-only is valid.
	if msg, ok := chatMessageFromData(map[string]any{"content": "just content"}); !ok || msg.Content != "just content" {
		t.Fatalf("expected content-only message, got %+v ok=%v", msg, ok)
	}
}

func TestIdeData_stringFromAny_Good(t *testing.T) {
	if stringFromAny("hi") != "hi" {
		t.Fatal("string passthrough failed")
	}
	if stringFromAny(123) != "" {
		t.Fatal("non-string should yield empty")
	}
	if stringFromAny(nil) != "" {
		t.Fatal("nil should yield empty")
	}
}

func TestIdeState_listBuilds_Good_FilterAndLimit(t *testing.T) {
	s := &Subsystem{}
	s.addBuild(BuildInfo{ID: "b1", Repo: "core/mcp", Status: "passed"})
	s.addBuild(BuildInfo{ID: "b2", Repo: "core/api", Status: "running"})
	s.addBuild(BuildInfo{ID: "b3", Repo: "core/mcp", Status: "failed"})

	// No filter: all three, most-recent-first.
	all := s.listBuilds("", 0)
	if len(all) != 3 {
		t.Fatalf("expected 3 builds, got %d", len(all))
	}
	if all[0].ID != "b3" {
		t.Fatalf("expected most-recent first (b3), got %q", all[0].ID)
	}

	// Repo filter.
	mcp := s.listBuilds("core/mcp", 0)
	if len(mcp) != 2 {
		t.Fatalf("expected 2 core/mcp builds, got %d", len(mcp))
	}

	// Limit.
	one := s.listBuilds("", 1)
	if len(one) != 1 {
		t.Fatalf("expected 1 build under limit, got %d", len(one))
	}
}

func TestIdeState_listBuilds_Ugly_Empty(t *testing.T) {
	if out := (&Subsystem{}).listBuilds("", 0); len(out) != 0 {
		t.Fatalf("expected empty for no builds, got %d", len(out))
	}
}

func TestIdeState_appendBuildLogAndTail_Good(t *testing.T) {
	s := &Subsystem{}
	for _, line := range []string{"l1", "l2", "l3", "l4"} {
		s.appendBuildLog("b1", line)
	}

	// Full tail.
	full := s.buildLogTail("b1", 0)
	if len(full) != 4 {
		t.Fatalf("expected 4 lines, got %d", len(full))
	}

	// Bounded tail.
	tail := s.buildLogTail("b1", 2)
	if len(tail) != 2 || tail[0] != "l3" || tail[1] != "l4" {
		t.Fatalf("unexpected tail: %v", tail)
	}

	// Tail larger than available collapses to all.
	big := s.buildLogTail("b1", 100)
	if len(big) != 4 {
		t.Fatalf("expected 4 lines for oversize tail, got %d", len(big))
	}
}

func TestIdeState_buildLogTail_Ugly_Missing(t *testing.T) {
	if out := (&Subsystem{}).buildLogTail("absent", 5); len(out) != 0 {
		t.Fatalf("expected empty for missing build, got %d", len(out))
	}
}
