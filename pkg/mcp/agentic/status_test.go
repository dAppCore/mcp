// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
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
