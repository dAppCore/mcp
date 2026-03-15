// SPDX-License-Identifier: EUPL-1.2

// Package agentic provides MCP tools for agent orchestration.
// Prepares sandboxed workspaces and dispatches subagents.
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
	"gopkg.in/yaml.v3"
)

// PrepSubsystem provides agentic MCP tools.
type PrepSubsystem struct {
	forgeURL    string
	forgeToken  string
	brainURL    string
	brainKey    string
	specsPath   string
	codePath    string
	client *http.Client
}

// NewPrep creates an agentic subsystem.
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
		forgeURL:    envOr("FORGE_URL", "https://forge.lthn.ai"),
		forgeToken:  forgeToken,
		brainURL:    envOr("CORE_BRAIN_URL", "https://api.lthn.sh"),
		brainKey:    brainKey,
		specsPath:   envOr("SPECS_PATH", filepath.Join(home, "Code", "host-uk", "specs")),
		codePath:    envOr("CODE_PATH", filepath.Join(home, "Code")),
		client: &http.Client{Timeout: 30 * time.Second},
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
		Description: "Prepare a sandboxed agent workspace with TODO.md, CLAUDE.md, CONTEXT.md, CONSUMERS.md, RECENT.md, and a git clone of the target repo in src/.",
	}, s.prepWorkspace)

	s.registerDispatchTool(server)
	s.registerStatusTool(server)
	s.registerResumeTool(server)
	s.registerCreatePRTool(server)
	s.registerListPRsTool(server)
	s.registerEpicTool(server)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "agentic_scan",
		Description: "Scan Forge repos for open issues with actionable labels (agentic, help-wanted, bug).",
	}, s.scan)

	s.registerPlanTools(server)
}

// Shutdown implements mcp.SubsystemWithShutdown.
func (s *PrepSubsystem) Shutdown(_ context.Context) error { return nil }

// --- Input/Output types ---

// PrepInput is the input for agentic_prep_workspace.
type PrepInput struct {
	Repo         string            `json:"repo"`                    // e.g. "go-io"
	Org          string            `json:"org,omitempty"`           // default "core"
	Issue        int               `json:"issue,omitempty"`         // Forge issue number
	Task         string            `json:"task,omitempty"`          // Task description (if no issue)
	Template     string            `json:"template,omitempty"`      // Prompt template: conventions, security, coding (default: coding)
	PlanTemplate string            `json:"plan_template,omitempty"` // Plan template slug: bug-fix, code-review, new-feature, refactor, feature-port
	Variables    map[string]string `json:"variables,omitempty"`     // Template variable substitution
}

// PrepOutput is the output for agentic_prep_workspace.
type PrepOutput struct {
	Success       bool   `json:"success"`
	WorkspaceDir  string `json:"workspace_dir"`
	WikiPages     int    `json:"wiki_pages"`
	SpecFiles     int    `json:"spec_files"`
	Memories      int    `json:"memories"`
	Consumers     int    `json:"consumers"`
	ClaudeMd      bool   `json:"claude_md"`
	GitLog        int    `json:"git_log_entries"`
}

