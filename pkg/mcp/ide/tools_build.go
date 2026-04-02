package ide

import (
	"context"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Build tool input/output types.

// BuildStatusInput is the input for ide_build_status.
type BuildStatusInput struct {
	BuildID string `json:"buildId"`
}

// BuildInfo represents a single build.
type BuildInfo struct {
	ID        string    `json:"id"`
	Repo      string    `json:"repo"`
	Branch    string    `json:"branch"`
	Status    string    `json:"status"`
	Duration  string    `json:"duration,omitempty"`
	StartedAt time.Time `json:"startedAt"`
}

// BuildStatusOutput is the output for ide_build_status.
type BuildStatusOutput struct {
	Build BuildInfo `json:"build"`
}

// BuildListInput is the input for ide_build_list.
type BuildListInput struct {
	Repo  string `json:"repo,omitempty"`
	Limit int    `json:"limit,omitempty"`
}

// BuildListOutput is the output for ide_build_list.
type BuildListOutput struct {
	Builds []BuildInfo `json:"builds"`
}

// BuildLogsInput is the input for ide_build_logs.
type BuildLogsInput struct {
	BuildID string `json:"buildId"`
	Tail    int    `json:"tail,omitempty"`
}

// BuildLogsOutput is the output for ide_build_logs.
type BuildLogsOutput struct {
	BuildID string   `json:"buildId"`
	Lines   []string `json:"lines"`
}

func (s *Subsystem) registerBuildTools(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "ide_build_status",
		Description: "Get the status of a specific build",
	}, s.buildStatus)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "ide_build_list",
		Description: "List recent builds, optionally filtered by repository",
	}, s.buildList)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "ide_build_logs",
		Description: "Retrieve log output for a build",
	}, s.buildLogs)
}

// buildStatus returns a local best-effort build status and refreshes the
// Laravel backend when the bridge is available.
func (s *Subsystem) buildStatus(_ context.Context, _ *mcp.CallToolRequest, input BuildStatusInput) (*mcp.CallToolResult, BuildStatusOutput, error) {
	if s.bridge != nil {
		_ = s.bridge.Send(BridgeMessage{
			Type: "build_status",
			Data: map[string]any{"buildId": input.BuildID},
		})
	}

	build := BuildInfo{ID: input.BuildID, Status: "unknown"}
	if cached, ok := s.buildSnapshot(input.BuildID); ok {
		build = cached
	} else if input.BuildID != "" {
		s.addBuild(build)
	}

	if input.BuildID != "" {
		s.appendBuildLog(input.BuildID, "build status requested")
	}

	return nil, BuildStatusOutput{Build: build}, nil
}

// buildList returns the local build list snapshot and refreshes the Laravel
// backend when the bridge is available.
func (s *Subsystem) buildList(_ context.Context, _ *mcp.CallToolRequest, input BuildListInput) (*mcp.CallToolResult, BuildListOutput, error) {
	if s.bridge != nil {
		_ = s.bridge.Send(BridgeMessage{
			Type: "build_list",
			Data: map[string]any{"repo": input.Repo, "limit": input.Limit},
		})
	}
	return nil, BuildListOutput{Builds: s.listBuilds(input.Repo, input.Limit)}, nil
}

// buildLogs returns the local build log snapshot and refreshes the Laravel
// backend when the bridge is available.
func (s *Subsystem) buildLogs(_ context.Context, _ *mcp.CallToolRequest, input BuildLogsInput) (*mcp.CallToolResult, BuildLogsOutput, error) {
	if s.bridge != nil {
		_ = s.bridge.Send(BridgeMessage{
			Type: "build_logs",
			Data: map[string]any{"buildId": input.BuildID, "tail": input.Tail},
		})
	}
	return nil, BuildLogsOutput{
		BuildID: input.BuildID,
		Lines:   s.buildLogTail(input.BuildID, input.Tail),
	}, nil
}
