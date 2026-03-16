// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	coreerr "forge.lthn.ai/core/go-log"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// --- agentic_create_pr ---

// CreatePRInput is the input for agentic_create_pr.
type CreatePRInput struct {
	Workspace string `json:"workspace"`            // workspace name (e.g. "mcp-1773581873")
	Title     string `json:"title,omitempty"`       // PR title (default: task description)
	Body      string `json:"body,omitempty"`        // PR body (default: auto-generated)
	Base      string `json:"base,omitempty"`        // base branch (default: "main")
	DryRun    bool   `json:"dry_run,omitempty"`     // preview without creating
}

// CreatePROutput is the output for agentic_create_pr.
type CreatePROutput struct {
	Success bool   `json:"success"`
	PRURL   string `json:"pr_url,omitempty"`
	PRNum   int    `json:"pr_number,omitempty"`
	Title   string `json:"title"`
	Branch  string `json:"branch"`
	Repo    string `json:"repo"`
	Pushed  bool   `json:"pushed"`
}

func (s *PrepSubsystem) registerCreatePRTool(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "agentic_create_pr",
		Description: "Create a pull request from an agent workspace. Pushes the branch to Forge and opens a PR. Links to the source issue if one was tracked.",
	}, s.createPR)
}

func (s *PrepSubsystem) createPR(ctx context.Context, _ *mcp.CallToolRequest, input CreatePRInput) (*mcp.CallToolResult, CreatePROutput, error) {
	if input.Workspace == "" {
		return nil, CreatePROutput{}, coreerr.E("createPR", "workspace is required", nil)
	}
	if s.forgeToken == "" {
		return nil, CreatePROutput{}, coreerr.E("createPR", "no Forge token configured", nil)
	}

	home, _ := os.UserHomeDir()
	wsDir := filepath.Join(home, "Code", "host-uk", "core", ".core", "workspace", input.Workspace)
	srcDir := filepath.Join(wsDir, "src")

	if _, err := os.Stat(srcDir); err != nil {
		return nil, CreatePROutput{}, coreerr.E("createPR", "workspace not found: "+input.Workspace, nil)
	}

	// Read workspace status for repo, branch, issue context
	st, err := readStatus(wsDir)
	if err != nil {
		return nil, CreatePROutput{}, coreerr.E("createPR", "no status.json", err)
	}

	if st.Branch == "" {
		// Detect branch from git
		branchCmd := exec.CommandContext(ctx, "git", "rev-parse", "--abbrev-ref", "HEAD")
		branchCmd.Dir = srcDir
		out, err := branchCmd.Output()
		if err != nil {
			return nil, CreatePROutput{}, coreerr.E("createPR", "failed to detect branch", err)
		}
		st.Branch = strings.TrimSpace(string(out))
	}

	org := st.Org
	if org == "" {
		org = "core"
	}
	base := input.Base
	if base == "" {
		base = "main"
	}

	// Build PR title
	title := input.Title
	if title == "" {
		title = st.Task
	}
	if title == "" {
		title = fmt.Sprintf("Agent work on %s", st.Branch)
	}

	// Build PR body
	body := input.Body
	if body == "" {
		body = s.buildPRBody(st)
	}

	if input.DryRun {
		return nil, CreatePROutput{
			Success: true,
			Title:   title,
			Branch:  st.Branch,
			Repo:    st.Repo,
		}, nil
	}

	// Push branch to forge
	pushCmd := exec.CommandContext(ctx, "git", "push", "-u", "origin", st.Branch)
	pushCmd.Dir = srcDir
	pushOut, err := pushCmd.CombinedOutput()
	if err != nil {
		return nil, CreatePROutput{}, coreerr.E("createPR", "git push failed: "+string(pushOut), err)
	}

	// Create PR via Forge API
	prURL, prNum, err := s.forgeCreatePR(ctx, org, st.Repo, st.Branch, base, title, body)
	if err != nil {
		return nil, CreatePROutput{}, coreerr.E("createPR", "failed to create PR", err)
	}

	// Update status with PR URL
	st.PRURL = prURL
	writeStatus(wsDir, st)

	// Comment on issue if tracked
	if st.Issue > 0 {
		comment := fmt.Sprintf("Pull request created: %s", prURL)
		s.commentOnIssue(ctx, org, st.Repo, st.Issue, comment)
	}

	return nil, CreatePROutput{
		Success: true,
		PRURL:   prURL,
		PRNum:   prNum,
		Title:   title,
		Branch:  st.Branch,
		Repo:    st.Repo,
		Pushed:  true,
	}, nil
}

func (s *PrepSubsystem) buildPRBody(st *WorkspaceStatus) string {
	var b strings.Builder
	b.WriteString("## Summary\n\n")
	if st.Task != "" {
		b.WriteString(st.Task)
		b.WriteString("\n\n")
	}
	if st.Issue > 0 {
		b.WriteString(fmt.Sprintf("Closes #%d\n\n", st.Issue))
	}
	b.WriteString(fmt.Sprintf("**Agent:** %s\n", st.Agent))
	b.WriteString(fmt.Sprintf("**Runs:** %d\n", st.Runs))
	b.WriteString("\n---\n*Created by agentic dispatch*\n")
	return b.String()
}

