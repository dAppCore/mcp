// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestBranchSlug_Good(t *testing.T) {
	got := branchSlug("Fix login crash in API v2")
	want := "fix-login-crash-in-api-v2"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestPrepWorkspace_Good_IssueBranchName(t *testing.T) {
	codePath := t.TempDir()
	repoDir := initTestRepo(t, codePath, "demo")
	_ = repoDir

	s := &PrepSubsystem{codePath: codePath}
	_, out, err := s.prepWorkspace(context.Background(), nil, PrepInput{
		Repo:  "demo",
		Issue: 42,
		Task:  "Fix login crash",
	})
	if err != nil {
		t.Fatalf("prepWorkspace failed: %v", err)
	}

	want := "agent/issue-42-fix-login-crash"
	if out.Branch != want {
		t.Fatalf("expected branch %q, got %q", want, out.Branch)
	}

	srcDir := filepath.Join(out.WorkspaceDir, "src")
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = srcDir
	data, err := cmd.Output()
	if err != nil {
		t.Fatalf("failed to read branch: %v", err)
	}
	if got := strings.TrimSpace(string(data)); got != want {
		t.Fatalf("expected git branch %q, got %q", want, got)
	}
}

func TestDispatchIssue_Bad_AssignedIssue(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			_ = json.NewEncoder(w).Encode(map[string]any{
				"title": "Fix login crash",
				"body":  "details",
				"state": "open",
				"assignee": map[string]any{
					"login": "someone-else",
				},
			})
		default:
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer srv.Close()

	s := &PrepSubsystem{
		forgeURL: srv.URL,
		client:   srv.Client(),
	}

	_, _, err := s.dispatchIssue(context.Background(), nil, IssueDispatchInput{
		Repo:   "demo",
		Org:    "core",
		Issue:  42,
		DryRun: true,
	})
	if err == nil {
		t.Fatal("expected assigned issue to fail")
	}
}

func TestLockIssue_Good_RequestBody(t *testing.T) {
	var gotMethod string
	var gotPath string
	var gotBody []byte

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		body, _ := io.ReadAll(r.Body)
		gotBody = append([]byte(nil), body...)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	s := &PrepSubsystem{
		forgeURL: srv.URL,
		client:   srv.Client(),
	}

	if err := s.lockIssue(context.Background(), "core", "demo", 42, "claude"); err != nil {
		t.Fatalf("lockIssue failed: %v", err)
	}

	if gotMethod != http.MethodPatch {
		t.Fatalf("expected PATCH, got %s", gotMethod)
	}
	if gotPath != "/api/v1/repos/core/demo/issues/42" {
		t.Fatalf("unexpected path %q", gotPath)
	}
	if !bytes.Contains(gotBody, []byte(`"assignees":["claude"]`)) {
		t.Fatalf("expected assignee in body, got %s", string(gotBody))
	}
	if !bytes.Contains(gotBody, []byte(`"in-progress"`)) {
		t.Fatalf("expected in-progress label in body, got %s", string(gotBody))
	}
}

func initTestRepo(t *testing.T, codePath, repo string) string {
	t.Helper()

	repoDir := filepath.Join(codePath, "core", repo)
	if err := os.MkdirAll(repoDir, 0o755); err != nil {
		t.Fatalf("mkdir repo dir: %v", err)
	}

	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = repoDir
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=Test User",
			"GIT_AUTHOR_EMAIL=test@example.com",
			"GIT_COMMITTER_NAME=Test User",
			"GIT_COMMITTER_EMAIL=test@example.com",
		)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v failed: %v\n%s", args, err, string(out))
		}
	}

	run("init", "-b", "main")
	if err := os.WriteFile(filepath.Join(repoDir, "README.md"), []byte("# demo\n"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	run("add", "README.md")
	run("commit", "-m", "initial commit")

	return repoDir
}
