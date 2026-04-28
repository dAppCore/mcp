// SPDX-License-Identifier: EUPL-1.2

// brain-seed imports Claude Code MEMORY.md files into the OpenBrain knowledge
// store via the shared OpenBrain HTTP client. The Laravel app handles
// embedding, Qdrant storage, and MariaDB dual-write internally.
//
// Usage:
//
//	go run ./cmd/brain-seed -api-key YOUR_KEY
//	go run ./cmd/brain-seed -api-key YOUR_KEY -api https://api.lthn.sh
//	go run ./cmd/brain-seed -api-key YOUR_KEY -dry-run
//	go run ./cmd/brain-seed -api-key YOUR_KEY -plans
//	go run ./cmd/brain-seed -api-key YOUR_KEY -claude-md  # Also import CLAUDE.md files
package main

import (
	"context"
	"flag"
	"os"
	"path/filepath"
	"regexp"

	core "dappco.re/go"
	brainclient "dappco.re/go/mcp/pkg/mcp/brain/client"
)

const seedDivider = "======================================================="

var (
	apiURL     = flag.String("api", brainclient.DefaultURL, "OpenBrain API base URL")
	apiKey     = flag.String("api-key", core.Env("CORE_BRAIN_KEY"), "OpenBrain API key (Bearer token)")
	server     = flag.String("server", "hosthub-agent", "Legacy MCP server ID flag; accepted for compatibility")
	org        = flag.String("org", core.Env("CORE_BRAIN_ORG"), "OpenBrain org for seeded memories")
	agent      = flag.String("agent", "charon", "Agent ID for attribution")
	dryRun     = flag.Bool("dry-run", false, "Preview without storing")
	plans      = flag.Bool("plans", false, "Also import plan documents")
	claudeMd   = flag.Bool("claude-md", false, "Also import CLAUDE.md files")
	memoryPath = flag.String("memory-path", "", "Override memory scan path (default: ~/.claude/projects/*/memory/)")
	planPath   = flag.String("plan-path", "", "Override plan scan path (default: ~/Code/*/docs/plans/)")
	codePath   = flag.String("code-path", "", "Override code root for CLAUDE.md scan (default: ~/Code)")
	maxChars   = flag.Int("max-chars", 3800, "Max chars per section (embeddinggemma limit ~4000)")
)

var openbrain *brainclient.Client

