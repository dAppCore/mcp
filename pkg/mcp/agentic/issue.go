// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"

	core "dappco.re/go"
	coremcp "dappco.re/go/mcp/pkg/mcp"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// IssueDispatchInput is the input for agentic_dispatch_issue.
//
//	input := IssueDispatchInput{
//	    Repo:  "go-io",
//	    Issue: 123,
//	    Agent: "claude",
//	}
type IssueDispatchInput struct {
	Repo     string `json:"repo"`               // Target repo (e.g. "go-io")
	Org      string `json:"org,omitempty"`      // Forge org (default "core")
	Issue    int    `json:"issue"`              // Forge issue number
	Agent    string `json:"agent,omitempty"`    // "claude" (default), "codex", "gemini"
	Template string `json:"template,omitempty"` // "conventions", "security", "coding" (default)
	DryRun   bool   `json:"dry_run,omitempty"`  // Preview without executing
}

type forgeIssue struct {
	Title  string `json:"title"`
	Body   string `json:"body"`
	State  string `json:"state"`
	Labels []struct {
		Name string `json:"name"`
	} `json:"labels"`
	Assignee *struct {
		Login string `json:"login"`
	} `json:"assignee"`
}

func (s *PrepSubsystem) registerIssueTools(svc *coremcp.Service) {
	server := svc.Server()
	coremcp.AddToolRecorded(svc, server, "agentic", &mcp.Tool{
		Name:        "agentic_dispatch_issue",
		Description: "Dispatch an agent to work on a Forge issue. Assigns the issue as a lock, prepends the issue body to TODO.md, creates an issue-specific branch, and spawns the agent.",
	}, s.dispatchIssue)

	// agentic_issue_dispatch is the spec-aligned name for the same action.
	coremcp.AddToolRecorded(svc, server, "agentic", &mcp.Tool{
		Name:        "agentic_issue_dispatch",
		Description: "Dispatch an agent to work on a Forge issue. Spec-aligned alias for agentic_dispatch_issue.",
	}, s.dispatchIssue)

	coremcp.AddToolRecorded(svc, server, "agentic", &mcp.Tool{
		Name:        "agentic_pr",
		Description: "Create a pull request from an agent workspace. Pushes the branch and creates a Forge PR linked to the tracked issue, if any.",
	}, s.createPR)
}

func (s *PrepSubsystem) dispatchIssue(ctx context.Context, req *mcp.CallToolRequest, input IssueDispatchInput) (*mcp.CallToolResult, DispatchOutput, error) {
	if input.Repo == "" {
		return nil, DispatchOutput{}, core.E("dispatchIssue", "repo is required", nil)
	}
	if input.Issue == 0 {
		return nil, DispatchOutput{}, core.E("dispatchIssue", "issue is required", nil)
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

	issue, err := s.fetchIssue(ctx, input.Org, input.Repo, input.Issue)
	if err != nil {
		return nil, DispatchOutput{}, err
	}
	if issue.State != "open" {
		return nil, DispatchOutput{}, core.E("dispatchIssue", core.Sprintf("issue %d is %s, not open", input.Issue, issue.State), nil)
	}
	if issue.Assignee != nil && issue.Assignee.Login != "" {
		return nil, DispatchOutput{}, core.E("dispatchIssue", core.Sprintf("issue %d is already assigned to %s", input.Issue, issue.Assignee.Login), nil)
	}

	if !input.DryRun {
		if err := s.lockIssue(ctx, input.Org, input.Repo, input.Issue, input.Agent); err != nil {
			return nil, DispatchOutput{}, err
		}

		var dispatchErr error
		defer func() {
			if dispatchErr != nil {
				if err := s.unlockIssue(ctx, input.Org, input.Repo, input.Issue, issue.Labels); err != nil {
					core.Error("agentic: failed to unlock issue after dispatch error", "err", err)
				}
			}
		}()

		result, out, dispatchErr := s.dispatch(ctx, req, DispatchInput{
			Repo:     input.Repo,
			Org:      input.Org,
			Issue:    input.Issue,
			Task:     issue.Title,
			Agent:    input.Agent,
			Template: input.Template,
			DryRun:   input.DryRun,
		})
		if dispatchErr != nil {
			return nil, DispatchOutput{}, dispatchErr
		}
		return result, out, nil
	}

	return s.dispatch(ctx, req, DispatchInput{
		Repo:     input.Repo,
		Org:      input.Org,
		Issue:    input.Issue,
		Task:     issue.Title,
		Agent:    input.Agent,
		Template: input.Template,
		DryRun:   input.DryRun,
	})
}

func (s *PrepSubsystem) unlockIssue(ctx context.Context, org, repo string, issue int, labels []struct {
	Name string `json:"name"`
}) error {
	updateURL := core.Sprintf("%s/api/v1/repos/%s/%s/issues/%d", s.forgeURL, org, repo, issue)
	issueLabels := make([]string, 0, len(labels))
	for _, label := range labels {
		if label.Name == "in-progress" {
			continue
		}
		issueLabels = append(issueLabels, label.Name)
	}
	if issueLabels == nil {
		issueLabels = []string{}
	}
	r := core.JSONMarshal(map[string]any{
		"assignees": []string{},
		"labels":    issueLabels,
	})
	if !r.OK {
		return core.E("unlockIssue", "failed to encode issue unlock", nil)
	}
	payload := r.Value.([]byte)

	req, err := http.NewRequestWithContext(ctx, http.MethodPatch, updateURL, bytes.NewReader(payload))
	if err != nil {
		return core.E("unlockIssue", "failed to build unlock request", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "token "+s.forgeToken)

	resp, err := s.client.Do(req)
	if err != nil {
		return core.E("unlockIssue", "failed to update issue", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= http.StatusBadRequest {
		return core.E("unlockIssue", core.Sprintf("issue unlock returned %d", resp.StatusCode), nil)
	}

	return nil
}

func (s *PrepSubsystem) fetchIssue(ctx context.Context, org, repo string, issue int) (*forgeIssue, error) {
	url := core.Sprintf("%s/api/v1/repos/%s/%s/issues/%d", s.forgeURL, org, repo, issue)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, core.E("fetchIssue", "failed to build request", err)
	}
	req.Header.Set("Authorization", "token "+s.forgeToken)

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, core.E("fetchIssue", "failed to fetch issue", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, core.E("fetchIssue", core.Sprintf("issue %d not found in %s/%s", issue, org, repo), nil)
	}

	var out forgeIssue
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, core.E("fetchIssue", "failed to decode issue", err)
	}
	return &out, nil
}

func (s *PrepSubsystem) lockIssue(ctx context.Context, org, repo string, issue int, assignee string) error {
	updateURL := core.Sprintf("%s/api/v1/repos/%s/%s/issues/%d", s.forgeURL, org, repo, issue)
	r := core.JSONMarshal(map[string]any{
		"assignees": []string{assignee},
		"labels":    []string{"in-progress"},
	})
	if !r.OK {
		return core.E("lockIssue", "failed to encode issue update", nil)
	}
	payload := r.Value.([]byte)

	req, err := http.NewRequestWithContext(ctx, http.MethodPatch, updateURL, bytes.NewReader(payload))
	if err != nil {
		return core.E("lockIssue", "failed to build update request", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "token "+s.forgeToken)

	resp, err := s.client.Do(req)
	if err != nil {
		return core.E("lockIssue", "failed to update issue", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= http.StatusBadRequest {
		return core.E("lockIssue", core.Sprintf("issue update returned %d", resp.StatusCode), nil)
	}

	return nil
}
