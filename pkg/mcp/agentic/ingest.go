// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"net/http"

	core "dappco.re/go/core"
	coreio "dappco.re/go/io"
	coremcp "dappco.re/go/mcp/pkg/mcp"
)

// ingestFindings reads the agent output log and creates issues via the API
// for scan/audit results. Only runs for conventions and security templates.
func (s *PrepSubsystem) ingestFindings(wsDir string) {
	st, err := readStatus(wsDir)
	if err != nil || st.Status != "completed" {
		return
	}

	// Read the log file
	logFiles := core.PathGlob(core.Path(wsDir, "agent-*.log"))
	if len(logFiles) == 0 {
		return
	}

	contentStr, err := coreio.Local.Read(logFiles[0])
	if err != nil || len(contentStr) < 100 {
		return
	}

	body := contentStr

	// Skip quota errors
	if core.Contains(body, "QUOTA_EXHAUSTED") || core.Contains(body, "QuotaError") {
		return
	}

	// Only ingest if there are actual findings (file:line references)
	findings := countFileRefs(body)
	issueCreated := false
	if findings < 2 {
		s.emitHarvestComplete(context.Background(), wsDir, st.Repo, findings, issueCreated)
		return // No meaningful findings
	}

	// Determine issue type from the template used
	issueType := "task"
	priority := "normal"
	if core.Contains(body, "security") || core.Contains(body, "Security") {
		issueType = "bug"
		priority = "high"
	}

	// Create a single issue per repo with all findings in the body
	title := core.Sprintf("Scan findings for %s (%d items)", st.Repo, findings)

	// Truncate body to reasonable size for issue description
	description := body
	if len(description) > 10000 {
		description = description[:10000] + "\n\n... (truncated, see full log in workspace)"
	}

	issueCreated = s.createIssueViaAPI(st.Repo, title, description, issueType, priority, "scan")
	s.emitHarvestComplete(context.Background(), wsDir, st.Repo, findings, issueCreated)
}

// countFileRefs counts file:line references in the output (indicates real findings)
func countFileRefs(body string) int {
	count := 0
	for i := 0; i < len(body)-5; i++ {
		if body[i] == '`' {
			// Look for pattern: `file.go:123`
			j := i + 1
			for j < len(body) && body[j] != '`' && j-i < 100 {
				j++
			}
			if j < len(body) && body[j] == '`' {
				ref := body[i+1 : j]
				if core.Contains(ref, ".go:") || core.Contains(ref, ".php:") {
					count++
				}
			}
		}
	}
	return count
}

// createIssueViaAPI posts an issue to the lthn.sh API
func (s *PrepSubsystem) createIssueViaAPI(repo, title, description, issueType, priority, source string) bool {
	if s.brainKey == "" {
		return false
	}

	// Read the agent API key from file
	home := core.Env("HOME")
	apiKeyData, err := coreio.Local.Read(core.Path(home, ".claude", "agent-api.key"))
	if err != nil {
		return false
	}
	apiKey := core.Trim(apiKeyData)

	payloadStr := core.JSONMarshalString(map[string]string{
		"title":       title,
		"description": description,
		"type":        issueType,
		"priority":    priority,
		"reporter":    "cladius",
	})

	req, err := http.NewRequest("POST", s.brainURL+"/v1/issues", core.NewReader(payloadStr))
	if err != nil {
		return false
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := s.client.Do(req)
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode < 400
}

// emitHarvestComplete announces that finding ingestion finished for a workspace.
//
//	ctx := context.Background()
//	s.emitHarvestComplete(ctx, "go-io-123", "go-io", 4, true)
func (s *PrepSubsystem) emitHarvestComplete(ctx context.Context, workspace, repo string, findings int, issueCreated bool) {
	s.emitChannel(ctx, coremcp.ChannelHarvestComplete, map[string]any{
		"workspace":     workspace,
		"repo":          repo,
		"findings":      findings,
		"issue_created": issueCreated,
	})
}
