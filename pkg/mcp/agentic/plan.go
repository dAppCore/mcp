// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"time"

	core "dappco.re/go/core"
	coreio "dappco.re/go/core/io"
	coreerr "dappco.re/go/core/log"
	coremcp "dappco.re/go/mcp/pkg/mcp"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Plan represents an implementation plan for agent work.
//
//	plan := Plan{
//	    Title:  "Add notifications",
//	    Status: "draft",
//	}
type Plan struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Status    string    `json:"status"` // draft, ready, in_progress, needs_verification, verified, approved
	Repo      string    `json:"repo,omitempty"`
	Org       string    `json:"org,omitempty"`
	Objective string    `json:"objective"`
	Phases    []Phase   `json:"phases,omitempty"`
	Notes     string    `json:"notes,omitempty"`
	Agent     string    `json:"agent,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Phase represents a phase within an implementation plan.
//
//	phase := Phase{Name: "Implementation", Status: "pending"}
type Phase struct {
	Number      int          `json:"number"`
	Name        string       `json:"name"`
	Status      string       `json:"status"` // pending, in_progress, done
	Criteria    []string     `json:"criteria,omitempty"`
	Tests       int          `json:"tests,omitempty"`
	Notes       string       `json:"notes,omitempty"`
	Checkpoints []Checkpoint `json:"checkpoints,omitempty"`
}

// Checkpoint records phase progress or completion details.
//
//	cp := Checkpoint{Notes: "Implemented transport hooks", Done: true}
type Checkpoint struct {
	Notes     string    `json:"notes,omitempty"`
	Done      bool      `json:"done,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// --- Input/Output types ---

// PlanCreateInput is the input for agentic_plan_create.
//
//	input := PlanCreateInput{Title: "Add notifications", Objective: "Broadcast MCP events"}
type PlanCreateInput struct {
	Title     string  `json:"title"`
	Objective string  `json:"objective"`
	Repo      string  `json:"repo,omitempty"`
	Org       string  `json:"org,omitempty"`
	Phases    []Phase `json:"phases,omitempty"`
	Notes     string  `json:"notes,omitempty"`
}

// PlanCreateOutput is the output for agentic_plan_create.
//
//	// out.Success == true, out.ID != ""
type PlanCreateOutput struct {
	Success bool   `json:"success"`
	ID      string `json:"id"`
	Path    string `json:"path"`
}

// PlanReadInput is the input for agentic_plan_read.
//
//	input := PlanReadInput{ID: "add-notifications"}
type PlanReadInput struct {
	ID string `json:"id"`
}

// PlanReadOutput is the output for agentic_plan_read.
//
//	// out.Plan.Title == "Add notifications"
type PlanReadOutput struct {
	Success bool `json:"success"`
	Plan    Plan `json:"plan"`
}

// PlanUpdateInput is the input for agentic_plan_update.
//
//	input := PlanUpdateInput{ID: "add-notifications", Status: "ready"}
type PlanUpdateInput struct {
	ID        string  `json:"id"`
	Status    string  `json:"status,omitempty"`
	Title     string  `json:"title,omitempty"`
	Objective string  `json:"objective,omitempty"`
	Phases    []Phase `json:"phases,omitempty"`
	Notes     string  `json:"notes,omitempty"`
	Agent     string  `json:"agent,omitempty"`
}

// PlanUpdateOutput is the output for agentic_plan_update.
//
//	// out.Plan.Status == "ready"
type PlanUpdateOutput struct {
	Success bool `json:"success"`
	Plan    Plan `json:"plan"`
}

// PlanDeleteInput is the input for agentic_plan_delete.
//
//	input := PlanDeleteInput{ID: "add-notifications"}
type PlanDeleteInput struct {
	ID string `json:"id"`
}

// PlanDeleteOutput is the output for agentic_plan_delete.
//
//	// out.Deleted == "add-notifications"
type PlanDeleteOutput struct {
	Success bool   `json:"success"`
	Deleted string `json:"deleted"`
}

