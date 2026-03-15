// SPDX-License-Identifier: EUPL-1.2

// Package agentic provides MCP tools for agent orchestration.
// Ported from CorePHP's Mod\Agentic to run standalone in core-mcp.
package agentic

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// PrepSubsystem provides the agentic:prep-workspace MCP tool.
type PrepSubsystem struct {
	forgeURL   string
	forgeToken string
	brainURL   string
	brainKey   string
	specsPath  string
	codePath   string
	client     *http.Client
}

// NewPrep creates an agentic prep subsystem.
func NewPrep() *PrepSubsystem {
	home, _ := os.UserHomeDir()

	forgeToken := os.Getenv("FORGE_TOKEN")
	if forgeToken == "" {
		forgeToken = os.Getenv("GITEA_TOKEN")
	}

	brainKey := os.Getenv("CORE_BRAIN_KEY")
	if brainKey == "" {
		if data, err := os.ReadFile(filepath.Join(home, ".claude", "brain.key")); err == nil {
			brainKey = strings.TrimSpace(string(data))
		}
	}

	return &PrepSubsystem{
		forgeURL:   envOr("FORGE_URL", "https://forge.lthn.ai"),
		forgeToken: forgeToken,
		brainURL:   envOr("CORE_BRAIN_URL", "https://api.lthn.sh"),
		brainKey:   brainKey,
		specsPath:  envOr("SPECS_PATH", filepath.Join(home, "Code", "host-uk", "specs")),
		codePath:   envOr("CODE_PATH", filepath.Join(home, "Code")),
		client:     &http.Client{Timeout: 30 * time.Second},
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// Name implements mcp.Subsystem.
func (s *PrepSubsystem) Name() string { return "agentic" }

// RegisterTools implements mcp.Subsystem.
func (s *PrepSubsystem) RegisterTools(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "agentic_prep_workspace",
		Description: "Prepare an agent workspace with CLAUDE.md, wiki KB, specs, OpenBrain context, consumer list, and recent git log for a target repo. Output goes to the repo's .core/ directory.",
	}, s.prepWorkspace)

	s.registerDispatchTool(server)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "agentic_scan",
		Description: "Scan Forge repos for open issues with actionable labels (agentic, help-wanted, bug). Returns a list of issues that can be dispatched to subagents.",
	}, s.scan)
}

// Shutdown implements mcp.SubsystemWithShutdown.
func (s *PrepSubsystem) Shutdown(_ context.Context) error { return nil }

// --- Input/Output types ---

// PrepInput is the input for agentic_prep_workspace.
type PrepInput struct {
	Repo    string `json:"repo"`              // e.g. "go-io"
	Org     string `json:"org,omitempty"`      // default "core"
	Issue   int    `json:"issue,omitempty"`    // Forge issue number
	Output  string `json:"output,omitempty"`   // override output dir
}

// PrepOutput is the output for agentic_prep_workspace.
type PrepOutput struct {
	Success    bool   `json:"success"`
	OutputDir  string `json:"output_dir"`
	WikiPages  int    `json:"wiki_pages"`
	SpecFiles  int    `json:"spec_files"`
	Memories   int    `json:"memories"`
	Consumers  int    `json:"consumers"`
	ClaudeMd   bool   `json:"claude_md"`
	GitLog     int    `json:"git_log_entries"`
}