func (s *PrepSubsystem) prepWorkspace(ctx context.Context, _ *mcp.CallToolRequest, input PrepInput) (*mcp.CallToolResult, PrepOutput, error) {
	if input.Repo == "" {
		return nil, PrepOutput{}, fmt.Errorf("repo is required")
	}
	if input.Org == "" {
		input.Org = "core"
	}
	if input.Template == "" {
		input.Template = "coding"
	}

	// Workspace root: .core/workspace/{repo}-{timestamp}/
	home, _ := os.UserHomeDir()
	wsRoot := filepath.Join(home, "Code", "host-uk", "core", ".core", "workspace")
	wsName := fmt.Sprintf("%s-%d", input.Repo, time.Now().Unix())
	wsDir := filepath.Join(wsRoot, wsName)

	// Create workspace structure
	// kb/ and specs/ will be created inside src/ after clone

	out := PrepOutput{WorkspaceDir: wsDir}

	// Source repo path
	repoPath := filepath.Join(s.codePath, "core", input.Repo)

	// 1. Clone repo into src/ and create feature branch
	srcDir := filepath.Join(wsDir, "src")
	cloneCmd := exec.CommandContext(ctx, "git", "clone", repoPath, srcDir)
	cloneCmd.Run()

	// Create feature branch
	taskSlug := strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '-' {
			return r
		}
		if r >= 'A' && r <= 'Z' {
			return r + 32 // lowercase
		}
		return '-'
	}, input.Task)
	if len(taskSlug) > 40 {
		taskSlug = taskSlug[:40]
	}
	taskSlug = strings.Trim(taskSlug, "-")
	branchName := fmt.Sprintf("agent/%s", taskSlug)

	branchCmd := exec.CommandContext(ctx, "git", "checkout", "-b", branchName)
	branchCmd.Dir = srcDir
	branchCmd.Run()

	// Create context dirs inside src/
	os.MkdirAll(filepath.Join(srcDir, "kb"), 0755)
	os.MkdirAll(filepath.Join(srcDir, "specs"), 0755)

	// Remote stays as local clone origin — agent cannot push to forge.
	// Reviewer pulls changes from workspace and pushes after verification.

	// 2. Copy CLAUDE.md and GEMINI.md to workspace
	claudeMdPath := filepath.Join(repoPath, "CLAUDE.md")
	if data, err := os.ReadFile(claudeMdPath); err == nil {
		os.WriteFile(filepath.Join(wsDir, "src", "CLAUDE.md"), data, 0644)
		out.ClaudeMd = true
	}
	// Copy GEMINI.md from core/agent (ethics framework for all agents)
	agentGeminiMd := filepath.Join(s.codePath, "core", "agent", "GEMINI.md")
	if data, err := os.ReadFile(agentGeminiMd); err == nil {
		os.WriteFile(filepath.Join(wsDir, "src", "GEMINI.md"), data, 0644)
	}

	// 3. Generate TODO.md
	if input.Issue > 0 {
		s.generateTodo(ctx, input.Org, input.Repo, input.Issue, wsDir)
	} else if input.Task != "" {
		todo := fmt.Sprintf("# TASK: %s\n\n**Repo:** %s/%s\n**Status:** ready\n\n## Objective\n\n%s\n",
			input.Task, input.Org, input.Repo, input.Task)
		os.WriteFile(filepath.Join(wsDir, "src", "TODO.md"), []byte(todo), 0644)
	}

	// 4. Generate CONTEXT.md from OpenBrain
	out.Memories = s.generateContext(ctx, input.Repo, wsDir)

	// 5. Generate CONSUMERS.md
	out.Consumers = s.findConsumers(input.Repo, wsDir)

	// 6. Generate RECENT.md
	out.GitLog = s.gitLog(repoPath, wsDir)

	// 7. Pull wiki pages into kb/
	out.WikiPages = s.pullWiki(ctx, input.Org, input.Repo, wsDir)

	// 8. Copy spec files into specs/
	out.SpecFiles = s.copySpecs(wsDir)

	// 9. Write PLAN.md from template (if specified)
	if input.PlanTemplate != "" {
		s.writePlanFromTemplate(input.PlanTemplate, input.Variables, input.Task, wsDir)
	}

	// 10. Write prompt template
	s.writePromptTemplate(input.Template, wsDir)

	out.Success = true
	return nil, out, nil
}

// --- Prompt templates ---

func (s *PrepSubsystem) writePromptTemplate(template, wsDir string) {
	var prompt string

	switch template {
	case "conventions":
		prompt = `Read CLAUDE.md for project conventions.
Review all Go files in src/ for:
- Error handling: should use coreerr.E() from go-log, not fmt.Errorf or errors.New
- Compile-time interface checks: var _ Interface = (*Impl)(nil)
- Import aliasing: stdlib io aliased as goio
- UK English in comments (colour not color, initialise not initialize)
- No fmt.Print* debug statements (use go-log)
- Test coverage gaps

Report findings with file:line references. Do not fix — only report.
`
	case "security":
		prompt = `Read CLAUDE.md for project context.
Review all Go files in src/ for security issues:
- Path traversal vulnerabilities
- Unvalidated input
- SQL injection (if applicable)
- Hardcoded credentials or tokens
- Unsafe type assertions
- Missing error checks
- Race conditions (shared state without mutex)
- Unsafe use of os/exec

Report findings with severity (critical/high/medium/low) and file:line references.
`
	case "coding":
		prompt = `Read CLAUDE.md for project conventions and context.
Read TODO.md for your task.
Read PLAN.md if it exists — work through each phase in order.
Read CONTEXT.md for relevant knowledge from previous sessions.
Read CONSUMERS.md to understand breaking change risk.
Read RECENT.md for recent changes.

Work in the src/ directory. Follow the conventions in CLAUDE.md.

## Workflow

If PLAN.md exists, work through it phase by phase:
1. Complete all tasks in the current phase
2. Commit with message: type(scope): description
3. Move to the next phase
4. If you are blocked or unsure, write BLOCKED.md explaining the question and stop

If no PLAN.md, complete TODO.md as a single unit of work.

## Commit Convention

Commit message format: type(scope): description
Co-Author: Co-Authored-By: Virgil <virgil@lethean.io>

Do NOT push. Commit only — a reviewer will verify and push.
`
	default:
		prompt = "Read TODO.md and complete the task. Work in src/.\n"
	}

	os.WriteFile(filepath.Join(wsDir, "src", "PROMPT.md"), []byte(prompt), 0644)
}

// --- Plan template rendering ---

