// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"testing"
	"time"
)

// newPlanSub returns a PrepSubsystem rooted at a fresh temp dir.
func newPlanSub(t *testing.T) *PrepSubsystem {
	t.Helper()
	return &PrepSubsystem{codePath: t.TempDir()}
}

func TestPlan_planCreate_Good(t *testing.T) {
	sub := newPlanSub(t)

	_, out, err := sub.planCreate(context.Background(), nil, PlanCreateInput{
		Title:     "Add Notifications",
		Objective: "Broadcast MCP events",
		Repo:      "core/mcp",
		Org:       "core",
		Phases: []Phase{
			{Name: "Implementation"},
			{Name: "Tests"},
		},
		Notes: "first cut",
	})
	if err != nil {
		t.Fatalf("planCreate: %v", err)
	}
	if !out.Success {
		t.Fatal("expected success")
	}
	if out.ID == "" {
		t.Fatal("expected non-empty id")
	}
	if out.Path == "" {
		t.Fatal("expected non-empty path")
	}

	// Round-trip: the plan should read back with defaulted phase fields.
	got, err := readPlan(sub.plansDir(), out.ID)
	if err != nil {
		t.Fatalf("readPlan: %v", err)
	}
	if got.Status != "draft" {
		t.Fatalf("expected status draft, got %q", got.Status)
	}
	if len(got.Phases) != 2 {
		t.Fatalf("expected 2 phases, got %d", len(got.Phases))
	}
	if got.Phases[0].Status != "pending" || got.Phases[1].Status != "pending" {
		t.Fatalf("expected phases defaulted to pending, got %q/%q", got.Phases[0].Status, got.Phases[1].Status)
	}
	if got.Phases[0].Number != 1 || got.Phases[1].Number != 2 {
		t.Fatalf("expected phase numbers 1,2; got %d,%d", got.Phases[0].Number, got.Phases[1].Number)
	}
}

func TestPlan_planCreate_Bad_MissingTitle(t *testing.T) {
	sub := newPlanSub(t)

	if _, _, err := sub.planCreate(context.Background(), nil, PlanCreateInput{
		Objective: "no title here",
	}); err == nil {
		t.Fatal("expected error for missing title")
	}
}

func TestPlan_planCreate_Bad_MissingObjective(t *testing.T) {
	sub := newPlanSub(t)

	if _, _, err := sub.planCreate(context.Background(), nil, PlanCreateInput{
		Title: "no objective here",
	}); err == nil {
		t.Fatal("expected error for missing objective")
	}
}

func TestPlan_planRead_Good(t *testing.T) {
	sub := newPlanSub(t)

	_, created, err := sub.planCreate(context.Background(), nil, PlanCreateInput{
		Title:     "Readable Plan",
		Objective: "Be readable",
	})
	if err != nil {
		t.Fatalf("planCreate: %v", err)
	}

	_, out, err := sub.planRead(context.Background(), nil, PlanReadInput{ID: created.ID})
	if err != nil {
		t.Fatalf("planRead: %v", err)
	}
	if !out.Success {
		t.Fatal("expected success")
	}
	if out.Plan.Title != "Readable Plan" {
		t.Fatalf("unexpected title %q", out.Plan.Title)
	}
}

func TestPlan_planRead_Bad_MissingID(t *testing.T) {
	sub := newPlanSub(t)

	if _, _, err := sub.planRead(context.Background(), nil, PlanReadInput{}); err == nil {
		t.Fatal("expected error for missing id")
	}
}

func TestPlan_planRead_Ugly_NotFound(t *testing.T) {
	sub := newPlanSub(t)

	if _, _, err := sub.planRead(context.Background(), nil, PlanReadInput{ID: "does-not-exist"}); err == nil {
		t.Fatal("expected error for unknown plan id")
	}
}

func TestPlan_planUpdate_Good_PartialFields(t *testing.T) {
	sub := newPlanSub(t)

	_, created, err := sub.planCreate(context.Background(), nil, PlanCreateInput{
		Title:     "Updatable",
		Objective: "original objective",
	})
	if err != nil {
		t.Fatalf("planCreate: %v", err)
	}

	_, out, err := sub.planUpdate(context.Background(), nil, PlanUpdateInput{
		ID:        created.ID,
		Status:    "ready",
		Objective: "new objective",
		Agent:     "hephaestus",
		Notes:     "updated",
	})
	if err != nil {
		t.Fatalf("planUpdate: %v", err)
	}
	if out.Plan.Status != "ready" {
		t.Fatalf("expected status ready, got %q", out.Plan.Status)
	}
	if out.Plan.Objective != "new objective" {
		t.Fatalf("expected objective updated, got %q", out.Plan.Objective)
	}
	if out.Plan.Agent != "hephaestus" {
		t.Fatalf("expected agent set, got %q", out.Plan.Agent)
	}
	// Title was not provided; it must be preserved.
	if out.Plan.Title != "Updatable" {
		t.Fatalf("expected title preserved, got %q", out.Plan.Title)
	}
}

