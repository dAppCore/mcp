// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"testing"
	"time"

	core "dappco.re/go"
)

func TestHelpers_parsePositiveInt_Good(t *testing.T) {
	cases := map[string]int{
		"0":     0,
		"7":     7,
		"  42 ": 42,
		"1000":  1000,
	}
	for in, want := range cases {
		got, err := parsePositiveInt(in)
		if err != nil {
			t.Fatalf("parsePositiveInt(%q): unexpected error %v", in, err)
		}
		if got != want {
			t.Fatalf("parsePositiveInt(%q) = %d, want %d", in, got, want)
		}
	}
}

func TestHelpers_parsePositiveInt_Bad_Empty(t *testing.T) {
	if _, err := parsePositiveInt("   "); err == nil {
		t.Fatal("expected error for empty value")
	}
}

func TestHelpers_parsePositiveInt_Ugly_NonNumeric(t *testing.T) {
	for _, in := range []string{"12a", "-5", "3.14", "abc"} {
		if _, err := parsePositiveInt(in); err == nil {
			t.Fatalf("expected error for %q", in)
		}
	}
}

func TestHelpers_itoa_Good(t *testing.T) {
	if itoa(0) != "0" {
		t.Fatalf("itoa(0) = %q", itoa(0))
	}
	if itoa(-12) != "-12" {
		t.Fatalf("itoa(-12) = %q", itoa(-12))
	}
	if itoa(2048) != "2048" {
		t.Fatalf("itoa(2048) = %q", itoa(2048))
	}
}

