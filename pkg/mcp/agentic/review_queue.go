// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"regexp"
	"time"

	core "dappco.re/go"
	coreio "dappco.re/go/io"
	coremcp "dappco.re/go/mcp/pkg/mcp"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ReviewQueueInput controls the review queue runner.
type ReviewQueueInput struct {
	Limit     int    `json:"limit,omitempty"`
	Reviewer  string `json:"reviewer,omitempty"`
	DryRun    bool   `json:"dry_run,omitempty"`
	LocalOnly bool   `json:"local_only,omitempty"`
}

// ReviewQueueOutput reports what happened.
type ReviewQueueOutput struct {
	Success   bool           `json:"success"`
	Processed []ReviewResult `json:"processed"`
	Skipped   []string       `json:"skipped,omitempty"`
	RateLimit *RateLimitInfo `json:"rate_limit,omitempty"`
}

// ReviewResult is the outcome of reviewing one repo.
type ReviewResult struct {
	Repo     string `json:"repo"`
	Verdict  string `json:"verdict"`
	Findings int    `json:"findings"`
	Action   string `json:"action"`
	Detail   string `json:"detail,omitempty"`
}

// RateLimitInfo tracks review rate limit state.
type RateLimitInfo struct {
	Limited bool      `json:"limited"`
	RetryAt time.Time `json:"retry_at,omitempty"`
	Message string    `json:"message,omitempty"`
}

func reviewQueueHomeDir() string {
	if home := os.Getenv("DIR_HOME"); home != "" {
		return home
	}
	home, _ := os.UserHomeDir()
	return home
}

func (s *PrepSubsystem) registerReviewQueueTool(svc *coremcp.Service) {
	server := svc.Server()
	coremcp.AddToolRecorded(svc, server, "agentic", &mcp.Tool{
		Name:        "agentic_review_queue",
		Description: "Process repositories that are ahead of the GitHub mirror and summarise review findings.",
	}, s.reviewQueue)
}

func (s *PrepSubsystem) reviewQueue(ctx context.Context, _ *mcp.CallToolRequest, input ReviewQueueInput) (*mcp.CallToolResult, ReviewQueueOutput, error) {
	limit := input.Limit
	if limit <= 0 {
		limit = 4
	}

	basePath := repoRootFromCodePath(s.codePath)
	candidates := s.findReviewCandidates(basePath)
	if len(candidates) == 0 {
		return nil, ReviewQueueOutput{Success: true, Processed: []ReviewResult{}}, nil
	}

	processed := make([]ReviewResult, 0, len(candidates))
	skipped := make([]string, 0)
	var rateInfo *RateLimitInfo

	for _, repo := range candidates {
		if len(processed) >= limit {
			skipped = append(skipped, repo+" (limit reached)")
			continue
		}

		if rateInfo != nil && rateInfo.Limited && time.Now().Before(rateInfo.RetryAt) {
			skipped = append(skipped, repo+" (rate limited)")
			continue
		}

		repoDir := core.Path(basePath, repo)
		reviewer := input.Reviewer
		if reviewer == "" {
			reviewer = "coderabbit"
		}

		result := s.reviewRepo(ctx, repoDir, repo, reviewer, input.DryRun, input.LocalOnly)
		if result.Verdict == "rate_limited" {
			retryAfter := parseRetryAfter(result.Detail)
			rateInfo = &RateLimitInfo{
				Limited: true,
				RetryAt: time.Now().Add(retryAfter),
				Message: result.Detail,
			}
			skipped = append(skipped, repo+" (rate limited)")
			continue
		}

		processed = append(processed, result)
	}

	if rateInfo != nil {
		s.saveRateLimitState(rateInfo)
	}

	return nil, ReviewQueueOutput{
		Success:   true,
		Processed: processed,
		Skipped:   skipped,
		RateLimit: rateInfo,
	}, nil
}