func main() {
	flag.Parse()

	core.Println("OpenBrain Seed — API Client")
	core.Println(seedDivider)

	if *apiKey == "" && !*dryRun {
		core.Println("ERROR: -api-key is required (or use -dry-run)")
		core.Println("  Generate one at: https://lthn.sh/admin/mcp/api-keys")
		os.Exit(1)
	}

	if *dryRun {
		core.Println("[DRY RUN] — no data will be stored")
	}

	core.Print(nil, "API: %s", *apiURL)
	core.Print(nil, "Org: %s | Agent: %s", *org, *agent)

	openbrain = brainclient.New(brainclient.Options{
		URL:     *apiURL,
		Key:     *apiKey,
		Org:     *org,
		AgentID: *agent,
	})

	// Discover memory files
	memPath := *memoryPath
	if memPath == "" {
		home, _ := os.UserHomeDir()
		memPath = filepath.Join(home, ".claude", "projects", "*", "memory")
	}
	memFiles, _ := filepath.Glob(filepath.Join(memPath, "*.md"))
	core.Print(nil, "\nFound %d memory files", len(memFiles))

	// Discover plan files
	var planFiles []string
	if *plans {
		pPath := *planPath
		if pPath == "" {
			home, _ := os.UserHomeDir()
			pPath = filepath.Join(home, "Code", "*", "docs", "plans")
		}
		planFiles, _ = filepath.Glob(filepath.Join(pPath, "*.md"))
		// Also check nested dirs (completed/, etc.)
		nested, _ := filepath.Glob(filepath.Join(pPath, "*", "*.md"))
		planFiles = append(planFiles, nested...)

		// Also check host-uk nested repos
		home, _ := os.UserHomeDir()
		hostUkPath := filepath.Join(home, "Code", "host-uk", "*", "docs", "plans")
		hostUkFiles, _ := filepath.Glob(filepath.Join(hostUkPath, "*.md"))
		planFiles = append(planFiles, hostUkFiles...)
		hostUkNested, _ := filepath.Glob(filepath.Join(hostUkPath, "*", "*.md"))
		planFiles = append(planFiles, hostUkNested...)

		core.Print(nil, "Found %d plan files", len(planFiles))
	}

	// Discover CLAUDE.md files
	var claudeFiles []string
	if *claudeMd {
		cPath := *codePath
		if cPath == "" {
			home, _ := os.UserHomeDir()
			cPath = filepath.Join(home, "Code")
		}
		claudeFiles = discoverClaudeMdFiles(cPath)
		core.Print(nil, "Found %d CLAUDE.md files", len(claudeFiles))
	}

	imported := 0
	skipped := 0
	errors := 0

	// Process memory files
	core.Println("\n--- Memory Files ---")
	for _, f := range memFiles {
		project := extractProject(f)
		sections := parseMarkdownSections(f)
		filename := core.TrimSuffix(filepath.Base(f), ".md")

		if len(sections) == 0 {
			core.Warn("brain-seed: skip file (no sections)", "project", project, "file", filename)
			skipped++
			continue
		}

		for _, sec := range sections {
			content := sec.heading + "\n\n" + sec.content
			if core.Trim(sec.content) == "" {
				skipped++
				continue
			}

			memType := inferType(sec.heading, sec.content, "memory")
			tags := buildTags(filename, "memory", project)
			confidence := confidenceForSource("memory")

			// Truncate to embedding model limit
			content = truncate(content, *maxChars)

			if *dryRun {
				core.Print(nil, "  [DRY] %s/%s :: %s (%s) — %d chars",
					project, filename, sec.heading, memType, len(content))
				imported++
				continue
			}

			if err := callBrainRemember(content, memType, tags, project, confidence); err != nil {
				core.Error("brain-seed: import failed", "project", project, "file", filename, "heading", sec.heading, "err", err)
				errors++
				continue
			}
			core.Print(nil, "  ok   %s/%s :: %s (%s)", project, filename, sec.heading, memType)
			imported++
		}
	}

	// Process plan files
	if *plans && len(planFiles) > 0 {
		core.Println("\n--- Plan Documents ---")
		for _, f := range planFiles {
			project := extractProjectFromPlan(f)
			sections := parseMarkdownSections(f)
			filename := core.TrimSuffix(filepath.Base(f), ".md")

			if len(sections) == 0 {
				skipped++
				continue
			}

			for _, sec := range sections {
				content := sec.heading + "\n\n" + sec.content
				if core.Trim(sec.content) == "" {
					skipped++
					continue
				}

				tags := buildTags(filename, "plans", project)
				content = truncate(content, *maxChars)

				if *dryRun {
					core.Print(nil, "  [DRY] %s :: %s / %s (plan) — %d chars",
						project, filename, sec.heading, len(content))
					imported++
					continue
				}

				if err := callBrainRemember(content, "plan", tags, project, 0.6); err != nil {
					core.Error("brain-seed: plan import failed", "project", project, "file", filename, "heading", sec.heading, "err", err)
					errors++
					continue
				}
				core.Print(nil, "  ok   %s :: %s / %s (plan)", project, filename, sec.heading)
				imported++
			}
		}
	}

	// Process CLAUDE.md files
	if *claudeMd && len(claudeFiles) > 0 {
		core.Println("\n--- CLAUDE.md Files ---")
		for _, f := range claudeFiles {
			project := extractProjectFromClaudeMd(f)
			sections := parseMarkdownSections(f)

			if len(sections) == 0 {
				skipped++
				continue
			}

			for _, sec := range sections {
				content := sec.heading + "\n\n" + sec.content
				if core.Trim(sec.content) == "" {
					skipped++
					continue
				}

				tags := buildTags("CLAUDE", "claude-md", project)
				content = truncate(content, *maxChars)

				if *dryRun {
					core.Print(nil, "  [DRY] %s :: CLAUDE.md / %s (convention) — %d chars",
						project, sec.heading, len(content))
					imported++
					continue
				}

				if err := callBrainRemember(content, "convention", tags, project, 0.9); err != nil {
					core.Error("brain-seed: claude-md import failed", "project", project, "heading", sec.heading, "err", err)
					errors++
					continue
				}
				core.Print(nil, "  ok   %s :: CLAUDE.md / %s (convention)", project, sec.heading)
				imported++
			}
		}
	}

	core.Print(nil, "\n%s", seedDivider)
	prefix := ""
	if *dryRun {
		prefix = "[DRY RUN] "
	}
	core.Print(nil, "%sImported: %d | Skipped: %d | Errors: %d", prefix, imported, skipped, errors)
}