// PlanListInput is the input for agentic_plan_list.
//
//	input := PlanListInput{Status: "draft"}
type PlanListInput struct {
	Status string `json:"status,omitempty"`
	Repo   string `json:"repo,omitempty"`
}

// PlanListOutput is the output for agentic_plan_list.
//
//	// len(out.Plans) >= 1
type PlanListOutput struct {
	Success bool   `json:"success"`
	Count   int    `json:"count"`
	Plans   []Plan `json:"plans"`
}

// PlanCheckpointInput is the input for agentic_plan_checkpoint.
//
//	input := PlanCheckpointInput{ID: "add-notifications", Phase: 1, Done: true}
type PlanCheckpointInput struct {
	ID    string `json:"id"`
	Phase int    `json:"phase"`
	Notes string `json:"notes,omitempty"`
	Done  bool   `json:"done,omitempty"`
}

// PlanCheckpointOutput is the output for agentic_plan_checkpoint.
//
//	// out.Plan.Phases[0].Status == "done"
type PlanCheckpointOutput struct {
	Success bool `json:"success"`
	Plan    Plan `json:"plan"`
}

// --- Registration ---

func (s *PrepSubsystem) registerPlanTools(svc *coremcp.Service) {
	server := svc.Server()
	coremcp.AddToolRecorded(svc, server, "agentic", &mcp.Tool{
		Name:        "agentic_plan_create",
		Description: "Create an implementation plan. Plans track phased work with acceptance criteria, status lifecycle (draft → ready → in_progress → needs_verification → verified → approved), and per-phase progress.",
	}, s.planCreate)

	coremcp.AddToolRecorded(svc, server, "agentic", &mcp.Tool{
		Name:        "agentic_plan_read",
		Description: "Read an implementation plan by ID. Returns the full plan with all phases, criteria, and status.",
	}, s.planRead)

	// agentic_plan_status is kept as a user-facing alias for the read tool.
	coremcp.AddToolRecorded(svc, server, "agentic", &mcp.Tool{
		Name:        "agentic_plan_status",
		Description: "Get the current status of an implementation plan by ID. Returns the full plan with all phases, criteria, and status.",
	}, s.planRead)

	coremcp.AddToolRecorded(svc, server, "agentic", &mcp.Tool{
		Name:        "agentic_plan_update",
		Description: "Update an implementation plan. Supports partial updates — only provided fields are changed. Use this to advance status, update phases, or add notes.",
	}, s.planUpdate)

	coremcp.AddToolRecorded(svc, server, "agentic", &mcp.Tool{
		Name:        "agentic_plan_delete",
		Description: "Delete an implementation plan by ID. Permanently removes the plan file.",
	}, s.planDelete)

	coremcp.AddToolRecorded(svc, server, "agentic", &mcp.Tool{
		Name:        "agentic_plan_list",
		Description: "List implementation plans. Supports filtering by status (draft, ready, in_progress, etc.) and repo.",
	}, s.planList)

	coremcp.AddToolRecorded(svc, server, "agentic", &mcp.Tool{
		Name:        "agentic_plan_checkpoint",
		Description: "Record a checkpoint for a plan phase and optionally mark the phase done.",
	}, s.planCheckpoint)
}

// --- Handlers ---