func TestHelpers_parseRetryAfter_Good_Units(t *testing.T) {
	cases := map[string]time.Duration{
		"retry in 3 minutes": 3 * time.Minute,
		"wait 2 hours":       2 * time.Hour,
		"try 45 seconds":     45 * time.Second,
		"30 SECONDS please":  30 * time.Second,
	}
	for in, want := range cases {
		if got := parseRetryAfter(in); got != want {
			t.Fatalf("parseRetryAfter(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestHelpers_parseRetryAfter_Ugly_NoMatchDefaults(t *testing.T) {
	for _, in := range []string{"", "no number here", "soon", "0 minutes"} {
		if got := parseRetryAfter(in); got != 5*time.Minute {
			t.Fatalf("parseRetryAfter(%q) = %v, want default 5m", in, got)
		}
	}
}

func TestHelpers_repoRootFromCodePath_Good(t *testing.T) {
	got := repoRootFromCodePath("/home/agent/work")
	want := core.Path("/home/agent/work", "core")
	if got != want {
		t.Fatalf("repoRootFromCodePath = %q, want %q", got, want)
	}
}

func TestHelpers_shellQuote_Good_Bad_Ugly(t *testing.T) {
	if shellQuote("") != "''" {
		t.Fatalf("empty value should quote to '', got %q", shellQuote(""))
	}
	if shellQuote("simple") != "'simple'" {
		t.Fatalf("simple value quote = %q", shellQuote("simple"))
	}
	// Embedded single quote must be escaped using the '"'"' idiom.
	got := shellQuote("a'b")
	if got != `'a'"'"'b'` {
		t.Fatalf("single-quote escape = %q", got)
	}
}

func TestHelpers_shellJoin_Good(t *testing.T) {
	got := shellJoin("git", "rev-list", "--count", "a..b")
	want := "'git' 'rev-list' '--count' 'a..b'"
	if got != want {
		t.Fatalf("shellJoin = %q, want %q", got, want)
	}
}

func TestHelpers_baseAgent_Good(t *testing.T) {
	cases := map[string]string{
		"codex":             "codex",
		"codex:gpt-5.4":     "codex",
		"claude:opus:extra": "claude",
		"":                  "",
	}
	for in, want := range cases {
		if got := baseAgent(in); got != want {
			t.Fatalf("baseAgent(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestHelpers_parseSimpleInt_Good_Bad(t *testing.T) {
	if n, ok := parseSimpleInt(" 12 "); !ok || n != 12 {
		t.Fatalf("parseSimpleInt(\" 12 \") = (%d,%v)", n, ok)
	}
	for _, in := range []string{"", "12x", "-1", "1.5"} {
		if _, ok := parseSimpleInt(in); ok {
			t.Fatalf("expected parseSimpleInt(%q) to fail", in)
		}
	}
}

func TestHelpers_countFindingHints_Good(t *testing.T) {
	output := "pkg/mcp/foo.go:12 has an issue\nsrc/app.ts:340 another\nplain prose with no ref\nlib/x.py:7 third"
	if got := countFindingHints(output); got != 3 {
		t.Fatalf("countFindingHints = %d, want 3", got)
	}
}

func TestHelpers_countFindingHints_Ugly_NoMatches(t *testing.T) {
	if got := countFindingHints("nothing here resembles a file ref"); got != 0 {
		t.Fatalf("countFindingHints = %d, want 0", got)
	}
}

func TestHelpers_sanitizeFilename_Good_Ugly(t *testing.T) {
	// Allowed chars survive; everything else becomes a dash.
	if got := sanitizeFilename("My_Plan-v1.2"); got != "My_Plan-v1.2" {
		t.Fatalf("sanitizeFilename allowed-set = %q", got)
	}
	if got := sanitizeFilename("a b/c\\d"); got != "a-b-c-d" {
		t.Fatalf("sanitizeFilename = %q, want a-b-c-d", got)
	}
}

func TestHelpers_trimDashes_Good(t *testing.T) {
	cases := map[string]string{
		"--abc--": "abc",
		"abc":     "abc",
		"---":     "",
		"-a-b-":   "a-b",
	}
	for in, want := range cases {
		if got := trimDashes(in); got != want {
			t.Fatalf("trimDashes(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestHelpers_listLocalRepos_Good(t *testing.T) {
	root := t.TempDir()
	// Two directories (repos) and one file (ignored).
	if err := coreio.Local.EnsureDir(core.Path(root, "repo-a")); err != nil {
		t.Fatalf("mkdir repo-a: %v", err)
	}
	if err := coreio.Local.EnsureDir(core.Path(root, "repo-b")); err != nil {
		t.Fatalf("mkdir repo-b: %v", err)
	}
	if err := writeAtomic(core.Path(root, "loose.txt"), "x"); err != nil {
		t.Fatalf("write loose file: %v", err)
	}

	repos := listLocalRepos(root)
	if len(repos) != 2 {
		t.Fatalf("expected 2 repos, got %d (%v)", len(repos), repos)
	}
}

func TestHelpers_listLocalRepos_Ugly_MissingDir(t *testing.T) {
	if repos := listLocalRepos(core.Path(t.TempDir(), "does-not-exist")); repos != nil {
		t.Fatalf("expected nil for missing dir, got %v", repos)
	}
}

func TestHelpers_buildPRBody_Good(t *testing.T) {
	sub := &PrepSubsystem{}
	body := sub.buildPRBody(&WorkspaceStatus{
		Task:  "Implement notifications",
		Issue: 1490,
		Agent: "codex",
		Runs:  3,
	})
	for _, want := range []string{"## Summary", "Implement notifications", "Closes #1490", "**Agent:** codex", "**Runs:** 3"} {
		if !core.Contains(body, want) {
			t.Fatalf("buildPRBody missing %q in:\n%s", want, body)
		}
	}
}

func TestHelpers_buildPRBody_Ugly_MinimalStatus(t *testing.T) {
	sub := &PrepSubsystem{}
	body := sub.buildPRBody(&WorkspaceStatus{Agent: "claude"})
	// No task, no issue: those sections must be omitted.
	if core.Contains(body, "Closes #") {
		t.Fatalf("expected no Closes line for zero issue:\n%s", body)
	}
	if !core.Contains(body, "**Agent:** claude") {
		t.Fatalf("expected agent line:\n%s", body)
	}
}