func (s *PrepSubsystem) prepWorkspace(ctx context.Context, _ *mcp.CallToolRequest, input PrepInput) (*mcp.CallToolResult, PrepOutput, error) {
	if input.Repo == "" {
		return nil, PrepOutput{}, fmt.Errorf("repo is required")
	}
	if input.Org == "" {
		input.Org = "core"
	}

	// Determine output directory
	repoPath := filepath.Join(s.codePath, "core", input.Repo)
	outputDir := filepath.Join(repoPath, ".core")
	if input.Output != "" {
		outputDir = input.Output
	}

	// Create directories
	os.MkdirAll(filepath.Join(outputDir, "kb"), 0755)
	os.MkdirAll(filepath.Join(outputDir, "specs"), 0755)

	out := PrepOutput{OutputDir: outputDir}

	// 1. Copy CLAUDE.md from target repo
	claudeMdPath := filepath.Join(repoPath, "CLAUDE.md")
	if data, err := os.ReadFile(claudeMdPath); err == nil {
		os.WriteFile(filepath.Join(outputDir, "CLAUDE.md"), data, 0644)
		out.ClaudeMd = true
	}

	// 2. Pull wiki pages from Forge
	out.WikiPages = s.pullWiki(ctx, input.Org, input.Repo, outputDir)

	// 3. Copy spec files
	out.SpecFiles = s.copySpecs(outputDir)

	// 4. Generate context from OpenBrain
	out.Memories = s.generateContext(ctx, input.Repo, outputDir)

	// 5. Find consumers (who imports this module)
	out.Consumers = s.findConsumers(input.Repo, outputDir)

	// 6. Recent git log
	out.GitLog = s.gitLog(repoPath, outputDir)

	// 7. Generate TODO from issue (if provided)
	if input.Issue > 0 {
		s.generateTodo(ctx, input.Org, input.Repo, input.Issue, outputDir)
	}

	out.Success = true
	return nil, out, nil
}

func (s *PrepSubsystem) pullWiki(ctx context.Context, org, repo, outputDir string) int {
	if s.forgeToken == "" {
		return 0
	}

	url := fmt.Sprintf("%s/api/v1/repos/%s/%s/wiki/pages", s.forgeURL, org, repo)
	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	req.Header.Set("Authorization", "token "+s.forgeToken)

	resp, err := s.client.Do(req)
	if err != nil || resp.StatusCode != 200 {
		return 0
	}
	defer resp.Body.Close()

	var pages []struct {
		Title  string `json:"title"`
		SubURL string `json:"sub_url"`
	}
	json.NewDecoder(resp.Body).Decode(&pages)

	count := 0
	for _, page := range pages {
		subURL := page.SubURL
		if subURL == "" {
			subURL = page.Title
		}

		pageURL := fmt.Sprintf("%s/api/v1/repos/%s/%s/wiki/page/%s", s.forgeURL, org, repo, subURL)
		pageReq, _ := http.NewRequestWithContext(ctx, "GET", pageURL, nil)
		pageReq.Header.Set("Authorization", "token "+s.forgeToken)

		pageResp, err := s.client.Do(pageReq)
		if err != nil || pageResp.StatusCode != 200 {
			continue
		}

		var pageData struct {
			ContentBase64 string `json:"content_base64"`
		}
		json.NewDecoder(pageResp.Body).Decode(&pageData)
		pageResp.Body.Close()

		if pageData.ContentBase64 == "" {
			continue
		}

		content, _ := base64.StdEncoding.DecodeString(pageData.ContentBase64)
		filename := strings.Map(func(r rune) rune {
			if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_' || r == '.' {
				return r
			}
			return '-'
		}, page.Title) + ".md"

		os.WriteFile(filepath.Join(outputDir, "kb", filename), content, 0644)
		count++
	}

	return count
}

func (s *PrepSubsystem) copySpecs(outputDir string) int {
	specFiles := []string{"AGENT_CONTEXT.md", "TASK_PROTOCOL.md"}
	count := 0

	for _, file := range specFiles {
		src := filepath.Join(s.specsPath, file)
		if data, err := os.ReadFile(src); err == nil {
			os.WriteFile(filepath.Join(outputDir, "specs", file), data, 0644)
			count++
		}
	}

	return count
}

