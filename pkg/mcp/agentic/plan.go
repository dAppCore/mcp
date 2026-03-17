// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	coreio "forge.lthn.ai/core/go-io"
	coreerr "forge.lthn.ai/core/go-log"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Plan represents an implementation plan for agent work.
type Plan struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Status    string    `json:"status"`            // draft, ready, in_progress, needs_verification, verified, approved
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
type Phase struct {
	Number   int      `json:"number"`
	Name     string   `json:"name"`
	Status   string   `json:"status"`             // pending, in_progress, done
	Criteria []string `json:"criteria,omitempty"`
	Tests    int      `json:"tests,omitempty"`
	Notes    string   `json:"notes,omitempty"`
}

// --- Input/Output types ---

// PlanCreateInput is the input for agentic_plan_create.
type PlanCreateInput struct {
	Title     string  `json:"title"`
	Objective string  `json:"objective"`
	Repo      string  `json:"repo,omitempty"`
	Org       string  `json:"org,omitempty"`
	Phases    []Phase `json:"phases,omitempty"`
	Notes     string  `json:"notes,omitempty"`
}

// PlanCreateOutput is the output for agentic_plan_create.
type PlanCreateOutput struct {
	Success bool   `json:"success"`
	ID      string `json:"id"`
	Path    string `json:"path"`
}

// PlanReadInput is the input for agentic_plan_read.
type PlanReadInput struct {
	ID string `json:"id"`
}

// PlanReadOutput is the output for agentic_plan_read.
type PlanReadOutput struct {
	Success bool `json:"success"`
	Plan    Plan `json:"plan"`
}

// PlanUpdateInput is the input for agentic_plan_update.
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
type PlanUpdateOutput struct {
	Success bool `json:"success"`
	Plan    Plan `json:"plan"`
}

// PlanDeleteInput is the input for agentic_plan_delete.
type PlanDeleteInput struct {
	ID string `json:"id"`
}

// PlanDeleteOutput is the output for agentic_plan_delete.
type PlanDeleteOutput struct {
	Success bool   `json:"success"`
	Deleted string `json:"deleted"`
}

// PlanListInput is the input for agentic_plan_list.
type PlanListInput struct {
	Status string `json:"status,omitempty"`
	Repo   string `json:"repo,omitempty"`
}

// PlanListOutput is the output for agentic_plan_list.
type PlanListOutput struct {
	Success bool   `json:"success"`
	Count   int    `json:"count"`
	Plans   []Plan `json:"plans"`
}

// --- Registration ---

func (s *PrepSubsystem) registerPlanTools(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "agentic_plan_create",
		Description: "Create an implementation plan. Plans track phased work with acceptance criteria, status lifecycle (draft → ready → in_progress → needs_verification → verified → approved), and per-phase progress.",
	}, s.planCreate)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "agentic_plan_read",
		Description: "Read an implementation plan by ID. Returns the full plan with all phases, criteria, and status.",
	}, s.planRead)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "agentic_plan_update",
		Description: "Update an implementation plan. Supports partial updates — only provided fields are changed. Use this to advance status, update phases, or add notes.",
	}, s.planUpdate)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "agentic_plan_delete",
		Description: "Delete an implementation plan by ID. Permanently removes the plan file.",
	}, s.planDelete)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "agentic_plan_list",
		Description: "List implementation plans. Supports filtering by status (draft, ready, in_progress, etc.) and repo.",
	}, s.planList)
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
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		id := strings.TrimSuffix(entry.Name(), ".json")
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

// --- Helpers ---

func (s *PrepSubsystem) plansDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "Code", "host-uk", "core", ".core", "plans")
}

func planPath(dir, id string) string {
	return filepath.Join(dir, id+".json")
}

func generatePlanID(title string) string {
	slug := strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '-' {
			return r
		}
		if r >= 'A' && r <= 'Z' {
			return r + 32
		}
		if r == ' ' {
			return '-'
		}
		return -1
	}, title)

	// Trim consecutive dashes and cap length
	for strings.Contains(slug, "--") {
		slug = strings.ReplaceAll(slug, "--", "-")
	}
	slug = strings.Trim(slug, "-")
	if len(slug) > 30 {
		slug = slug[:30]
	}
	slug = strings.TrimRight(slug, "-")

	// Append short random suffix for uniqueness
	b := make([]byte, 3)
	rand.Read(b)
	return slug + "-" + hex.EncodeToString(b)
}

func readPlan(dir, id string) (*Plan, error) {
	data, err := coreio.Local.Read(planPath(dir, id))
	if err != nil {
		return nil, coreerr.E("readPlan", "plan not found: "+id, nil)
	}

	var plan Plan
	if err := json.Unmarshal([]byte(data), &plan); err != nil {
		return nil, coreerr.E("readPlan", "failed to parse plan "+id, err)
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

	return path, coreio.Local.Write(path, string(data))
}

func validPlanStatus(status string) bool {
	switch status {
	case "draft", "ready", "in_progress", "needs_verification", "verified", "approved":
		return true
	}
	return false
}