func TestPlan_planUpdate_Bad_MissingID(t *testing.T) {
	sub := newPlanSub(t)

	if _, _, err := sub.planUpdate(context.Background(), nil, PlanUpdateInput{Status: "ready"}); err == nil {
		t.Fatal("expected error for missing id")
	}
}

func TestPlan_planUpdate_Ugly_InvalidStatus(t *testing.T) {
	sub := newPlanSub(t)

	_, created, err := sub.planCreate(context.Background(), nil, PlanCreateInput{
		Title:     "Status Guard",
		Objective: "reject bad status",
	})
	if err != nil {
		t.Fatalf("planCreate: %v", err)
	}

	if _, _, err := sub.planUpdate(context.Background(), nil, PlanUpdateInput{
		ID:     created.ID,
		Status: "totally-not-valid",
	}); err == nil {
		t.Fatal("expected error for invalid status")
	}
}

func TestPlan_planUpdate_Ugly_NotFound(t *testing.T) {
	sub := newPlanSub(t)

	if _, _, err := sub.planUpdate(context.Background(), nil, PlanUpdateInput{
		ID:    "nope",
		Notes: "x",
	}); err == nil {
		t.Fatal("expected error for unknown plan id")
	}
}

func TestPlan_planDelete_Good(t *testing.T) {
	sub := newPlanSub(t)

	_, created, err := sub.planCreate(context.Background(), nil, PlanCreateInput{
		Title:     "Doomed",
		Objective: "to be deleted",
	})
	if err != nil {
		t.Fatalf("planCreate: %v", err)
	}

	_, out, err := sub.planDelete(context.Background(), nil, PlanDeleteInput{ID: created.ID})
	if err != nil {
		t.Fatalf("planDelete: %v", err)
	}
	if out.Deleted != created.ID {
		t.Fatalf("expected deleted %q, got %q", created.ID, out.Deleted)
	}

	// A second read must now fail.
	if _, _, err := sub.planRead(context.Background(), nil, PlanReadInput{ID: created.ID}); err == nil {
		t.Fatal("expected read of deleted plan to fail")
	}
}

func TestPlan_planDelete_Bad_MissingID(t *testing.T) {
	sub := newPlanSub(t)

	if _, _, err := sub.planDelete(context.Background(), nil, PlanDeleteInput{}); err == nil {
		t.Fatal("expected error for missing id")
	}
}

func TestPlan_planDelete_Ugly_NotFound(t *testing.T) {
	sub := newPlanSub(t)

	if _, _, err := sub.planDelete(context.Background(), nil, PlanDeleteInput{ID: "ghost"}); err == nil {
		t.Fatal("expected error for unknown plan id")
	}
}

func TestPlan_planList_Good_FiltersByStatusAndRepo(t *testing.T) {
	sub := newPlanSub(t)
	ctx := context.Background()

	// Three plans: two draft (one in core/mcp), one ready.
	if _, _, err := sub.planCreate(ctx, nil, PlanCreateInput{Title: "A draft mcp", Objective: "o", Repo: "core/mcp"}); err != nil {
		t.Fatalf("create A: %v", err)
	}
	if _, _, err := sub.planCreate(ctx, nil, PlanCreateInput{Title: "B draft other", Objective: "o", Repo: "core/api"}); err != nil {
		t.Fatalf("create B: %v", err)
	}
	_, readyPlan, err := sub.planCreate(ctx, nil, PlanCreateInput{Title: "C plan", Objective: "o", Repo: "core/mcp"})
	if err != nil {
		t.Fatalf("create C: %v", err)
	}
	if _, _, err := sub.planUpdate(ctx, nil, PlanUpdateInput{ID: readyPlan.ID, Status: "ready"}); err != nil {
		t.Fatalf("update C: %v", err)
	}

	// No filter: all three.
	_, all, err := sub.planList(ctx, nil, PlanListInput{})
	if err != nil {
		t.Fatalf("planList all: %v", err)
	}
	if all.Count != 3 {
		t.Fatalf("expected 3 plans, got %d", all.Count)
	}

	// Status filter.
	_, drafts, err := sub.planList(ctx, nil, PlanListInput{Status: "draft"})
	if err != nil {
		t.Fatalf("planList draft: %v", err)
	}
	if drafts.Count != 2 {
		t.Fatalf("expected 2 draft plans, got %d", drafts.Count)
	}

	// Repo filter.
	_, mcpPlans, err := sub.planList(ctx, nil, PlanListInput{Repo: "core/mcp"})
	if err != nil {
		t.Fatalf("planList repo: %v", err)
	}
	if mcpPlans.Count != 2 {
		t.Fatalf("expected 2 core/mcp plans, got %d", mcpPlans.Count)
	}
}

