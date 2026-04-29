package agentic

import "testing"

func TestDispatch_PublicTypes(t *testing.T) {
	input := DispatchInput{Repo: "go-mcp", Task: "run audit", Agent: "codex", Variables: map[string]string{"suite": "audit"}}
	out := DispatchOutput{Success: true, Agent: input.Agent, Repo: input.Repo, WorkspaceDir: "/tmp/work"}
	if out.Agent != "codex" || input.Variables["suite"] != "audit" {
		t.Fatalf("unexpected dispatch values: %+v %+v", input, out)
	}
}
