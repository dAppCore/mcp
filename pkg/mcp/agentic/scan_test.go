package agentic

import "testing"

func TestScan_PublicTypes(t *testing.T) {
	issue := ScanIssue{Repo: "go-mcp", Number: 42, Title: "Fix audit", Labels: []string{"agentic"}}
	out := ScanOutput{Success: true, Issues: []ScanIssue{issue}, Count: 1}
	if out.Issues[0].Number != 42 || out.Count != len(out.Issues) {
		t.Fatalf("unexpected scan output: %+v", out)
	}
}
