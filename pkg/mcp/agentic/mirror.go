// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"

	coremcp "dappco.re/go/mcp/pkg/mcp"
	coreerr "dappco.re/go/core/log"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// MirrorInput controls Forge to GitHub mirror sync.
type MirrorInput struct {
	Repo     string `json:"repo,omitempty"`
	DryRun   bool   `json:"dry_run,omitempty"`
	MaxFiles int    `json:"max_files,omitempty"`
}

// MirrorOutput reports mirror sync results.
type MirrorOutput struct {
	Success bool         `json:"success"`
	Synced  []MirrorSync `json:"synced"`
	Skipped []string     `json:"skipped,omitempty"`
	Count   int          `json:"count"`
}

// MirrorSync records one repo sync attempt.
type MirrorSync struct {
	Repo         string `json:"repo"`
	CommitsAhead int    `json:"commits_ahead"`
	FilesChanged int    `json:"files_changed"`
	PRURL        string `json:"pr_url,omitempty"`
	Pushed       bool   `json:"pushed"`
	Skipped      string `json:"skipped,omitempty"`
}

func (s *PrepSubsystem) registerMirrorTool(svc *coremcp.Service) {
	server := svc.Server()
	coremcp.AddToolRecorded(svc, server, "agentic", &mcp.Tool{
		Name:        "agentic_mirror",
		Description: "Mirror Forge repositories to GitHub and open a GitHub PR when there are commits ahead of the remote mirror.",
	}, s.mirror)
}

func (s *PrepSubsystem) mirror(ctx context.Context, _ *mcp.CallToolRequest, input MirrorInput) (*mcp.CallToolResult, MirrorOutput, error) {
	maxFiles := input.MaxFiles
	if maxFiles <= 0 {
		maxFiles = 50
	}

	basePath := repoRootFromCodePath(s.codePath)
	repos := []string{}
	if input.Repo != "" {
		repos = []string{input.Repo}
	} else {
		repos = listLocalRepos(basePath)
	}

	synced := make([]MirrorSync, 0, len(repos))
	skipped := make([]string, 0)

	for _, repo := range repos {
		repoDir := filepath.Join(basePath, repo)
		if !hasRemote(repoDir, "github") {
			skipped = append(skipped, repo+": no github remote")
			continue
		}

		if _, err := exec.LookPath("git"); err != nil {
			return nil, MirrorOutput{}, coreerr.E("mirror", "git CLI is not available", err)
		}

		_, _ = gitOutput(repoDir, "fetch", "github")
		ahead := commitsAhead(repoDir, "github/main", "HEAD")
		if ahead <= 0 {
			continue
		}

		files := filesChanged(repoDir, "github/main", "HEAD")
		sync := MirrorSync{
			Repo:         repo,
			CommitsAhead: ahead,
			FilesChanged: files,
		}

		if files > maxFiles {
			sync.Skipped = fmt.Sprintf("%d files exceeds limit of %d", files, maxFiles)
			synced = append(synced, sync)
			continue
		}

		if input.DryRun {
			sync.Skipped = "dry run"
			synced = append(synced, sync)
			continue
		}

		if err := ensureDevBranch(repoDir); err != nil {
			sync.Skipped = err.Error()
			synced = append(synced, sync)
			continue
		}
		sync.Pushed = true

		prURL, err := createGitHubPR(ctx, repoDir, repo, ahead, files)
		if err != nil {
			sync.Skipped = err.Error()
		} else {
			sync.PRURL = prURL
		}

		synced = append(synced, sync)
	}

	return nil, MirrorOutput{
		Success: true,
		Synced:  synced,
		Skipped: skipped,
		Count:   len(synced),
	}, nil
}
