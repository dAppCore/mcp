package agentic

import "testing"

func TestEpic_PublicTypes(t *testing.T) {
	input := EpicInput{Repo: "go-mcp", Title: "Compliance", Tasks: []string{"audit"}}
	child := ChildRef{Number: 7, Title: input.Tasks[0], URL: "https://forge.example/7"}
	out := EpicOutput{Success: true, EpicNumber: 1, Children: []ChildRef{child}}
	if out.Children[0].Title != "audit" || !out.Success {
		t.Fatalf("unexpected epic output: %+v", out)
	}
}