func (s *PrepSubsystem) generateContext(ctx context.Context, repo, outputDir string) int {
	if s.brainKey == "" {
		return 0
	}

	// Query OpenBrain for repo-specific knowledge
	body, _ := json.Marshal(map[string]any{
		"query":    "architecture conventions key interfaces for " + repo,
		"top_k":    10,
		"project":  repo,
		"agent_id": "cladius",
	})

	req, _ := http.NewRequestWithContext(ctx, "POST", s.brainURL+"/v1/brain/recall", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.brainKey)

	resp, err := s.client.Do(req)
	if err != nil || resp.StatusCode != 200 {
		return 0
	}
	defer resp.Body.Close()

	respData, _ := io.ReadAll(resp.Body)
	var result struct {
		Memories []map[string]any `json:"memories"`
	}
	json.Unmarshal(respData, &result)

	var content strings.Builder
	content.WriteString("# Agent Context — " + repo + "\n\n")
	content.WriteString("> Auto-generated by agentic_prep_workspace MCP tool.\n\n")

	for i, mem := range result.Memories {
		memType, _ := mem["type"].(string)
		memContent, _ := mem["content"].(string)
		memProject, _ := mem["project"].(string)
		score, _ := mem["score"].(float64)
		content.WriteString(fmt.Sprintf("### %d. %s [%s] (score: %.3f)\n\n%s\n\n", i+1, memProject, memType, score, memContent))
	}

	os.WriteFile(filepath.Join(outputDir, "context.md"), []byte(content.String()), 0644)
	return len(result.Memories)
}

func (s *PrepSubsystem) findConsumers(repo, outputDir string) int {
	goWorkPath := filepath.Join(s.codePath, "go.work")
	modulePath := "forge.lthn.ai/core/" + repo

	// Scan all go.mod files in the workspace for imports of this module
	workData, err := os.ReadFile(goWorkPath)
	if err != nil {
		return 0
	}

	var consumers []string
	for _, line := range strings.Split(string(workData), "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "./") {
			continue
		}
		dir := filepath.Join(s.codePath, strings.TrimPrefix(line, "./"))
		goMod := filepath.Join(dir, "go.mod")
		modData, err := os.ReadFile(goMod)
		if err != nil {
			continue
		}
		if strings.Contains(string(modData), modulePath) && !strings.HasPrefix(string(modData), "module "+modulePath) {
			consumers = append(consumers, filepath.Base(dir))
		}
	}

	if len(consumers) > 0 {
		content := "# Consumers of " + repo + "\n\n"
		content += "These modules import `" + modulePath + "`:\n\n"
		for _, c := range consumers {
			content += "- " + c + "\n"
		}
		content += fmt.Sprintf("\n**Breaking change risk: %d consumers.**\n", len(consumers))
		os.WriteFile(filepath.Join(outputDir, "consumers.md"), []byte(content), 0644)
	}

	return len(consumers)
}

func (s *PrepSubsystem) gitLog(repoPath, outputDir string) int {
	cmd := exec.Command("git", "log", "--oneline", "-20")
	cmd.Dir = repoPath
	output, err := cmd.Output()
	if err != nil {
		return 0
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) > 0 && lines[0] != "" {
		content := "# Recent Changes\n\n```\n" + string(output) + "```\n"
		os.WriteFile(filepath.Join(outputDir, "recent.md"), []byte(content), 0644)
	}

	return len(lines)
}

func (s *PrepSubsystem) generateTodo(ctx context.Context, org, repo string, issue int, outputDir string) {
	if s.forgeToken == "" {
		return
	}

	url := fmt.Sprintf("%s/api/v1/repos/%s/%s/issues/%d", s.forgeURL, org, repo, issue)
	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	req.Header.Set("Authorization", "token "+s.forgeToken)

	resp, err := s.client.Do(req)
	if err != nil || resp.StatusCode != 200 {
		return
	}
	defer resp.Body.Close()

	var issueData struct {
		Title string `json:"title"`
		Body  string `json:"body"`
	}
	json.NewDecoder(resp.Body).Decode(&issueData)

	content := fmt.Sprintf("# TASK: %s\n\n", issueData.Title)
	content += fmt.Sprintf("**Status:** ready\n")
	content += fmt.Sprintf("**Source:** %s/%s/%s/issues/%d\n", s.forgeURL, org, repo, issue)
	content += fmt.Sprintf("**Repo:** %s/%s\n\n---\n\n", org, repo)
	content += "## Objective\n\n" + issueData.Body + "\n"

	os.WriteFile(filepath.Join(outputDir, "todo.md"), []byte(content), 0644)
}