func (s *PrepSubsystem) forgeCreatePR(ctx context.Context, org, repo, head, base, title, body string) (string, int, error) {
	payload, _ := json.Marshal(map[string]any{
		"title": title,
		"body":  body,
		"head":  head,
		"base":  base,
	})

	url := fmt.Sprintf("%s/api/v1/repos/%s/%s/pulls", s.forgeURL, org, repo)
	req, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "token "+s.forgeToken)

	resp, err := s.client.Do(req)
	if err != nil {
		return "", 0, coreerr.E("forgeCreatePR", "request failed", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 201 {
		var errBody map[string]any
		json.NewDecoder(resp.Body).Decode(&errBody)
		msg, _ := errBody["message"].(string)
		return "", 0, coreerr.E("forgeCreatePR", fmt.Sprintf("HTTP %d: %s", resp.StatusCode, msg), nil)
	}

	var pr struct {
		Number  int    `json:"number"`
		HTMLURL string `json:"html_url"`
	}
	json.NewDecoder(resp.Body).Decode(&pr)

	return pr.HTMLURL, pr.Number, nil
}

func (s *PrepSubsystem) commentOnIssue(ctx context.Context, org, repo string, issue int, comment string) {
	payload, _ := json.Marshal(map[string]string{"body": comment})

	url := fmt.Sprintf("%s/api/v1/repos/%s/%s/issues/%d/comments", s.forgeURL, org, repo, issue)
	req, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "token "+s.forgeToken)

	resp, err := s.client.Do(req)
	if err != nil {
		return
	}
	resp.Body.Close()
}

// --- agentic_list_prs ---

// ListPRsInput is the input for agentic_list_prs.
type ListPRsInput struct {
	Org   string `json:"org,omitempty"`   // forge org (default "core")
	Repo  string `json:"repo,omitempty"` // specific repo, or empty for all
	State string `json:"state,omitempty"` // "open" (default), "closed", "all"
	Limit int    `json:"limit,omitempty"` // max results (default 20)
}

// ListPRsOutput is the output for agentic_list_prs.
type ListPRsOutput struct {
	Success bool     `json:"success"`
	Count   int      `json:"count"`
	PRs     []PRInfo `json:"prs"`
}

// PRInfo represents a pull request.
type PRInfo struct {
	Repo      string   `json:"repo"`
	Number    int      `json:"number"`
	Title     string   `json:"title"`
	State     string   `json:"state"`
	Author    string   `json:"author"`
	Branch    string   `json:"branch"`
	Base      string   `json:"base"`
	Labels    []string `json:"labels,omitempty"`
	Mergeable bool     `json:"mergeable"`
	URL       string   `json:"url"`
}

func (s *PrepSubsystem) registerListPRsTool(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "agentic_list_prs",
		Description: "List pull requests across Forge repos. Filter by org, repo, and state (open/closed/all).",
	}, s.listPRs)
}

func (s *PrepSubsystem) listPRs(ctx context.Context, _ *mcp.CallToolRequest, input ListPRsInput) (*mcp.CallToolResult, ListPRsOutput, error) {
	if s.forgeToken == "" {
		return nil, ListPRsOutput{}, coreerr.E("listPRs", "no Forge token configured", nil)
	}

	if input.Org == "" {
		input.Org = "core"
	}
	if input.State == "" {
		input.State = "open"
	}
	if input.Limit == 0 {
		input.Limit = 20
	}

	var repos []string
	if input.Repo != "" {
		repos = []string{input.Repo}
	} else {
		var err error
		repos, err = s.listOrgRepos(ctx, input.Org)
		if err != nil {
			return nil, ListPRsOutput{}, err
		}
	}

	var allPRs []PRInfo

	for _, repo := range repos {
		prs, err := s.listRepoPRs(ctx, input.Org, repo, input.State)
		if err != nil {
			continue
		}
		allPRs = append(allPRs, prs...)

		if len(allPRs) >= input.Limit {
			break
		}
	}

	if len(allPRs) > input.Limit {
		allPRs = allPRs[:input.Limit]
	}

	return nil, ListPRsOutput{
		Success: true,
		Count:   len(allPRs),
		PRs:     allPRs,
	}, nil
}

func (s *PrepSubsystem) listRepoPRs(ctx context.Context, org, repo, state string) ([]PRInfo, error) {
	url := fmt.Sprintf("%s/api/v1/repos/%s/%s/pulls?state=%s&limit=10",
		s.forgeURL, org, repo, state)
	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	req.Header.Set("Authorization", "token "+s.forgeToken)

	resp, err := s.client.Do(req)
	if err != nil || resp.StatusCode != 200 {
		return nil, coreerr.E("listRepoPRs", "failed to list PRs for "+repo, err)
	}
	defer resp.Body.Close()

	var prs []struct {
		Number    int    `json:"number"`
		Title     string `json:"title"`
		State     string `json:"state"`
		Mergeable bool   `json:"mergeable"`
		HTMLURL   string `json:"html_url"`
		Head      struct {
			Ref string `json:"ref"`
		} `json:"head"`
		Base struct {
			Ref string `json:"ref"`
		} `json:"base"`
		User struct {
			Login string `json:"login"`
		} `json:"user"`
		Labels []struct {
			Name string `json:"name"`
		} `json:"labels"`
	}
	json.NewDecoder(resp.Body).Decode(&prs)

	var result []PRInfo
	for _, pr := range prs {
		var labels []string
		for _, l := range pr.Labels {
			labels = append(labels, l.Name)
		}
		result = append(result, PRInfo{
			Repo:      repo,
			Number:    pr.Number,
			Title:     pr.Title,
			State:     pr.State,
			Author:    pr.User.Login,
			Branch:    pr.Head.Ref,
			Base:      pr.Base.Ref,
			Labels:    labels,
			Mergeable: pr.Mergeable,
			URL:       pr.HTMLURL,
		})
	}

	return result, nil
}
