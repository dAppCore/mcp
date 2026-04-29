// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"github.com/goccy/go-json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	core "dappco.re/go"
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

	srcDir := core.Path(out.WorkspaceDir, "src")
	cmd := shellCommand(context.Background(), srcDir, "git", "rev-parse", "--abbrev-ref", "HEAD")
	data, err := cmd.Output()
	if err != nil {
		t.Fatalf("failed to read branch: %v", err)
	}
	if got := core.Trim(string(data)); got != want {
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

func TestDispatchIssue_Good_UnlocksOnPrepFailure(t *testing.T) {
	var methods []string
	var bodies []string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		methods = append(methods, r.Method)
		bodies = append(bodies, string(body))

		switch r.Method {
		case http.MethodGet:
			_ = json.NewEncoder(w).Encode(map[string]any{
				"title": "Fix login crash",
				"body":  "details",
				"state": "open",
				"labels": []map[string]any{
					{"name": "bug"},
				},
			})
		case http.MethodPatch:
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))
	defer srv.Close()

	s := &PrepSubsystem{
		forgeURL:   srv.URL,
		forgeToken: "token",
		client:     srv.Client(),
		codePath:   t.TempDir(),
	}

	_, _, err := s.dispatchIssue(context.Background(), nil, IssueDispatchInput{
		Repo:  "demo",
		Org:   "core",
		Issue: 42,
	})
	if err == nil {
		t.Fatal("expected dispatch to fail when the repo clone is missing")
	}

	if got, want := len(methods), 3; got != want {
		t.Fatalf("expected %d requests, got %d (%v)", want, got, methods)
	}
	if methods[0] != http.MethodGet {
		t.Fatalf("expected first request to fetch issue, got %s", methods[0])
	}
	if methods[1] != http.MethodPatch {
		t.Fatalf("expected second request to lock issue, got %s", methods[1])
	}
	if methods[2] != http.MethodPatch {
		t.Fatalf("expected third request to unlock issue, got %s", methods[2])
	}
	if !core.Contains(bodies[1], `"assignees":["claude"]`) {
		t.Fatalf("expected lock request to assign claude, got %s", bodies[1])
	}
	if !core.Contains(bodies[2], `"assignees":[]`) {
		t.Fatalf("expected unlock request to clear assignees, got %s", bodies[2])
	}
	if !core.Contains(bodies[2], `"labels":["bug"]`) {
		t.Fatalf("expected unlock request to preserve original labels, got %s", bodies[2])
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
	if !core.Contains(string(gotBody), `"assignees":["claude"]`) {
		t.Fatalf("expected assignee in body, got %s", string(gotBody))
	}
	if !core.Contains(string(gotBody), `"in-progress"`) {
		t.Fatalf("expected in-progress label in body, got %s", string(gotBody))
	}
}

func initTestRepo(t *testing.T, codePath, repo string) string {
	t.Helper()

	repoDir := core.Path(codePath, "core", repo)
	if r := core.MkdirAll(repoDir, 0o755); !r.OK {
		t.Fatalf("mkdir repo dir: %v", resultError(r))
	}

	run := func(args ...string) {
		t.Helper()
		cmd := shellCommand(context.Background(), repoDir, "git", args...)
		cmd.Env = append(core.Environ(),
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
	if r := core.WriteFile(core.Path(repoDir, "README.md"), []byte("# demo\n"), 0o644); !r.OK {
		t.Fatalf("write file: %v", resultError(r))
	}
	run("add", "README.md")
	run("commit", "-m", "initial commit")

	return repoDir
}
