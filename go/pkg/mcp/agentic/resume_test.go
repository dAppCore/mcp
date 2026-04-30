package agentic

import "testing"

func TestResume_PublicTypes(t *testing.T) {
	input := ResumeInput{Workspace: "go-mcp-1", Answer: "continue", Agent: "codex"}
	out := ResumeOutput{Success: true, Workspace: input.Workspace, Agent: input.Agent, PID: 123}
	if out.Workspace != "go-mcp-1" || out.Agent != "codex" || out.PID == 0 {
		t.Fatalf("unexpected resume output: %+v", out)
	}
}
