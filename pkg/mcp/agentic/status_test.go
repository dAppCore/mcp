// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"path/filepath"
	"testing"
	"time"
)

func TestStatus_Good_EmptyWorkspaceSet(t *testing.T) {
	sub := &PrepSubsystem{codePath: t.TempDir()}

	_, out, err := sub.status(context.Background(), nil, StatusInput{})
	if err != nil {
		t.Fatalf("status failed: %v", err)
	}
	if out.Count != 0 {
		t.Fatalf("expected count 0, got %d", out.Count)
	}
	if len(out.Workspaces) != 0 {
		t.Fatalf("expected empty workspace list, got %d entries", len(out.Workspaces))
	}
}

func TestPlanRead_Good_ReturnsWrittenPlan(t *testing.T) {
	sub := &PrepSubsystem{codePath: t.TempDir()}

	plan := &Plan{
		ID:        "plan-1",
		Title:     "Read me",
		Status:    "ready",
		Objective: "Verify plan reads",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if _, err := writePlan(sub.plansDir(), plan); err != nil {
		t.Fatalf("writePlan failed: %v", err)
	}

	_, out, err := sub.planRead(context.Background(), nil, PlanReadInput{ID: plan.ID})
	if err != nil {
		t.Fatalf("planRead failed: %v", err)
	}
	if !out.Success {
		t.Fatal("expected success output")
	}
	if out.Plan.ID != plan.ID {
		t.Fatalf("expected plan %q, got %q", plan.ID, out.Plan.ID)
	}
	if out.Plan.Title != plan.Title {
		t.Fatalf("expected title %q, got %q", plan.Title, out.Plan.Title)
	}
}

func TestStatus_Good_ExposesWorkspaceMetadata(t *testing.T) {
	root := t.TempDir()
	sub := &PrepSubsystem{codePath: root}

	wsDir := filepath.Join(root, ".core", "workspace", "repo-123")
	plan := &WorkspaceStatus{
		Status: "completed",
		Agent:  "claude",
		Repo:   "go-mcp",
		Branch: "agent/issue-42-fix-status",
		Issue:  42,
		PRURL:  "https://forge.example/pr/42",
		Task:   "Fix status output",
		Runs:   2,
	}
	if err := writeStatus(wsDir, plan); err != nil {
		t.Fatalf("writeStatus failed: %v", err)
	}

	_, out, err := sub.status(context.Background(), nil, StatusInput{})
	if err != nil {
		t.Fatalf("status failed: %v", err)
	}
	if out.Count != 1 {
		t.Fatalf("expected count 1, got %d", out.Count)
	}

	info := out.Workspaces[0]
	if info.Branch != plan.Branch {
		t.Fatalf("expected branch %q, got %q", plan.Branch, info.Branch)
	}
	if info.Issue != plan.Issue {
		t.Fatalf("expected issue %d, got %d", plan.Issue, info.Issue)
	}
	if info.PRURL != plan.PRURL {
		t.Fatalf("expected PR URL %q, got %q", plan.PRURL, info.PRURL)
	}
}