// callBrainRemember sends a memory to OpenBrain via /v1/brain/remember.
func callBrainRemember(content, memType string, tags []string, project string, confidence float64) error {
	if openbrain == nil {
		openbrain = brainclient.New(brainclient.Options{
			URL:     *apiURL,
			Key:     *apiKey,
			Org:     *org,
			AgentID: *agent,
		})
	}

	input := brainclient.RememberInput{
		Content:    content,
		Type:       memType,
		Tags:       tags,
		Org:        *org,
		AgentID:    *agent,
		Confidence: confidence,
	}
	if project != "" && project != "unknown" {
		input.Project = project
	}
	_, err := openbrain.Remember(context.Background(), input)
	return core.Wrap(err, "callBrainRemember", "remember")
}

// truncate caps content to maxLen chars, appending an ellipsis if truncated.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	// Find last space before limit to avoid splitting mid-word
	cut := maxLen
	if idx := lastByteIndex(s[:maxLen], ' '); idx > maxLen-200 {
		cut = idx
	}
	return s[:cut] + "…"
}

func lastByteIndex(s string, target byte) int {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == target {
			return i
		}
	}
	return -1
}

// discoverClaudeMdFiles finds CLAUDE.md files across a code directory.
func discoverClaudeMdFiles(codePath string) []string {
	var files []string

	// Walk up to 4 levels deep, skip node_modules/vendor/.claude
	if err := filepath.WalkDir(codePath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			name := d.Name()
			if name == "node_modules" || name == "vendor" || name == ".claude" {
				return filepath.SkipDir
			}
			// Limit depth
			rel, _ := filepath.Rel(codePath, path)
			if len(core.Split(rel, string(os.PathSeparator))) > 4 {
				return filepath.SkipDir
			}
			return nil
		}
		if d.Name() == "CLAUDE.md" {
			files = append(files, path)
		}
		return nil
	}); err != nil {
		core.Error("brain-seed: failed to discover CLAUDE.md files", "path", codePath, "err", err)
	}

	return files
}

// section is a parsed markdown section.
type section struct {
	heading string
	content string
}

var headingRe = regexp.MustCompile(`^#{1,3}\s+(.+)$`)

// parseMarkdownSections splits a markdown file by headings.
func parseMarkdownSections(path string) []section {
	data, err := readLocal(path)
	if err != nil || len(data) == 0 {
		return nil
	}

	var sections []section
	lines := core.Split(data, "\n")
	var curHeading string
	var curContent []string

	for _, line := range lines {
		if m := headingRe.FindStringSubmatch(line); m != nil {
			if curHeading != "" && len(curContent) > 0 {
				text := core.Trim(core.Join("\n", curContent...))
				if text != "" {
					sections = append(sections, section{curHeading, text})
				}
			}
			curHeading = core.Trim(m[1])
			curContent = nil
		} else {
			curContent = append(curContent, line)
		}
	}

	// Flush last section
	if curHeading != "" && len(curContent) > 0 {
		text := core.Trim(core.Join("\n", curContent...))
		if text != "" {
			sections = append(sections, section{curHeading, text})
		}
	}

	// If no headings found, treat entire file as one section
	if len(sections) == 0 && core.Trim(data) != "" {
		sections = append(sections, section{
			heading: core.TrimSuffix(filepath.Base(path), ".md"),
			content: core.Trim(data),
		})
	}

	return sections
}

