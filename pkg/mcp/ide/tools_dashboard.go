// SPDX-License-Identifier: EUPL-1.2

package ide

import (
	"context"
	"sync"
	"time"

	coremcp "dappco.re/go/mcp/pkg/mcp"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Dashboard tool input/output types.

// DashboardOverviewInput is the input for ide_dashboard_overview.
//
//	input := DashboardOverviewInput{}
type DashboardOverviewInput struct{}

// DashboardOverview contains high-level platform stats.
//
//	overview := DashboardOverview{Repos: 12, ActiveSessions: 3}
type DashboardOverview struct {
	Repos          int  `json:"repos"`
	Services       int  `json:"services"`
	ActiveSessions int  `json:"activeSessions"`
	RecentBuilds   int  `json:"recentBuilds"`
	BridgeOnline   bool `json:"bridgeOnline"`
}

// DashboardOverviewOutput is the output for ide_dashboard_overview.
//
//	// out.Overview.BridgeOnline reports bridge connectivity
type DashboardOverviewOutput struct {
	Overview DashboardOverview `json:"overview"`
}

// DashboardActivityInput is the input for ide_dashboard_activity.
//
//	input := DashboardActivityInput{Limit: 25}
type DashboardActivityInput struct {
	Limit int `json:"limit,omitempty"`
}

// ActivityEvent represents a single activity feed item.
//
//	event := ActivityEvent{Type: "build", Message: "build finished"}
type ActivityEvent struct {
	Type      string    `json:"type"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
}

// DashboardActivityOutput is the output for ide_dashboard_activity.
//
//	// out.Events contains the recent activity feed
type DashboardActivityOutput struct {
	Events []ActivityEvent `json:"events"`
}

// DashboardMetricsInput is the input for ide_dashboard_metrics.
//
//	input := DashboardMetricsInput{Period: "24h"}
type DashboardMetricsInput struct {
	Period string `json:"period,omitempty"` // "1h", "24h", "7d"
}

// DashboardMetrics contains aggregate metrics.
//
//	metrics := DashboardMetrics{BuildsTotal: 42, SuccessRate: 0.95}
type DashboardMetrics struct {
	BuildsTotal   int     `json:"buildsTotal"`
	BuildsSuccess int     `json:"buildsSuccess"`
	BuildsFailed  int     `json:"buildsFailed"`
	AvgBuildTime  string  `json:"avgBuildTime"`
	AgentSessions int     `json:"agentSessions"`
	MessagesTotal int     `json:"messagesTotal"`
	SuccessRate   float64 `json:"successRate"`
}

// DashboardMetricsOutput is the output for ide_dashboard_metrics.
//
//	// out.Metrics summarises the selected time window
type DashboardMetricsOutput struct {
	Period  string           `json:"period"`
	Metrics DashboardMetrics `json:"metrics"`
}

// DashboardStateInput is the input for ide_dashboard_state.
//
//	input := DashboardStateInput{}
type DashboardStateInput struct{}

// DashboardStateOutput is the output for ide_dashboard_state.
//
//	// out.State["theme"] == "dark"
type DashboardStateOutput struct {
	State     map[string]any `json:"state"`     // arbitrary key/value map
	UpdatedAt time.Time      `json:"updatedAt"` // when the state last changed
}

// DashboardUpdateInput is the input for ide_dashboard_update.
//
//	input := DashboardUpdateInput{
//	    State:   map[string]any{"theme": "light", "sidebar": true},
//	    Replace: false,
//	}
type DashboardUpdateInput struct {
	State   map[string]any `json:"state"`             // partial or full state
	Replace bool           `json:"replace,omitempty"` // true to overwrite, false to merge (default)
}

// DashboardUpdateOutput is the output for ide_dashboard_update.
//
//	// out.State reflects the merged/replaced state
type DashboardUpdateOutput struct {
	State     map[string]any `json:"state"`     // merged state after the update
	UpdatedAt time.Time      `json:"updatedAt"` // when the state was applied
}

// dashboardStateStore holds the mutable dashboard UI state shared between the
// IDE frontend and MCP callers. Access is guarded by dashboardStateMu.
var (
	dashboardStateMu      sync.RWMutex
	dashboardStateStore   = map[string]any{}
	dashboardStateUpdated time.Time
)

func (s *Subsystem) registerDashboardTools(svc *coremcp.Service) {
	server := svc.Server()
	coremcp.AddToolRecorded(svc, server, "ide", &mcp.Tool{
		Name:        "ide_dashboard_overview",
		Description: "Get a high-level overview of the platform (repos, services, sessions, builds)",
	}, s.dashboardOverview)

	coremcp.AddToolRecorded(svc, server, "ide", &mcp.Tool{
		Name:        "ide_dashboard_activity",
		Description: "Get the recent activity feed",
	}, s.dashboardActivity)

	coremcp.AddToolRecorded(svc, server, "ide", &mcp.Tool{
		Name:        "ide_dashboard_metrics",
		Description: "Get aggregate build and agent metrics for a time period",
	}, s.dashboardMetrics)

	coremcp.AddToolRecorded(svc, server, "ide", &mcp.Tool{
		Name:        "ide_dashboard_state",
		Description: "Get the current dashboard UI state (arbitrary key/value map shared with the IDE).",
	}, s.dashboardState)

	coremcp.AddToolRecorded(svc, server, "ide", &mcp.Tool{
		Name:        "ide_dashboard_update",
		Description: "Update the dashboard UI state. Merges into existing state by default; set replace=true to overwrite.",
	}, s.dashboardUpdate)
}

// dashboardOverview returns a platform overview with bridge status and
// locally tracked session counts.
func (s *Subsystem) dashboardOverview(_ context.Context, _ *mcp.CallToolRequest, _ DashboardOverviewInput) (*mcp.CallToolResult, DashboardOverviewOutput, error) {
	connected := s.bridge != nil && s.bridge.Connected()
	activeSessions := len(s.listSessions())
	builds := s.listBuilds("", 0)
	repos := s.buildRepoCount()

	if s.bridge != nil {
		_ = s.bridge.Send(BridgeMessage{Type: "dashboard_overview"})
	}

	return nil, DashboardOverviewOutput{
		Overview: DashboardOverview{
			Repos:          repos,
			Services:       len(builds),
			ActiveSessions: activeSessions,
			RecentBuilds:   len(builds),
			BridgeOnline:   connected,
		},
	}, nil
}

// dashboardActivity returns the local activity feed and refreshes the Laravel
// backend when the bridge is available.
func (s *Subsystem) dashboardActivity(_ context.Context, _ *mcp.CallToolRequest, input DashboardActivityInput) (*mcp.CallToolResult, DashboardActivityOutput, error) {
	if s.bridge != nil {
		_ = s.bridge.Send(BridgeMessage{
			Type: "dashboard_activity",
			Data: map[string]any{"limit": input.Limit},
		})
	}
	return nil, DashboardActivityOutput{Events: s.activityFeed(input.Limit)}, nil
}

// dashboardMetrics returns local session and message counts and refreshes the
// Laravel backend when the bridge is available.
func (s *Subsystem) dashboardMetrics(_ context.Context, _ *mcp.CallToolRequest, input DashboardMetricsInput) (*mcp.CallToolResult, DashboardMetricsOutput, error) {
	period := input.Period
	if period == "" {
		period = "24h"
	}
	if s.bridge != nil {
		_ = s.bridge.Send(BridgeMessage{
			Type: "dashboard_metrics",
			Data: map[string]any{"period": period},
		})
	}

	s.stateMu.Lock()
	sessions := len(s.sessions)
	messages := 0
	builds := make([]BuildInfo, 0, len(s.buildOrder))
	for _, id := range s.buildOrder {
		if build, ok := s.builds[id]; ok {
			builds = append(builds, build)
		}
	}
	for _, history := range s.chats {
		messages += len(history)
	}
	s.stateMu.Unlock()

	total := len(builds)
	success := 0
	failed := 0
	var durationTotal time.Duration
	var durationCount int
	for _, build := range builds {
		switch build.Status {
		case "success", "succeeded", "completed", "passed":
			success++
		case "failed", "error":
			failed++
		}
		if build.Duration == "" {
			continue
		}
		if d, err := time.ParseDuration(build.Duration); err == nil {
			durationTotal += d
			durationCount++
		}
	}

	avgBuildTime := ""
	if durationCount > 0 {
		avgBuildTime = (durationTotal / time.Duration(durationCount)).String()
	}

	successRate := 0.0
	if total > 0 {
		successRate = float64(success) / float64(total)
	}

	return nil, DashboardMetricsOutput{
		Period: period,
		Metrics: DashboardMetrics{
			BuildsTotal:   total,
			BuildsSuccess: success,
			BuildsFailed:  failed,
			AvgBuildTime:  avgBuildTime,
			AgentSessions: sessions,
			MessagesTotal: messages,
			SuccessRate:   successRate,
		},
	}, nil
}

// dashboardState returns the current dashboard UI state as a snapshot.
//
//	out := s.dashboardState(ctx, nil, DashboardStateInput{})
func (s *Subsystem) dashboardState(_ context.Context, _ *mcp.CallToolRequest, _ DashboardStateInput) (*mcp.CallToolResult, DashboardStateOutput, error) {
	dashboardStateMu.RLock()
	defer dashboardStateMu.RUnlock()

	snapshot := make(map[string]any, len(dashboardStateStore))
	for k, v := range dashboardStateStore {
		snapshot[k] = v
	}

	return nil, DashboardStateOutput{
		State:     snapshot,
		UpdatedAt: dashboardStateUpdated,
	}, nil
}

// dashboardUpdate merges or replaces the dashboard UI state and emits an
// activity event so the IDE can react to the change.
//
//	out := s.dashboardUpdate(ctx, nil, DashboardUpdateInput{State: map[string]any{"theme": "dark"}})
func (s *Subsystem) dashboardUpdate(ctx context.Context, _ *mcp.CallToolRequest, input DashboardUpdateInput) (*mcp.CallToolResult, DashboardUpdateOutput, error) {
	now := time.Now()

	dashboardStateMu.Lock()
	if input.Replace || dashboardStateStore == nil {
		dashboardStateStore = make(map[string]any, len(input.State))
	}
	for k, v := range input.State {
		dashboardStateStore[k] = v
	}
	dashboardStateUpdated = now

	snapshot := make(map[string]any, len(dashboardStateStore))
	for k, v := range dashboardStateStore {
		snapshot[k] = v
	}
	dashboardStateMu.Unlock()

	// Record the change on the activity feed so ide_dashboard_activity
	// reflects state transitions alongside build/session events.
	s.recordActivity("dashboard_state", "dashboard state updated")

	// Push the update over the Laravel bridge when available so web clients
	// stay in sync with desktop tooling.
	if s.bridge != nil {
		_ = s.bridge.Send(BridgeMessage{
			Type: "dashboard_update",
			Data: snapshot,
		})
	}

	// Surface the change on the shared MCP notifier so connected sessions
	// receive a JSON-RPC notification alongside the tool response.
	if s.notifier != nil {
		s.notifier.ChannelSend(ctx, "dashboard.state.updated", map[string]any{
			"state":     snapshot,
			"updatedAt": now,
		})
	}

	return nil, DashboardUpdateOutput{
		State:     snapshot,
		UpdatedAt: now,
	}, nil
}

// resetDashboardState clears the shared dashboard state. Intended for tests.
func resetDashboardState() {
	dashboardStateMu.Lock()
	defer dashboardStateMu.Unlock()
	dashboardStateStore = map[string]any{}
	dashboardStateUpdated = time.Time{}
}
