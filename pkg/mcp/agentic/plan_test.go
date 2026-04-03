// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"testing"
	"time"
)

func TestPlanCheckpoint_Good_AppendsCheckpointAndMarksPhaseDone(t *testing.T) {
	root := t.TempDir()
	sub := &PrepSubsystem{codePath: root}

	plan := &Plan{
		ID:        "plan-1",
		Title:     "Test plan",
		Status:    "in_progress",
		Objective: "Verify checkpoints",
		Phases: []Phase{
			{
				Number: 1,
				Name:   "Phase 1",
				Status: "in_progress",
			},
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if _, err := writePlan(sub.plansDir(), plan); err != nil {
		t.Fatalf("writePlan failed: %v", err)
	}

	_, out, err := sub.planCheckpoint(context.Background(), nil, PlanCheckpointInput{
		ID:    plan.ID,
		Phase: 1,
		Notes: "Implementation verified",
		Done:  true,
	})
	if err != nil {
		t.Fatalf("planCheckpoint failed: %v", err)
	}
	if !out.Success {
		t.Fatal("expected checkpoint output success")
	}
	if out.Plan.Phases[0].Status != "done" {
		t.Fatalf("expected phase status done, got %q", out.Plan.Phases[0].Status)
	}
	if len(out.Plan.Phases[0].Checkpoints) != 1 {
		t.Fatalf("expected 1 checkpoint, got %d", len(out.Plan.Phases[0].Checkpoints))
	}
	if out.Plan.Phases[0].Checkpoints[0].Notes != "Implementation verified" {
		t.Fatalf("unexpected checkpoint notes: %q", out.Plan.Phases[0].Checkpoints[0].Notes)
	}
	if !out.Plan.Phases[0].Checkpoints[0].Done {
		t.Fatal("expected checkpoint to be marked done")
	}
	if out.Plan.Phases[0].Checkpoints[0].CreatedAt.IsZero() {
		t.Fatal("expected checkpoint timestamp")
	}
}
