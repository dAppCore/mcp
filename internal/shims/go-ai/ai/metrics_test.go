package ai

import (
	"time"

	. "dappco.re/go"
)

// moved AX-7 triplet TestMetrics_Record_Good
func TestMetrics_Record_Good(t *T) {
	resetAX7Events()
	err := Record(Event{Type: "build", AgentID: "codex", Repo: "mcp"})
	AssertNoError(t, err)
	got, err := ReadEvents(time.Time{})
	AssertNoError(t, err)
	AssertLen(t, got, 1)
	AssertEqual(t, "build", got[0].Type)
}

// moved AX-7 triplet TestMetrics_Record_Bad
func TestMetrics_Record_Bad(t *T) {
	resetAX7Events()
	err := Record(Event{})
	AssertNoError(t, err)
	got, err := ReadEvents(time.Time{})
	AssertNoError(t, err)
	AssertLen(t, got, 1)
}

// moved AX-7 triplet TestMetrics_Record_Ugly
func TestMetrics_Record_Ugly(t *T) {
	resetAX7Events()
	err := Record(Event{Type: "zero-time"})
	AssertNoError(t, err)
	got, err := ReadEvents(time.Time{})
	AssertNoError(t, err)
	AssertLen(t, got, 1)
	AssertFalse(t, got[0].Timestamp.IsZero())
}

// moved AX-7 triplet TestMetrics_ReadEvents_Good
func TestMetrics_ReadEvents_Good(t *T) {
	resetAX7Events()
	now := time.Now()
	events.Lock()
	events.items = []Event{
		{Type: "old", Timestamp: now.Add(-time.Hour)},
		{Type: "new", Timestamp: now.Add(time.Hour)},
	}
	events.Unlock()
	got, err := ReadEvents(now)
	AssertNoError(t, err)
	AssertLen(t, got, 1)
	AssertEqual(t, "new", got[0].Type)
}

// moved AX-7 triplet TestMetrics_ReadEvents_Bad
func TestMetrics_ReadEvents_Bad(t *T) {
	resetAX7Events()
	err := Record(Event{Type: "build"})
	AssertNoError(t, err)
	got, err := ReadEvents(time.Now().Add(time.Hour))
	AssertNoError(t, err)
	AssertLen(t, got, 0)
}

// moved AX-7 triplet TestMetrics_ReadEvents_Ugly
func TestMetrics_ReadEvents_Ugly(t *T) {
	resetAX7Events()
	events.Lock()
	events.items = []Event{{Type: "zero"}}
	events.Unlock()
	got, err := ReadEvents(time.Time{})
	AssertNoError(t, err)
	AssertLen(t, got, 0)
}

// moved AX-7 triplet TestMetrics_Summary_Good
func TestMetrics_Summary_Good(t *T) {
	got := Summary([]Event{
		{Type: "build", Repo: "mcp", AgentID: "codex"},
		{Type: "build", Repo: "mcp", AgentID: "claude"},
	})
	AssertEqual(t, 2, got["total"])
	AssertLen(t, got["by_type"], 1)
}

// moved AX-7 triplet TestMetrics_Summary_Bad
func TestMetrics_Summary_Bad(t *T) {
	got := Summary(nil)
	AssertEqual(t, 0, got["total"])
	AssertLen(t, got["by_type"], 0)
}

// moved AX-7 triplet TestMetrics_Summary_Ugly
func TestMetrics_Summary_Ugly(t *T) {
	got := Summary([]Event{{Type: "", Repo: "", AgentID: ""}})
	AssertEqual(t, 1, got["total"])
	AssertLen(t, got["by_repo"], 0)
}