// writePlanFromTemplate loads a YAML plan template, substitutes variables,
// and writes PLAN.md into the workspace src/ directory.
func (s *PrepSubsystem) writePlanFromTemplate(templateSlug string, variables map[string]string, task string, wsDir string) {
	// Look for template in core/agent/prompts/templates/
	templatePath := filepath.Join(s.codePath, "core", "agent", "prompts", "templates", templateSlug+".yaml")
	data, err := os.ReadFile(templatePath)
	if err != nil {
		// Try .yml extension
		templatePath = filepath.Join(s.codePath, "core", "agent", "prompts", "templates", templateSlug+".yml")
		data, err = os.ReadFile(templatePath)
		if err != nil {
			return // Template not found, skip silently
		}
	}

	content := string(data)

	// Substitute variables ({{variable_name}} → value)
	for key, value := range variables {
		content = strings.ReplaceAll(content, "{{"+key+"}}", value)
		content = strings.ReplaceAll(content, "{{ "+key+" }}", value)
	}

	// Parse the YAML to render as markdown
	var tmpl struct {
		Name        string   `yaml:"name"`
		Description string   `yaml:"description"`
		Guidelines  []string `yaml:"guidelines"`
		Phases      []struct {
			Name        string   `yaml:"name"`
			Description string   `yaml:"description"`
			Tasks       []any    `yaml:"tasks"`
		} `yaml:"phases"`
	}

	if err := yaml.Unmarshal([]byte(content), &tmpl); err != nil {
		return
	}

	// Render as PLAN.md
	var plan strings.Builder
	plan.WriteString("# Plan: " + tmpl.Name + "\n\n")
	if task != "" {
		plan.WriteString("**Task:** " + task + "\n\n")
	}
	if tmpl.Description != "" {
		plan.WriteString(tmpl.Description + "\n\n")
	}

	if len(tmpl.Guidelines) > 0 {
		plan.WriteString("## Guidelines\n\n")
		for _, g := range tmpl.Guidelines {
			plan.WriteString("- " + g + "\n")
		}
		plan.WriteString("\n")
	}

	for i, phase := range tmpl.Phases {
		plan.WriteString(fmt.Sprintf("## Phase %d: %s\n\n", i+1, phase.Name))
		if phase.Description != "" {
			plan.WriteString(phase.Description + "\n\n")
		}
		for _, task := range phase.Tasks {
			switch t := task.(type) {
			case string:
				plan.WriteString("- [ ] " + t + "\n")
			case map[string]any:
				if name, ok := t["name"].(string); ok {
					plan.WriteString("- [ ] " + name + "\n")
				}
			}
		}
		plan.WriteString("\n**Commit after completing this phase.**\n\n---\n\n")
	}

	os.WriteFile(filepath.Join(wsDir, "src", "PLAN.md"), []byte(plan.String()), 0644)
}

// --- Helpers (unchanged) ---

func (s *PrepSubsystem) pullWiki(ctx context.Context, org, repo, wsDir string) int {
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

		os.WriteFile(filepath.Join(wsDir, "src", "kb", filename), content, 0644)
		count++
	}

	return count
}

func (s *PrepSubsystem) copySpecs(wsDir string) int {
	specFiles := []string{"AGENT_CONTEXT.md", "TASK_PROTOCOL.md"}
	count := 0

	for _, file := range specFiles {
		src := filepath.Join(s.specsPath, file)
		if data, err := os.ReadFile(src); err == nil {
			os.WriteFile(filepath.Join(wsDir, "src", "specs", file), data, 0644)
			count++
		}
	}

	return count
}

func (s *PrepSubsystem) generateContext(ctx context.Context, repo, wsDir string) int {
	if s.brainKey == "" {
		return 0
	}

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
	content.WriteString("# Context — " + repo + "\n\n")
	content.WriteString("> Relevant knowledge from OpenBrain.\n\n")

	for i, mem := range result.Memories {
		memType, _ := mem["type"].(string)
		memContent, _ := mem["content"].(string)
		memProject, _ := mem["project"].(string)
		score, _ := mem["score"].(float64)
		content.WriteString(fmt.Sprintf("### %d. %s [%s] (score: %.3f)\n\n%s\n\n", i+1, memProject, memType, score, memContent))
	}

	os.WriteFile(filepath.Join(wsDir, "src", "CONTEXT.md"), []byte(content.String()), 0644)
	return len(result.Memories)
}

func (s *PrepSubsystem) findConsumers(repo, wsDir string) int {
	goWorkPath := filepath.Join(s.codePath, "go.work")
	modulePath := "forge.lthn.ai/core/" + repo

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
		os.WriteFile(filepath.Join(wsDir, "src", "CONSUMERS.md"), []byte(content), 0644)
	}

	return len(consumers)
}

func (s *PrepSubsystem) gitLog(repoPath, wsDir string) int {
	cmd := exec.Command("git", "log", "--oneline", "-20")
	cmd.Dir = repoPath
	output, err := cmd.Output()
	if err != nil {
		return 0
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) > 0 && lines[0] != "" {
		content := "# Recent Changes\n\n```\n" + string(output) + "```\n"
		os.WriteFile(filepath.Join(wsDir, "src", "RECENT.md"), []byte(content), 0644)
	}

	return len(lines)
}

func (s *PrepSubsystem) generateTodo(ctx context.Context, org, repo string, issue int, wsDir string) {
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

	os.WriteFile(filepath.Join(wsDir, "src", "TODO.md"), []byte(content), 0644)
}