func (s *PrepSubsystem) planCreate(_ context.Context, _ *mcp.CallToolRequest, input PlanCreateInput) (*mcp.CallToolResult, PlanCreateOutput, error) {
	if input.Title == "" {
		return nil, PlanCreateOutput{}, coreerr.E("planCreate", "title is required", nil)
	}
	if input.Objective == "" {
		return nil, PlanCreateOutput{}, coreerr.E("planCreate", "objective is required", nil)
	}

	id := generatePlanID(input.Title)
	plan := Plan{
		ID:        id,
		Title:     input.Title,
		Status:    "draft",
		Repo:      input.Repo,
		Org:       input.Org,
		Objective: input.Objective,
		Phases:    input.Phases,
		Notes:     input.Notes,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Default phase status to pending
	for i := range plan.Phases {
		if plan.Phases[i].Status == "" {
			plan.Phases[i].Status = "pending"
		}
		if plan.Phases[i].Number == 0 {
			plan.Phases[i].Number = i + 1
		}
	}

	path, err := writePlan(s.plansDir(), &plan)
	if err != nil {
		return nil, PlanCreateOutput{}, coreerr.E("planCreate", "failed to write plan", err)
	}

	return nil, PlanCreateOutput{
		Success: true,
		ID:      id,
		Path:    path,
	}, nil
}

func (s *PrepSubsystem) planRead(_ context.Context, _ *mcp.CallToolRequest, input PlanReadInput) (*mcp.CallToolResult, PlanReadOutput, error) {
	if input.ID == "" {
		return nil, PlanReadOutput{}, coreerr.E("planRead", "id is required", nil)
	}

	plan, err := readPlan(s.plansDir(), input.ID)
	if err != nil {
		return nil, PlanReadOutput{}, err
	}

	return nil, PlanReadOutput{
		Success: true,
		Plan:    *plan,
	}, nil
}

func (s *PrepSubsystem) planUpdate(_ context.Context, _ *mcp.CallToolRequest, input PlanUpdateInput) (*mcp.CallToolResult, PlanUpdateOutput, error) {
	if input.ID == "" {
		return nil, PlanUpdateOutput{}, coreerr.E("planUpdate", "id is required", nil)
	}

	plan, err := readPlan(s.plansDir(), input.ID)
	if err != nil {
		return nil, PlanUpdateOutput{}, err
	}

	// Apply partial updates
	if input.Status != "" {
		if !validPlanStatus(input.Status) {
			return nil, PlanUpdateOutput{}, coreerr.E("planUpdate", "invalid status: "+input.Status+" (valid: draft, ready, in_progress, needs_verification, verified, approved)", nil)
		}
		plan.Status = input.Status
	}
	if input.Title != "" {
		plan.Title = input.Title
	}
	if input.Objective != "" {
		plan.Objective = input.Objective
	}
	if input.Phases != nil {
		plan.Phases = input.Phases
	}
	if input.Notes != "" {
		plan.Notes = input.Notes
	}
	if input.Agent != "" {
		plan.Agent = input.Agent
	}

	plan.UpdatedAt = time.Now()

	if _, err := writePlan(s.plansDir(), plan); err != nil {
		return nil, PlanUpdateOutput{}, coreerr.E("planUpdate", "failed to write plan", err)
	}

	return nil, PlanUpdateOutput{
		Success: true,
		Plan:    *plan,
	}, nil
}

func (s *PrepSubsystem) planDelete(_ context.Context, _ *mcp.CallToolRequest, input PlanDeleteInput) (*mcp.CallToolResult, PlanDeleteOutput, error) {
	if input.ID == "" {
		return nil, PlanDeleteOutput{}, coreerr.E("planDelete", "id is required", nil)
	}

	path := planPath(s.plansDir(), input.ID)
	if !coreio.Local.IsFile(path) {
		return nil, PlanDeleteOutput{}, coreerr.E("planDelete", "plan not found: "+input.ID, nil)
	}

	if err := coreio.Local.Delete(path); err != nil {
		return nil, PlanDeleteOutput{}, coreerr.E("planDelete", "failed to delete plan", err)
	}

	return nil, PlanDeleteOutput{
		Success: true,
		Deleted: input.ID,
	}, nil
}

func (s *PrepSubsystem) planList(_ context.Context, _ *mcp.CallToolRequest, input PlanListInput) (*mcp.CallToolResult, PlanListOutput, error) {
	dir := s.plansDir()
	if err := coreio.Local.EnsureDir(dir); err != nil {
		return nil, PlanListOutput{}, coreerr.E("planList", "failed to access plans directory", err)
	}

	entries, err := coreio.Local.List(dir)
	if err != nil {
		return nil, PlanListOutput{}, coreerr.E("planList", "failed to read plans directory", err)
	}

	var plans []Plan
	for _, entry := range entries {
		if entry.IsDir() || !core.HasSuffix(entry.Name(), ".json") {
			continue
		}

		id := core.TrimSuffix(entry.Name(), ".json")
		plan, err := readPlan(dir, id)
		if err != nil {
			continue
		}

		// Apply filters
		if input.Status != "" && plan.Status != input.Status {
			continue
		}
		if input.Repo != "" && plan.Repo != input.Repo {
			continue
		}

		plans = append(plans, *plan)
	}

	return nil, PlanListOutput{
		Success: true,
		Count:   len(plans),
		Plans:   plans,
	}, nil
}

func (s *PrepSubsystem) planCheckpoint(_ context.Context, _ *mcp.CallToolRequest, input PlanCheckpointInput) (*mcp.CallToolResult, PlanCheckpointOutput, error) {
	if input.ID == "" {
		return nil, PlanCheckpointOutput{}, coreerr.E("planCheckpoint", "id is required", nil)
	}
	if input.Phase <= 0 {
		return nil, PlanCheckpointOutput{}, coreerr.E("planCheckpoint", "phase must be greater than zero", nil)
	}
	if input.Notes == "" && !input.Done {
		return nil, PlanCheckpointOutput{}, coreerr.E("planCheckpoint", "notes or done is required", nil)
	}

	plan, err := readPlan(s.plansDir(), input.ID)
	if err != nil {
		return nil, PlanCheckpointOutput{}, err
	}

	phaseIndex := input.Phase - 1
	if phaseIndex >= len(plan.Phases) {
		return nil, PlanCheckpointOutput{}, coreerr.E("planCheckpoint", "phase not found", nil)
	}

	phase := &plan.Phases[phaseIndex]
	phase.Checkpoints = append(phase.Checkpoints, Checkpoint{
		Notes:     input.Notes,
		Done:      input.Done,
		CreatedAt: time.Now(),
	})
	if input.Done {
		phase.Status = "done"
	}

	plan.UpdatedAt = time.Now()
	if _, err := writePlan(s.plansDir(), plan); err != nil {
		return nil, PlanCheckpointOutput{}, coreerr.E("planCheckpoint", "failed to write plan", err)
	}

	return nil, PlanCheckpointOutput{
		Success: true,
		Plan:    *plan,
	}, nil
}

// --- Helpers ---

func (s *PrepSubsystem) plansDir() string {
	return core.Path(s.codePath, ".core", "plans")
}

func planPath(dir, id string) string {
	return core.Path(dir, id+".json")
}

func generatePlanID(title string) string {
	b := core.NewBuilder()
	b.Grow(len(title))
	for _, r := range title {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '-':
			b.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			b.WriteRune(r + 32)
		case r == ' ':
			b.WriteByte('-')
		}
	}
	slug := b.String()

	// Collapse consecutive dashes and cap length
	for core.Contains(slug, "--") {
		slug = core.Replace(slug, "--", "-")
	}
	slug = trimDashes(slug)
	if len(slug) > 30 {
		slug = trimDashes(slug[:30])
	}

	// Append short random suffix for uniqueness
	rnd := make([]byte, 3)
	rand.Read(rnd)
	return slug + "-" + hex.EncodeToString(rnd)
}

func readPlan(dir, id string) (*Plan, error) {
	data, err := coreio.Local.Read(planPath(dir, id))
	if err != nil {
		return nil, coreerr.E("readPlan", "plan not found: "+id, err)
	}

	var plan Plan
	if r := core.JSONUnmarshal([]byte(data), &plan); !r.OK {
		return nil, coreerr.E("readPlan", "failed to parse plan "+id, nil)
	}
	return &plan, nil
}

func writePlan(dir string, plan *Plan) (string, error) {
	if err := coreio.Local.EnsureDir(dir); err != nil {
		return "", coreerr.E("writePlan", "failed to create plans directory", err)
	}

	path := planPath(dir, plan.ID)
	data, err := json.MarshalIndent(plan, "", "  ")
	if err != nil {
		return "", err
	}

	return path, writeAtomic(path, string(data))
}

func validPlanStatus(status string) bool {
	switch status {
	case "draft", "ready", "in_progress", "needs_verification", "verified", "approved":
		return true
	}
	return false
}
