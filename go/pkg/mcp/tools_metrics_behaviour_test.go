// SPDX-License-Identifier: EUPL-1.2

package mcp

import (
	"context"
	"testing"
	"time"
)

func TestMetrics_recordAndRead_Good(t *testing.T) {
	// Capture a cut-off just before we record so the package-global event slice
	// from other tests does not bleed into this query window.
	since := time.Now()
	time.Sleep(time.Millisecond)

	for _, ev := range []metricEvent{
		{Type: "dispatch.complete", AgentID: "cladius", Repo: "core/mcp"},
		{Type: "dispatch.complete", AgentID: "hephaestus", Repo: "core/mcp"},
		{Type: "dispatch.fail", AgentID: "cladius", Repo: "core/api"},
	} {
		if err := recordMetricEvent(ev); err != nil {
			t.Fatalf("recordMetricEvent: %v", err)
		}
	}

	got, err := readMetricEvents(since)
	if err != nil {
		t.Fatalf("readMetricEvents: %v", err)
	}
	if len(got) < 3 {
		t.Fatalf("expected at least 3 events since cut-off, got %d", len(got))
	}
	// recordMetricEvent stamps a timestamp when zero.
	for _, e := range got {
		if e.Timestamp.IsZero() {
			t.Fatal("expected recorded event to have a non-zero timestamp")
		}
	}
}

func TestMetrics_readMetricEvents_Ugly_FutureSinceEmpty(t *testing.T) {
	if err := recordMetricEvent(metricEvent{Type: "x", Timestamp: time.Now()}); err != nil {
		t.Fatalf("recordMetricEvent: %v", err)
	}
	// A cut-off in the future excludes everything.
	got, err := readMetricEvents(time.Now().Add(time.Hour))
	if err != nil {
		t.Fatalf("readMetricEvents: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected 0 events for future cut-off, got %d", len(got))
	}
}

func TestMetrics_summarizeMetricEvents_Good(t *testing.T) {
	events := []metricEvent{
		{Type: "a", AgentID: "x", Repo: "r1"},
		{Type: "a", AgentID: "y", Repo: "r1"},
		{Type: "b", AgentID: "x", Repo: "r2"},
		{Type: "", AgentID: "", Repo: ""}, // empty fields are ignored
	}
	summary := summarizeMetricEvents(events)

	if total, _ := summary["total"].(int); total != 4 {
		t.Fatalf("expected total 4, got %v", summary["total"])
	}

	byType := convertMetricCounts(summary["by_type"])
	counts := map[string]int{}
	for _, mc := range byType {
		counts[mc.Key] = mc.Count
	}
	if counts["a"] != 2 || counts["b"] != 1 {
		t.Fatalf("unexpected by_type counts: %+v", counts)
	}
	if _, ok := counts[""]; ok {
		t.Fatal("empty type should not be counted")
	}

	byRepo := convertMetricCounts(summary["by_repo"])
	if len(byRepo) != 2 {
		t.Fatalf("expected 2 repos, got %d", len(byRepo))
	}
	byAgent := convertMetricCounts(summary["by_agent"])
	if len(byAgent) != 2 {
		t.Fatalf("expected 2 agents, got %d", len(byAgent))
	}
}

func TestMetrics_convertMetricCounts_Ugly_NilAndWrongType(t *testing.T) {
	if got := convertMetricCounts(nil); len(got) != 0 {
		t.Fatalf("expected empty slice for nil, got %d", len(got))
	}
	if got := convertMetricCounts("not a slice"); len(got) != 0 {
		t.Fatalf("expected empty slice for wrong type, got %d", len(got))
	}
}

func TestMetrics_metricsRecord_Good(t *testing.T) {
	s, err := New(Options{})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	_, out, err := s.metricsRecord(context.Background(), nil, MetricsRecordInput{
		Type:    "tool.call",
		AgentID: "cladius",
		Repo:    "core/mcp",
		Data:    map[string]any{"tool": "file_read"},
	})
	if err != nil {
		t.Fatalf("metricsRecord: %v", err)
	}
	if !out.Success {
		t.Fatal("expected success")
	}
	if out.Timestamp.IsZero() {
		t.Fatal("expected a server-assigned timestamp")
	}
}

func TestMetrics_metricsRecord_Bad_EmptyType(t *testing.T) {
	s, err := New(Options{})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if _, _, err := s.metricsRecord(context.Background(), nil, MetricsRecordInput{
		AgentID: "cladius",
	}); err == nil {
		t.Fatal("expected error for empty type")
	}
}

func TestMetrics_metricsQuery_Good(t *testing.T) {
	s, err := New(Options{})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	// Seed a couple of events through the public record handler.
	for _, typ := range []string{"q.one", "q.one", "q.two"} {
		if _, _, err := s.metricsRecord(context.Background(), nil, MetricsRecordInput{
			Type:    typ,
			AgentID: "cladius",
			Repo:    "core/mcp",
		}); err != nil {
			t.Fatalf("seed metricsRecord: %v", err)
		}
	}

	_, out, err := s.metricsQuery(context.Background(), nil, MetricsQueryInput{Since: "1h"})
	if err != nil {
		t.Fatalf("metricsQuery: %v", err)
	}
	if out.Total < 3 {
		t.Fatalf("expected at least 3 events in the last hour, got %d", out.Total)
	}
	// Events list is capped at DefaultMetricsLimit and ordered most-recent-first.
	if len(out.Events) > DefaultMetricsLimit {
		t.Fatalf("expected at most %d events, got %d", DefaultMetricsLimit, len(out.Events))
	}
}

func TestMetrics_metricsQuery_Bad_InvalidSince(t *testing.T) {
	s, err := New(Options{})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if _, _, err := s.metricsQuery(context.Background(), nil, MetricsQueryInput{Since: "not-a-duration"}); err == nil {
		t.Fatal("expected error for invalid since value")
	}
}

func TestMetrics_metricsQuery_Good_DefaultSince(t *testing.T) {
	s, err := New(Options{})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	// Empty Since must fall back to DefaultMetricsSince without error.
	if _, _, err := s.metricsQuery(context.Background(), nil, MetricsQueryInput{}); err != nil {
		t.Fatalf("metricsQuery with default since: %v", err)
	}
}
