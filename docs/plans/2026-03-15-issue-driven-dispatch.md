# Issue-Driven Dispatch Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Agents claim Forge issues, work in sandboxed workspaces with feature branches, and create PRs when done. Assignment = lock (no two agents work the same issue).

**Architecture:** New MCP tool `agentic_dispatch_issue` takes an issue number + repo, assigns the issue to the agent (lock), preps a workspace with the issue body as TODO.md, creates a feature branch `agent/issue-{num}-{slug}`, and dispatches. On completion, a `agentic_pr` tool creates a PR via the Forge API linking back to the issue.

**Tech Stack:** Go, MCP SDK, Forge/Gitea API (go-scm), git

---

## Existing Infrastructure

Already built:
- `agentic_dispatch` — preps workspace + spawns agent (dispatch.go)
- `agentic_scan` — finds issues with actionable labels (scan.go)
- `prepWorkspace()` — clones repo, creates feature branch, writes context files
- `generateTodo()` — fetches issue from Forge API, writes TODO.md
- PrepSubsystem has `forgeURL`, `forgeToken`, `client` fields

The issue flow just wires scan → dispatch together with assignment as the lock.

## File Structure

| File | Action | Purpose |
|------|--------|---------|
| `pkg/mcp/agentic/issue.go` | Create | `agentic_dispatch_issue` + `agentic_pr` tools |
| `pkg/mcp/agentic/prep.go` | Modify | Register new tools |

---

## Task 1: Issue Dispatch Tool

**Files:**
- Create: `pkg/mcp/agentic/issue.go`
- Modify: `pkg/mcp/agentic/prep.go`

- [ ] **Step 1: Create issue.go with input/output types**

```go
// issue.go
package agentic

// IssueDispatchInput for agentic_dispatch_issue
type IssueDispatchInput struct {
    Repo     string `json:"repo"`              // Target repo (e.g. "go-io")
    Org      string `json:"org,omitempty"`      // Forge org (default "core")
    Issue    int    `json:"issue"`             // Forge issue number
    Agent    string `json:"agent,omitempty"`   // "gemini", "codex", "claude" (default "claude")
    Template string `json:"template,omitempty"` // "conventions", "security", "coding" (default "coding")
    DryRun   bool   `json:"dry_run,omitempty"`
}

// PRInput for agentic_pr
type PRInput struct {
    Workspace string `json:"workspace"`           // Workspace name
    Title     string `json:"title,omitempty"`      // PR title (default: from issue)
    Body      string `json:"body,omitempty"`       // PR body (default: auto-generated)
    Base      string `json:"base,omitempty"`       // Base branch (default: "main")
}
```

- [ ] **Step 2: Implement agentic_dispatch_issue**

