// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"

	core "dappco.re/go/core"
	coreerr "dappco.re/go/log"
	coremcp "dappco.re/go/mcp/pkg/mcp"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// --- agentic_create_epic ---

// EpicInput is the input for agentic_create_epic.
type EpicInput struct {
	Repo     string   `json:"repo"`               // Target repo (e.g. "go-scm")
	Org      string   `json:"org,omitempty"`      // Forge org (default "core")
	Title    string   `json:"title"`              // Epic title
	Body     string   `json:"body,omitempty"`     // Epic description (above checklist)
	Tasks    []string `json:"tasks"`              // Sub-task titles (become child issues)
	Labels   []string `json:"labels,omitempty"`   // Labels for epic + children (e.g. ["agentic"])
	Dispatch bool     `json:"dispatch,omitempty"` // Auto-dispatch agents to each child
	Agent    string   `json:"agent,omitempty"`    // Agent type for dispatch (default "claude")
	Template string   `json:"template,omitempty"` // Prompt template for dispatch (default "coding")
}

// EpicOutput is the output for agentic_create_epic.
type EpicOutput struct {
	Success    bool       `json:"success"`
	EpicNumber int        `json:"epic_number"`
	EpicURL    string     `json:"epic_url"`
	Children   []ChildRef `json:"children"`
	Dispatched int        `json:"dispatched,omitempty"`
}

// ChildRef references a child issue.
type ChildRef struct {
	Number int    `json:"number"`
	Title  string `json:"title"`
	URL    string `json:"url"`
}

func (s *PrepSubsystem) registerEpicTool(svc *coremcp.Service) {
	server := svc.Server()
	coremcp.AddToolRecorded(svc, server, "agentic", &mcp.Tool{
		Name:        "agentic_create_epic",
		Description: "Create an epic issue with child issues on Forge. Each task becomes a child issue linked via checklist. Optionally auto-dispatch agents to work each child.",
	}, s.createEpic)
}

func (s *PrepSubsystem) createEpic(ctx context.Context, req *mcp.CallToolRequest, input EpicInput) (*mcp.CallToolResult, EpicOutput, error) {
	if input.Title == "" {
		return nil, EpicOutput{}, coreerr.E("createEpic", "title is required", nil)
	}
	if len(input.Tasks) == 0 {
		return nil, EpicOutput{}, coreerr.E("createEpic", "at least one task is required", nil)
	}
	if s.forgeToken == "" {
		return nil, EpicOutput{}, coreerr.E("createEpic", "no Forge token configured", nil)
	}
	if input.Org == "" {
		input.Org = "core"
	}
	if input.Agent == "" {
		input.Agent = "claude"
	}
	if input.Template == "" {
		input.Template = "coding"
	}

	// Ensure "agentic" label exists
	labels := input.Labels
	hasAgentic := false
	for _, l := range labels {
		if l == "agentic" {
			hasAgentic = true
			break
		}
	}
	if !hasAgentic {
		labels = append(labels, "agentic")
	}

	// Get label IDs
	labelIDs := s.resolveLabelIDs(ctx, input.Org, input.Repo, labels)

	// Step 1: Create child issues first (we need their numbers for the checklist)
	var children []ChildRef
	for _, task := range input.Tasks {
		child, err := s.createIssue(ctx, input.Org, input.Repo, task, "", labelIDs)
		if err != nil {
			continue // Skip failed children, create what we can
		}
		children = append(children, child)
	}

	// Step 2: Build epic body with checklist
	body := core.NewBuilder()
	if input.Body != "" {
		body.WriteString(input.Body)
		body.WriteString("\n\n")
	}
	body.WriteString("## Tasks\n\n")
	for _, child := range children {
		body.WriteString(core.Sprintf("- [ ] #%d %s\n", child.Number, child.Title))
	}

	// Step 3: Create epic issue
	epicLabels := append(labelIDs, s.resolveLabelIDs(ctx, input.Org, input.Repo, []string{"epic"})...)
	epic, err := s.createIssue(ctx, input.Org, input.Repo, input.Title, body.String(), epicLabels)
	if err != nil {
		return nil, EpicOutput{}, coreerr.E("createEpic", "failed to create epic", err)
	}

	out := EpicOutput{
		Success:    true,
		EpicNumber: epic.Number,
		EpicURL:    epic.URL,
		Children:   children,
	}

	// Step 4: Optionally dispatch agents to each child
	if input.Dispatch {
		for _, child := range children {
			_, _, err := s.dispatch(ctx, req, DispatchInput{
				Repo:     input.Repo,
				Org:      input.Org,
				Task:     child.Title,
				Agent:    input.Agent,
				Template: input.Template,
				Issue:    child.Number,
			})
			if err == nil {
				out.Dispatched++
			}
		}
	}

	return nil, out, nil
}