func readLocal(path string) (string, error) {
	r := (&core.Fs{}).New("/").Read(path)
	if !r.OK {
		if err, ok := r.Value.(error); ok && err != nil {
			return "", err
		}
		return "", core.E("brain-seed.readLocal", "failed to read file", nil)
	}
	data, ok := r.Value.(string)
	if !ok {
		return "", core.E("brain-seed.readLocal", "unexpected read result", nil)
	}
	return data, nil
}

// extractProject derives a project name from a Claude memory path.
// ~/.claude/projects/-Users-snider-Code-eaas/memory/MEMORY.md → "eaas"
func extractProject(path string) string {
	re := regexp.MustCompile(`projects/[^/]*-([^-/]+)/memory/`)
	if m := re.FindStringSubmatch(path); m != nil {
		return m[1]
	}
	return "unknown"
}

// extractProjectFromPlan derives a project name from a plan path.
// ~/Code/eaas/docs/plans/foo.md → "eaas"
// ~/Code/host-uk/core/docs/plans/foo.md → "core"
func extractProjectFromPlan(path string) string {
	// Check host-uk nested repos first
	re := regexp.MustCompile(`Code/host-uk/([^/]+)/docs/plans/`)
	if m := re.FindStringSubmatch(path); m != nil {
		return m[1]
	}
	re = regexp.MustCompile(`Code/([^/]+)/docs/plans/`)
	if m := re.FindStringSubmatch(path); m != nil {
		return m[1]
	}
	return "unknown"
}

// extractProjectFromClaudeMd derives a project name from a CLAUDE.md path.
// ~/Code/host-uk/core/CLAUDE.md → "core"
// ~/Code/eaas/CLAUDE.md → "eaas"
func extractProjectFromClaudeMd(path string) string {
	re := regexp.MustCompile(`Code/host-uk/([^/]+)/`)
	if m := re.FindStringSubmatch(path); m != nil {
		return m[1]
	}
	re = regexp.MustCompile(`Code/([^/]+)/`)
	if m := re.FindStringSubmatch(path); m != nil {
		return m[1]
	}
	return "unknown"
}

// inferType guesses the memory type from heading + content keywords.
func inferType(heading, content, source string) string {
	// Source-specific defaults (match PHP BrainIngestCommand behaviour)
	if source == "plans" {
		return "plan"
	}
	if source == "claude-md" {
		return "convention"
	}

	lower := core.Lower(heading + " " + content)
	patterns := map[string][]string{
		"architecture": {"architecture", "stack", "infrastructure", "layer", "service mesh"},
		"convention":   {"convention", "standard", "naming", "pattern", "rule", "coding"},
		"decision":     {"decision", "chose", "strategy", "approach", "domain"},
		"bug":          {"bug", "fix", "broken", "error", "issue", "lesson"},
		"plan":         {"plan", "todo", "roadmap", "milestone", "phase", "task"},
		"research":     {"research", "finding", "discovery", "analysis", "rfc"},
	}
	for t, keywords := range patterns {
		for _, kw := range keywords {
			if core.Contains(lower, kw) {
				return t
			}
		}
	}
	return "observation"
}

// buildTags creates the tag list for a memory.
func buildTags(filename, source, project string) []string {
	tags := []string{"source:" + source}
	if project != "" && project != "unknown" {
		tags = append(tags, "project:"+project)
	}
	if filename != "MEMORY" && filename != "CLAUDE" {
		tags = append(tags, core.Replace(core.Replace(filename, "-", " "), "_", " "))
	}
	return tags
}

// confidenceForSource returns a default confidence level matching the PHP ingest command.
func confidenceForSource(source string) float64 {
	switch source {
	case "claude-md":
		return 0.9
	case "memory":
		return 0.8
	case "plans":
		return 0.6
	default:
		return 0.5
	}
}