Flow:
1. Fetch issue from Forge API to validate it exists
2. Check issue is not already assigned (if assigned, return error — it's locked)
3. Assign issue to agent (POST /api/v1/repos/{org}/{repo}/issues/{num} with assignee)
4. Add "in-progress" label
5. Call existing `dispatch()` with the issue number (it already handles TODO.md generation)

```go
func (s *PrepSubsystem) dispatchIssue(ctx context.Context, req *mcp.CallToolRequest, input IssueDispatchInput) (*mcp.CallToolResult, DispatchOutput, error) {
    if input.Issue == 0 {
        return nil, DispatchOutput{}, fmt.Errorf("issue number is required")
    }
    if input.Repo == "" {
        return nil, DispatchOutput{}, fmt.Errorf("repo is required")
    }
    if input.Org == "" {
        input.Org = "core"
    }
    if input.Agent == "" {
        input.Agent = "claude"
    }

    // 1. Fetch issue to validate and check assignment
    issueURL := fmt.Sprintf("%s/api/v1/repos/%s/%s/issues/%d", s.forgeURL, input.Org, input.Repo, input.Issue)
    issueReq, _ := http.NewRequestWithContext(ctx, "GET", issueURL, nil)
    issueReq.Header.Set("Authorization", "token "+s.forgeToken)

    resp, err := s.client.Do(issueReq)
    if err != nil || resp.StatusCode != 200 {
        return nil, DispatchOutput{}, fmt.Errorf("issue %d not found in %s/%s", input.Issue, input.Org, input.Repo)
    }
    defer resp.Body.Close()

    var issue struct {
        Title    string `json:"title"`
        Assignee *struct {
            Login string `json:"login"`
        } `json:"assignee"`
        State string `json:"state"`
    }
    json.NewDecoder(resp.Body).Decode(&issue)

    if issue.State != "open" {
        return nil, DispatchOutput{}, fmt.Errorf("issue %d is %s, not open", input.Issue, issue.State)
    }

    // 2. Check lock (assignment)
    if issue.Assignee != nil && issue.Assignee.Login != "" {
        return nil, DispatchOutput{}, fmt.Errorf("issue %d already assigned to %s", input.Issue, issue.Assignee.Login)
    }

    // 3. Assign to agent (lock)
    if !input.DryRun && s.forgeToken != "" {
        assignBody, _ := json.Marshal(map[string]any{"assignees": []string{input.Agent}})
        assignReq, _ := http.NewRequestWithContext(ctx, "PATCH", issueURL, bytes.NewReader(assignBody))
        assignReq.Header.Set("Authorization", "token "+s.forgeToken)
        assignReq.Header.Set("Content-Type", "application/json")
        s.client.Do(assignReq)
    }

    // 4. Dispatch via existing dispatch()
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
```

- [ ] **Step 3: Implement agentic_pr**

Creates a PR from the agent's feature branch back to main, referencing the issue.

```go
func (s *PrepSubsystem) createPR(ctx context.Context, _ *mcp.CallToolRequest, input PRInput) (*mcp.CallToolResult, map[string]any, error) {
    if input.Workspace == "" {
        return nil, nil, fmt.Errorf("workspace is required")
    }

    home, _ := os.UserHomeDir()
    wsDir := filepath.Join(home, "Code", "host-uk", "core", ".core", "workspace", input.Workspace)
    srcDir := filepath.Join(wsDir, "src")

    // Read status to get repo info
    st, err := readStatus(wsDir)
    if err != nil {
        return nil, nil, fmt.Errorf("no status.json: %w", err)
    }

    // Get current branch name
    branchCmd := exec.CommandContext(ctx, "git", "branch", "--show-current")
    branchCmd.Dir = srcDir
    branchOut, err := branchCmd.Output()
    if err != nil {
        return nil, nil, fmt.Errorf("failed to get branch: %w", err)
    }
    branch := strings.TrimSpace(string(branchOut))

    // Push branch to forge
    pushCmd := exec.CommandContext(ctx, "git", "push", "origin", branch)
    pushCmd.Dir = srcDir
    if out, err := pushCmd.CombinedOutput(); err != nil {
        return nil, nil, fmt.Errorf("push failed: %s", string(out))
    }

    // Determine PR title and body
    title := input.Title
    if title == "" {
        title = st.Task
    }
    base := input.Base
    if base == "" {
        base = "main"
    }
    body := input.Body
    if body == "" {
        body = fmt.Sprintf("Automated PR from agent workspace `%s`.\n\nTask: %s", input.Workspace, st.Task)
    }

    // Create PR via Forge API
    org := "core" // TODO: extract from status
    prURL := fmt.Sprintf("%s/api/v1/repos/%s/%s/pulls", s.forgeURL, org, st.Repo)
    prBody, _ := json.Marshal(map[string]any{
        "title": title,
        "body":  body,
        "head":  branch,
        "base":  base,
    })

    prReq, _ := http.NewRequestWithContext(ctx, "POST", prURL, bytes.NewReader(prBody))
    prReq.Header.Set("Authorization", "token "+s.forgeToken)
    prReq.Header.Set("Content-Type", "application/json")

    resp, err := s.client.Do(prReq)
    if err != nil {
        return nil, nil, fmt.Errorf("PR creation failed: %w", err)
    }
    defer resp.Body.Close()

    var prResult map[string]any
    json.NewDecoder(resp.Body).Decode(&prResult)

    if resp.StatusCode >= 400 {
        return nil, nil, fmt.Errorf("PR creation returned %d", resp.StatusCode)
    }

    return nil, prResult, nil
}
```

- [ ] **Step 4: Register tools in prep.go**

Add to `RegisterTools()`:

```go
s.registerIssueTools(server)
```

Registration:

```go
func (s *PrepSubsystem) registerIssueTools(server *mcp.Server) {
    mcp.AddTool(server, &mcp.Tool{
        Name:        "agentic_dispatch_issue",
        Description: "Dispatch an agent to work on a Forge issue. Assigns the issue (lock), preps workspace with issue body as TODO.md, creates feature branch, spawns agent.",
    }, s.dispatchIssue)

    mcp.AddTool(server, &mcp.Tool{
        Name:        "agentic_pr",
        Description: "Create a PR from an agent workspace. Pushes the feature branch and creates a pull request on Forge linking to the original issue.",
    }, s.createPR)
}
```

- [ ] **Step 5: Verify compilation**

Run: `go vet ./pkg/mcp/agentic/`
Expected: clean

- [ ] **Step 6: Commit**

```bash
git add pkg/mcp/agentic/issue.go pkg/mcp/agentic/prep.go
git commit -m "feat(agentic): issue-driven dispatch — claim, branch, PR

New MCP tools:
- agentic_dispatch_issue: assigns issue (lock), preps workspace, dispatches
- agentic_pr: pushes branch, creates PR via Forge API

Assignment = lock — no two agents work the same issue.

Co-Authored-By: Virgil <virgil@lethean.io>"
```

---

## Summary

**Total: 1 task, 6 steps**

After completion, the full issue lifecycle:
1. `agentic_scan` — find issues with actionable labels
2. `agentic_dispatch_issue` — claim issue (assign = lock), prep workspace, spawn agent
3. Agent works in sandboxed workspace with feature branch
4. Agent writes BLOCKED.md if stuck → `agentic_resume` to continue
5. `agentic_pr` — push branch, create PR referencing the issue
6. PR reviewed and merged

Community flow:
- Maintainer creates issue with `agentic` label
- Agent scans, claims, works, PRs
- Maintainer reviews and merges
