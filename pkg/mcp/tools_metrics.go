package mcp

import (
	"context"
	"strconv"
	"time"

	core "dappco.re/go/core"
	"forge.lthn.ai/core/go-ai/ai"
	"forge.lthn.ai/core/go-log"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Default values for metrics operations.
const (
	DefaultMetricsSince = "7d"
	DefaultMetricsLimit = 10
)

// MetricsRecordInput contains parameters for recording a metrics event.
//
//	input := MetricsRecordInput{
//	    Type:    "dispatch.complete",
//	    AgentID: "cladius",
//	    Repo:    "core-php",
//	    Data:    map[string]any{"duration": "4m32s"},
//	}
type MetricsRecordInput struct {
	Type    string         `json:"type"`               // e.g. "dispatch.complete"
	AgentID string         `json:"agent_id,omitempty"` // e.g. "cladius"
	Repo    string         `json:"repo,omitempty"`     // e.g. "core-php"
	Data    map[string]any `json:"data,omitempty"`     // arbitrary key-value data
}

// MetricsRecordOutput contains the result of recording a metrics event.
//
//	// out.Success == true, out.Timestamp == 2026-03-21T14:30:00Z
type MetricsRecordOutput struct {
	Success   bool      `json:"success"`   // true when the event was recorded
	Timestamp time.Time `json:"timestamp"` // server-assigned timestamp
}

// MetricsQueryInput contains parameters for querying metrics.
//
//	input := MetricsQueryInput{Since: "24h"}
type MetricsQueryInput struct {
	Since string `json:"since,omitempty"` // e.g. "7d", "24h", "30m" (default: "7d")
}

// MetricsQueryOutput contains the results of a metrics query.
//
//	// out.Total == 42, len(out.Events) <= 10
type MetricsQueryOutput struct {
	Total   int                `json:"total"`    // total events in range
	ByType  []MetricCount      `json:"by_type"`  // counts grouped by event type
	ByRepo  []MetricCount      `json:"by_repo"`  // counts grouped by repository
	ByAgent []MetricCount      `json:"by_agent"` // counts grouped by agent ID
	Events  []MetricEventBrief `json:"events"`   // most recent 10 events
}

// MetricCount represents a count for a specific key.
//
//	// mc.Key == "dispatch.complete", mc.Count == 15
type MetricCount struct {
	Key   string `json:"key"`   // e.g. "dispatch.complete" or "core-php"
	Count int    `json:"count"` // number of events matching this key
}

// MetricEventBrief represents a brief summary of an event.
//
//	// ev.Type == "dispatch.complete", ev.AgentID == "cladius", ev.Repo == "core-php"
type MetricEventBrief struct {
	Type      string    `json:"type"`               // e.g. "dispatch.complete"
	Timestamp time.Time `json:"timestamp"`          // when the event occurred
	AgentID   string    `json:"agent_id,omitempty"` // e.g. "cladius"
	Repo      string    `json:"repo,omitempty"`     // e.g. "core-php"
}

// registerMetricsTools adds metrics tools to the MCP server.
func (s *Service) registerMetricsTools(server *mcp.Server) {
	addToolRecorded(s, server, "metrics", &mcp.Tool{
		Name:        "metrics_record",
		Description: "Record a metrics event for AI/security tracking. Events are stored in daily JSONL files.",
	}, s.metricsRecord)

	addToolRecorded(s, server, "metrics", &mcp.Tool{
		Name:        "metrics_query",
		Description: "Query metrics events and get aggregated statistics by type, repo, and agent.",
	}, s.metricsQuery)
}

// metricsRecord handles the metrics_record tool call.
func (s *Service) metricsRecord(ctx context.Context, req *mcp.CallToolRequest, input MetricsRecordInput) (*mcp.CallToolResult, MetricsRecordOutput, error) {
	s.logger.Info("MCP tool execution", "tool", "metrics_record", "type", input.Type, "agent_id", input.AgentID, "repo", input.Repo, "user", log.Username())

	// Validate input
	if input.Type == "" {
		return nil, MetricsRecordOutput{}, log.E("metricsRecord", "type cannot be empty", nil)
	}

	// Create the event
	event := ai.Event{
		Type:      input.Type,
		Timestamp: time.Now(),
		AgentID:   input.AgentID,
		Repo:      input.Repo,
		Data:      input.Data,
	}

	// Record the event
	if err := ai.Record(event); err != nil {
		log.Error("mcp: metrics record failed", "type", input.Type, "err", err)
		return nil, MetricsRecordOutput{}, log.E("metricsRecord", "failed to record metrics", err)
	}

	return nil, MetricsRecordOutput{
		Success:   true,
		Timestamp: event.Timestamp,
	}, nil
}

// metricsQuery handles the metrics_query tool call.
func (s *Service) metricsQuery(ctx context.Context, req *mcp.CallToolRequest, input MetricsQueryInput) (*mcp.CallToolResult, MetricsQueryOutput, error) {
	// Apply defaults
	since := input.Since
	if since == "" {
		since = DefaultMetricsSince
	}

	s.logger.Info("MCP tool execution", "tool", "metrics_query", "since", since, "user", log.Username())

	// Parse the duration
	duration, err := parseDuration(since)
	if err != nil {
		return nil, MetricsQueryOutput{}, log.E("metricsQuery", "invalid since value", err)
	}

	sinceTime := time.Now().Add(-duration)

	// Read events
	events, err := ai.ReadEvents(sinceTime)
	if err != nil {
		log.Error("mcp: metrics query failed", "since", since, "err", err)
		return nil, MetricsQueryOutput{}, log.E("metricsQuery", "failed to read metrics", err)
	}

	// Get summary
	summary := ai.Summary(events)

	// Build output
	total, _ := summary["total"].(int)
	output := MetricsQueryOutput{
		Total:   total,
		ByType:  convertMetricCounts(summary["by_type"]),
		ByRepo:  convertMetricCounts(summary["by_repo"]),
		ByAgent: convertMetricCounts(summary["by_agent"]),
		Events:  make([]MetricEventBrief, 0, DefaultMetricsLimit),
	}

	// Get recent events (last 10, most recent first)
	startIdx := max(len(events)-DefaultMetricsLimit, 0)
	for i := len(events) - 1; i >= startIdx; i-- {
		ev := events[i]
		output.Events = append(output.Events, MetricEventBrief{
			Type:      ev.Type,
			Timestamp: ev.Timestamp,
			AgentID:   ev.AgentID,
			Repo:      ev.Repo,
		})
	}

	return nil, output, nil
}

// convertMetricCounts converts the summary map format to MetricCount slice.
func convertMetricCounts(data any) []MetricCount {
	if data == nil {
		return []MetricCount{}
	}

	items, ok := data.([]map[string]any)
	if !ok {
		return []MetricCount{}
	}

	result := make([]MetricCount, len(items))
	for i, item := range items {
		key, _ := item["key"].(string)
		count, _ := item["count"].(int)
		result[i] = MetricCount{Key: key, Count: count}
	}
	return result
}

// parseDuration parses a duration string like "7d", "24h", "30m".
func parseDuration(s string) (time.Duration, error) {
	if s == "" {
		return 0, log.E("parseDuration", "duration cannot be empty", nil)
	}

	s = core.Trim(s)
	if len(s) < 2 {
		return 0, log.E("parseDuration", "invalid duration format: "+s, nil)
	}

	// Get the numeric part and unit
	unit := s[len(s)-1]
	numStr := s[:len(s)-1]

	num, err := strconv.Atoi(numStr)
	if err != nil {
		return 0, log.E("parseDuration", "invalid duration number: "+numStr, err)
	}

	if num <= 0 {
		return 0, log.E("parseDuration", core.Sprintf("duration must be positive: %d", num), nil)
	}

	switch unit {
	case 'd':
		return time.Duration(num) * 24 * time.Hour, nil
	case 'h':
		return time.Duration(num) * time.Hour, nil
	case 'm':
		return time.Duration(num) * time.Minute, nil
	default:
		return 0, log.E("parseDuration", "invalid duration unit: "+string(unit)+" (expected d, h, or m)", nil)
	}
}