// createIssue creates a single issue on Forge and returns its reference.
func (s *PrepSubsystem) createIssue(ctx context.Context, org, repo, title, body string, labelIDs []int64) (ChildRef, error) {
	payload := map[string]any{
		"title": title,
	}
	if body != "" {
		payload["body"] = body
	}
	if len(labelIDs) > 0 {
		payload["labels"] = labelIDs
	}

	r := core.JSONMarshal(payload)
	if !r.OK {
		return ChildRef{}, coreerr.E("createIssue", "failed to encode issue payload", nil)
	}
	data := r.Value.([]byte)
	url := core.Sprintf("%s/api/v1/repos/%s/%s/issues", s.forgeURL, org, repo)
	req, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "token "+s.forgeToken)

	resp, err := s.client.Do(req)
	if err != nil {
		return ChildRef{}, coreerr.E("createIssue", "request failed", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 201 {
		return ChildRef{}, coreerr.E("createIssue", core.Sprintf("returned %d", resp.StatusCode), nil)
	}

	var result struct {
		Number  int    `json:"number"`
		HTMLURL string `json:"html_url"`
	}
	json.NewDecoder(resp.Body).Decode(&result)

	return ChildRef{
		Number: result.Number,
		Title:  title,
		URL:    result.HTMLURL,
	}, nil
}

// resolveLabelIDs looks up label IDs by name, creating labels that don't exist.
func (s *PrepSubsystem) resolveLabelIDs(ctx context.Context, org, repo string, names []string) []int64 {
	if len(names) == 0 {
		return nil
	}

	// Fetch existing labels
	url := core.Sprintf("%s/api/v1/repos/%s/%s/labels?limit=50", s.forgeURL, org, repo)
	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	req.Header.Set("Authorization", "token "+s.forgeToken)

	resp, err := s.client.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil
	}

	var existing []struct {
		ID   int64  `json:"id"`
		Name string `json:"name"`
	}
	json.NewDecoder(resp.Body).Decode(&existing)

	nameToID := make(map[string]int64)
	for _, l := range existing {
		nameToID[l.Name] = l.ID
	}

	var ids []int64
	for _, name := range names {
		if id, ok := nameToID[name]; ok {
			ids = append(ids, id)
		} else {
			// Create the label
			id := s.createLabel(ctx, org, repo, name)
			if id > 0 {
				ids = append(ids, id)
			}
		}
	}

	return ids
}

// createLabel creates a label on Forge and returns its ID.
func (s *PrepSubsystem) createLabel(ctx context.Context, org, repo, name string) int64 {
	colours := map[string]string{
		"agentic":     "#7c3aed",
		"epic":        "#dc2626",
		"bug":         "#ef4444",
		"help-wanted": "#22c55e",
	}
	colour := colours[name]
	if colour == "" {
		colour = "#6b7280"
	}

	r := core.JSONMarshal(map[string]string{
		"name":  name,
		"color": colour,
	})
	if !r.OK {
		return 0
	}
	payload := r.Value.([]byte)

	url := core.Sprintf("%s/api/v1/repos/%s/%s/labels", s.forgeURL, org, repo)
	req, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "token "+s.forgeToken)

	resp, err := s.client.Do(req)
	if err != nil {
		return 0
	}
	defer resp.Body.Close()
	if resp.StatusCode != 201 {
		return 0
	}

	var result struct {
		ID int64 `json:"id"`
	}
	json.NewDecoder(resp.Body).Decode(&result)
	return result.ID
}

// listOrgRepos is defined in pr.go