func TestPlan_planList_Ugly_EmptyDir(t *testing.T) {
	sub := newPlanSub(t)

	_, out, err := sub.planList(context.Background(), nil, PlanListInput{})
	if err != nil {
		t.Fatalf("planList on empty dir: %v", err)
	}
	if out.Count != 0 {
		t.Fatalf("expected 0 plans, got %d", out.Count)
	}
}

func TestPlan_planCheckpoint_Bad_MissingID(t *testing.T) {
	sub := newPlanSub(t)

	if _, _, err := sub.planCheckpoint(context.Background(), nil, PlanCheckpointInput{Phase: 1, Done: true}); err == nil {
		t.Fatal("expected error for missing id")
	}
}

func TestPlan_planCheckpoint_Bad_NonPositivePhase(t *testing.T) {
	sub := newPlanSub(t)

	if _, _, err := sub.planCheckpoint(context.Background(), nil, PlanCheckpointInput{ID: "x", Phase: 0, Done: true}); err == nil {
		t.Fatal("expected error for non-positive phase")
	}
}

func TestPlan_planCheckpoint_Bad_NoNotesNoDone(t *testing.T) {
	sub := newPlanSub(t)

	if _, _, err := sub.planCheckpoint(context.Background(), nil, PlanCheckpointInput{ID: "x", Phase: 1}); err == nil {
		t.Fatal("expected error when neither notes nor done provided")
	}
}

func TestPlan_planCheckpoint_Ugly_PhaseOutOfRange(t *testing.T) {
	sub := newPlanSub(t)

	plan := &Plan{
		ID:        "single-phase",
		Title:     "Single",
		Status:    "in_progress",
		Objective: "o",
		Phases:    []Phase{{Number: 1, Name: "Phase 1", Status: "pending"}},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if _, err := writePlan(sub.plansDir(), plan); err != nil {
		t.Fatalf("writePlan: %v", err)
	}

	if _, _, err := sub.planCheckpoint(context.Background(), nil, PlanCheckpointInput{
		ID:    plan.ID,
		Phase: 5,
		Done:  true,
	}); err == nil {
		t.Fatal("expected error for phase out of range")
	}
}

func TestPlan_validPlanStatus_Good_Bad(t *testing.T) {
	valid := []string{"draft", "ready", "in_progress", "needs_verification", "verified", "approved"}
	for _, s := range valid {
		if !validPlanStatus(s) {
			t.Fatalf("expected %q to be valid", s)
		}
	}
	for _, s := range []string{"", "done", "DRAFT", "approved "} {
		if validPlanStatus(s) {
			t.Fatalf("expected %q to be invalid", s)
		}
	}
}

func TestPlan_generatePlanID_Good_SlugAndUnique(t *testing.T) {
	id := generatePlanID("Add  Notifications!! To The   System")
	// Punctuation dropped, spaces become single dashes, lower-cased, suffix appended.
	if id == "" {
		t.Fatal("expected non-empty id")
	}
	for _, r := range id {
		ok := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-'
		if !ok {
			t.Fatalf("unexpected rune %q in id %q", r, id)
		}
	}
	// Two calls with the same title must differ (random suffix).
	if id == generatePlanID("Add  Notifications!! To The   System") {
		t.Fatal("expected unique ids across calls")
	}
}

func TestPlan_generatePlanID_Ugly_LongTitleCapped(t *testing.T) {
	long := "This Is An Extremely Long Plan Title That Should Be Capped For Filesystem Safety"
	id := generatePlanID(long)
	// slug capped at 30, plus "-" and 6 hex chars = 37 max.
	if len(id) > 37 {
		t.Fatalf("expected id length <= 37, got %d (%q)", len(id), id)
	}
}
