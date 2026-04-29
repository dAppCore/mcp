package ai

import (
	"sync"
	"time"
)

type Event struct {
	Type      string         `json:"type"`
	Timestamp time.Time      `json:"timestamp"`
	AgentID   string         `json:"agent_id,omitempty"`
	Repo      string         `json:"repo,omitempty"`
	Data      map[string]any `json:"data,omitempty"`
}

var events struct {
	sync.Mutex
	items []Event
}

func Record(
	event Event,
) (
	_ error, // result
) {
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}
	events.Lock()
	defer events.Unlock()
	events.items = append(events.items, event)
	return nil
}

func ReadEvents(since time.Time) (
	[]Event,
	error,
) {
	events.Lock()
	defer events.Unlock()
	out := make([]Event, 0, len(events.items))
	for _, event := range events.items {
		if event.Timestamp.IsZero() || event.Timestamp.Before(since) {
			continue
		}
		out = append(out, event)
	}
	return out, nil
}

func Summary(input []Event) map[string]any {
	byType := map[string]int{}
	byRepo := map[string]int{}
	byAgent := map[string]int{}
	for _, event := range input {
		if event.Type != "" {
			byType[event.Type]++
		}
		if event.Repo != "" {
			byRepo[event.Repo]++
		}
		if event.AgentID != "" {
			byAgent[event.AgentID]++
		}
	}
	return map[string]any{
		"total":    len(input),
		"by_type":  metricCounts(byType),
		"by_repo":  metricCounts(byRepo),
		"by_agent": metricCounts(byAgent),
	}
}

func metricCounts(counts map[string]int) []map[string]any {
	out := make([]map[string]any, 0, len(counts))
	for key, count := range counts {
		out = append(out, map[string]any{"key": key, "count": count})
	}
	return out
}
