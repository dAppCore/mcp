// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"github.com/goccy/go-json"
	"regexp"
	"strconv"
	"time"

	core "dappco.re/go"
)

func listLocalRepos(basePath string) []string {
	entries, err := coreio.Local.List(basePath)
	if err != nil {
		return nil
	}

	repos := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			repos = append(repos, entry.Name())
		}
	}
	return repos
}

func hasRemote(repoDir, remote string) bool {
	cmd := shellCommand(context.Background(), repoDir, "git", "remote", "get-url", remote)
	if out, err := cmd.Output(); err == nil {
		return core.Trim(string(out)) != ""
	}
	return false
}

func commitsAhead(repoDir, baseRef, headRef string) int {
	cmd := shellCommand(context.Background(), repoDir, "git", "rev-list", "--count", baseRef+".."+headRef)
	out, err := cmd.Output()
	if err != nil {
		return 0
	}

	count, err := parsePositiveInt(core.Trim(string(out)))
	if err != nil {
		return 0
	}
	return count
}

func filesChanged(repoDir, baseRef, headRef string) int {
	cmd := shellCommand(context.Background(), repoDir, "git", "diff", "--name-only", baseRef+".."+headRef)
	out, err := cmd.Output()
	if err != nil {
		return 0
	}

	count := 0
	for _, line := range core.Split(core.Trim(string(out)), "\n") {
		if core.Trim(line) != "" {
			count++
		}
	}
	return count
}

func gitOutput(repoDir string, args ...string) (
	string,
	error,
) {
	cmd := shellCommand(context.Background(), repoDir, "git", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", core.E("gitOutput", string(out), err)
	}
	return core.Trim(string(out)), nil
}

func parsePositiveInt(value string) (
	int,
	error,
) {
	value = core.Trim(value)
	if value == "" {
		return 0, core.E("parsePositiveInt", "empty value", nil)
	}
	n := 0
	for _, r := range value {
		if r < '0' || r > '9' {
			return 0, core.E("parsePositiveInt", "value contains non-numeric characters", nil)
		}
		n = n*10 + int(r-'0')
	}
	return n, nil
}

func readGitHubPRURL(repoDir string) (
	string,
	error,
) {
	cmd := shellCommand(context.Background(), repoDir, "gh", "pr", "list", "--head", "dev", "--state", "open", "--json", "url", "--limit", "1")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}

	var rows []struct {
		URL string `json:"url"`
	}
	if err := json.Unmarshal(out, &rows); err != nil {
		return "", err
	}
	if len(rows) == 0 {
		return "", nil
	}
	return rows[0].URL, nil
}

func createGitHubPR(ctx context.Context, repoDir, repo string, commits, files int) (
	string,
	error,
) {
	if err := commandAvailable("gh"); err != nil {
		return "", core.E("createGitHubPR", "gh CLI is not available", err)
	}

	if url, err := readGitHubPRURL(repoDir); err == nil && url != "" {
		return url, nil
	}

	body := "## Forge -> GitHub Sync\n\n"
	body += "**Commits:** " + itoa(commits) + "\n"
	body += "**Files changed:** " + itoa(files) + "\n\n"
	body += "Automated sync from Forge (forge.lthn.ai) to GitHub mirror.\n"
	body += "Review with CodeRabbit before merging.\n\n"
	body += "---\n"
	body += "Co-Authored-By: Virgil <virgil@lethean.io>"

	title := "[sync] " + repo + ": " + itoa(commits) + " commits, " + itoa(files) + " files"

	cmd := shellCommand(ctx, repoDir, "gh", "pr", "create",
		"--head", "dev",
		"--base", "main",
		"--title", title,
		"--body", body,
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", core.E("createGitHubPR", string(out), err)
	}

	lines := core.Split(core.Trim(string(out)), "\n")
	if len(lines) == 0 {
		return "", nil
	}
	return core.Trim(lines[len(lines)-1]), nil
}

func ensureDevBranch(
	repoDir string,
) (
	_ error, // result
) {
	cmd := shellCommand(context.Background(), repoDir, "git", "push", "github", "HEAD:refs/heads/dev", "--force")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return core.E("ensureDevBranch", string(out), err)
	}
	return nil
}

func reviewerCommand(ctx context.Context, repoDir, reviewer string) *core.Cmd {
	switch reviewer {
	case "coderabbit":
		return shellCommand(ctx, repoDir, "coderabbit", "review")
	case "codex":
		return shellCommand(ctx, repoDir, "codex", "review")
	case "both":
		return shellCommand(ctx, repoDir, "coderabbit", "review")
	default:
		return shellCommand(ctx, repoDir, reviewer)
	}
}

func itoa(value int) string {
	return strconv.Itoa(value)
}

func parseRetryAfter(detail string) time.Duration {
	re := regexp.MustCompile(`(?i)(\d+)\s*(minute|minutes|hour|hours|second|seconds)`)
	match := re.FindStringSubmatch(detail)
	if len(match) != 3 {
		return 5 * time.Minute
	}

	n, err := strconv.Atoi(match[1])
	if err != nil || n <= 0 {
		return 5 * time.Minute
	}

	switch core.Lower(match[2]) {
	case "hour", "hours":
		return time.Duration(n) * time.Hour
	case "second", "seconds":
		return time.Duration(n) * time.Second
	default:
		return time.Duration(n) * time.Minute
	}
}

func repoRootFromCodePath(codePath string) string {
	return core.Path(codePath, "core")
}

func shellCommand(ctx context.Context, dir, command string, args ...string) *core.Cmd {
	script := "exec " + shellJoin(command, args...)
	cmd := &core.Cmd{
		Path: "/bin/sh",
		Args: []string{"sh", "-c", script},
		Dir:  dir,
	}
	if ctx != nil && ctx.Done() != nil {
		go func() {
			<-ctx.Done()
			if cmd.Process != nil {
				if err := cmd.Process.Kill(); err != nil {
					core.Error("agentic command kill failed", "err", err)
				}
			}
		}()
	}
	return cmd
}

func shellJoin(command string, args ...string) string {
	parts := []string{shellQuote(command)}
	for _, arg := range args {
		parts = append(parts, shellQuote(arg))
	}
	return core.Join(" ", parts...)
}

func shellQuote(value string) string {
	if value == "" {
		return "''"
	}
	return "'" + core.Replace(value, "'", "'\"'\"'") + "'"
}

func commandAvailable(command string) (
	_ error, // result
) {
	cmd := &core.Cmd{
		Path: "/bin/sh",
		Args: []string{"sh", "-c", "command -v " + shellQuote(command)},
	}
	out, err := cmd.Output()
	if err != nil {
		return err
	}
	if core.Trim(string(out)) == "" {
		return core.E("commandAvailable", "command not found: "+command, nil)
	}
	return nil
}

func resultError(result core.Result) (
	_ error, // result
) {
	if err, ok := result.Value.(error); ok {
		return err
	}
	return core.E("core result failed", "no error value", nil)
}