func (s *PrepSubsystem) findReviewCandidates(basePath string) []string {
	entries, err := os.ReadDir(basePath)
	if err != nil {
		return nil
	}

	candidates := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		repoDir := core.Path(basePath, entry.Name())
		if !hasRemote(repoDir, "github") {
			continue
		}
		if commitsAhead(repoDir, "github/main", "HEAD") <= 0 {
			continue
		}
		candidates = append(candidates, entry.Name())
	}
	return candidates
}

func (s *PrepSubsystem) reviewRepo(ctx context.Context, repoDir, repo, reviewer string, dryRun, localOnly bool) ReviewResult {
	result := ReviewResult{Repo: repo}

	if rl := s.loadRateLimitState(); rl != nil && rl.Limited && time.Now().Before(rl.RetryAt) {
		result.Verdict = "rate_limited"
		result.Detail = core.Sprintf("retry after %s", rl.RetryAt.Format(time.RFC3339))
		return result
	}

	cmd := reviewerCommand(ctx, repoDir, reviewer)
	cmd.Dir = repoDir
	out, err := cmd.CombinedOutput()
	output := core.Trim(string(out))

	if core.Contains(core.Lower(output), "rate limit") {
		result.Verdict = "rate_limited"
		result.Detail = output
		return result
	}

	if err != nil && !core.Contains(output, "No findings") && !core.Contains(output, "no issues") {
		result.Verdict = "error"
		if output != "" {
			result.Detail = output
		} else {
			result.Detail = err.Error()
		}
		return result
	}

	s.storeReviewOutput(repoDir, repo, reviewer, output)
	result.Findings = countFindingHints(output)

	if core.Contains(output, "No findings") || core.Contains(output, "no issues") || core.Contains(output, "LGTM") {
		result.Verdict = "clean"
		if dryRun {
			result.Action = "skipped (dry run)"
			return result
		}
		if localOnly {
			result.Action = "local only"
			return result
		}

		if url, err := readGitHubPRURL(repoDir); err == nil && url != "" {
			mergeCmd := exec.CommandContext(ctx, "gh", "pr", "merge", "--auto", "--squash", "--delete-branch")
			mergeCmd.Dir = repoDir
			if mergeOut, err := mergeCmd.CombinedOutput(); err == nil {
				result.Action = "merged"
				result.Detail = core.Trim(string(mergeOut))
				return result
			}
		}

		result.Action = "waiting"
		return result
	}

	result.Verdict = "findings"
	if dryRun {
		result.Action = "skipped (dry run)"
		return result
	}

	result.Action = "waiting"
	return result
}

func (s *PrepSubsystem) storeReviewOutput(repoDir, repo, reviewer, output string) {
	home := reviewQueueHomeDir()
	dataDir := core.Path(home, ".core", "training", "reviews")
	if err := coreio.Local.EnsureDir(dataDir); err != nil {
		return
	}

	payload := map[string]string{
		"repo":     repo,
		"reviewer": reviewer,
		"output":   output,
		"source":   repoDir,
	}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return
	}

	name := core.Sprintf("%s-%s-%d.json", repo, reviewer, time.Now().Unix())
	_ = writeAtomic(core.Path(dataDir, name), string(data))
}

func (s *PrepSubsystem) saveRateLimitState(info *RateLimitInfo) {
	home := reviewQueueHomeDir()
	path := core.Path(home, ".core", "coderabbit-ratelimit.json")
	data, err := json.Marshal(info)
	if err != nil {
		return
	}
	_ = writeAtomic(path, string(data))
}

func (s *PrepSubsystem) loadRateLimitState() *RateLimitInfo {
	home := reviewQueueHomeDir()
	path := core.Path(home, ".core", "coderabbit-ratelimit.json")
	data, err := coreio.Local.Read(path)
	if err != nil {
		return nil
	}

	var info RateLimitInfo
	if err := json.Unmarshal([]byte(data), &info); err != nil {
		return nil
	}
	if !info.Limited {
		return nil
	}
	return &info
}

func countFindingHints(output string) int {
	re := regexp.MustCompile(`(?m)[^ \t\n\r]+\.(?:go|php|ts|tsx|js|jsx|py|rb|java|cs|cpp|cxx|cc|md):\d+`)
	return len(re.FindAllString(output, -1))
}
