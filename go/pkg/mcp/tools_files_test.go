// SPDX-License-Identifier: EUPL-1.2

package mcp

import (
	"context"
	"testing"
)

// newTmpService returns a workspace-medium-backed Service rooted at a temp dir.
func newTmpService(t *testing.T) *Service {
	t.Helper()
	s, err := New(Options{WorkspaceRoot: t.TempDir()})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if s.medium == nil {
		t.Fatal("expected workspace medium to be set")
	}
	return s
}

func TestFiles_createDirectory_Good(t *testing.T) {
	s := newTmpService(t)
	ctx := context.Background()

	_, out, err := s.createDirectory(ctx, nil, CreateDirectoryInput{Path: "sub/dir"})
	if err != nil {
		t.Fatalf("createDirectory: %v", err)
	}
	if !out.Success || out.Path != "sub/dir" {
		t.Fatalf("unexpected output: %+v", out)
	}

	_, exists, err := s.fileExists(ctx, nil, FileExistsInput{Path: "sub/dir"})
	if err != nil {
		t.Fatalf("fileExists: %v", err)
	}
	if !exists.Exists || !exists.IsDir {
		t.Fatalf("expected created directory to exist: %+v", exists)
	}
}

func TestFiles_deleteFile_Good(t *testing.T) {
	s := newTmpService(t)
	ctx := context.Background()

	if err := s.medium.Write("doomed.txt", "bye"); err != nil {
		t.Fatalf("seed write: %v", err)
	}

	_, out, err := s.deleteFile(ctx, nil, DeleteFileInput{Path: "doomed.txt"})
	if err != nil {
		t.Fatalf("deleteFile: %v", err)
	}
	if !out.Success {
		t.Fatal("expected success")
	}
	if s.medium.Exists("doomed.txt") {
		t.Fatal("expected file to be gone after delete")
	}
}

func TestFiles_renameFile_Good(t *testing.T) {
	s := newTmpService(t)
	ctx := context.Background()

	if err := s.medium.Write("from.txt", "content"); err != nil {
		t.Fatalf("seed write: %v", err)
	}

	_, out, err := s.renameFile(ctx, nil, RenameFileInput{OldPath: "from.txt", NewPath: "to.txt"})
	if err != nil {
		t.Fatalf("renameFile: %v", err)
	}
	if !out.Success || out.NewPath != "to.txt" {
		t.Fatalf("unexpected output: %+v", out)
	}
	if s.medium.Exists("from.txt") {
		t.Fatal("expected old path to be gone")
	}
	if !s.medium.Exists("to.txt") {
		t.Fatal("expected new path to exist")
	}
}

func TestFiles_fileExists_Ugly_Missing(t *testing.T) {
	s := newTmpService(t)
	_, out, err := s.fileExists(context.Background(), nil, FileExistsInput{Path: "nope.txt"})
	if err != nil {
		t.Fatalf("fileExists: %v", err)
	}
	if out.Exists {
		t.Fatal("expected Exists false for missing path")
	}
}

func TestFiles_editDiff_Good_SingleReplace(t *testing.T) {
	s := newTmpService(t)
	ctx := context.Background()

	if err := s.medium.Write("code.go", "alpha beta alpha"); err != nil {
		t.Fatalf("seed write: %v", err)
	}

	_, out, err := s.editDiff(ctx, nil, EditDiffInput{
		Path:      "code.go",
		OldString: "alpha",
		NewString: "gamma",
	})
	if err != nil {
		t.Fatalf("editDiff: %v", err)
	}
	if out.Replacements != 1 {
		t.Fatalf("expected 1 replacement, got %d", out.Replacements)
	}

	got, err := s.medium.Read("code.go")
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	if got != "gamma beta alpha" {
		t.Fatalf("expected only first occurrence replaced, got %q", got)
	}
}

func TestFiles_editDiff_Good_ReplaceAll(t *testing.T) {
	s := newTmpService(t)
	ctx := context.Background()

	if err := s.medium.Write("code.go", "x y x z x"); err != nil {
		t.Fatalf("seed write: %v", err)
	}

	_, out, err := s.editDiff(ctx, nil, EditDiffInput{
		Path:       "code.go",
		OldString:  "x",
		NewString:  "Q",
		ReplaceAll: true,
	})
	if err != nil {
		t.Fatalf("editDiff: %v", err)
	}
	if out.Replacements != 3 {
		t.Fatalf("expected 3 replacements, got %d", out.Replacements)
	}
	got, _ := s.medium.Read("code.go")
	if got != "Q y Q z Q" {
		t.Fatalf("unexpected content %q", got)
	}
}

func TestFiles_editDiff_Bad_EmptyOldString(t *testing.T) {
	s := newTmpService(t)
	if err := s.medium.Write("code.go", "data"); err != nil {
		t.Fatalf("seed write: %v", err)
	}
	if _, _, err := s.editDiff(context.Background(), nil, EditDiffInput{
		Path:      "code.go",
		OldString: "",
		NewString: "x",
	}); err == nil {
		t.Fatal("expected error for empty old_string")
	}
}

func TestFiles_editDiff_Ugly_NotFound(t *testing.T) {
	s := newTmpService(t)
	if err := s.medium.Write("code.go", "hello world"); err != nil {
		t.Fatalf("seed write: %v", err)
	}
	if _, _, err := s.editDiff(context.Background(), nil, EditDiffInput{
		Path:      "code.go",
		OldString: "absent",
		NewString: "x",
	}); err == nil {
		t.Fatal("expected error when old_string is absent")
	}
}

func TestFiles_detectLanguage_Good(t *testing.T) {
	s := newTmpService(t)
	cases := map[string]string{
		"main.go":           "go",
		"app.ts":            "typescript",
		"index.tsx":         "typescript",
		"server.py":         "python",
		"lib.rs":            "rust",
		"Dockerfile":        "dockerfile",
		"deploy/Dockerfile": "dockerfile",
		"notes.md":          "markdown",
		"unknown.zzz":       "plaintext",
		"noext":             "plaintext",
	}
	for path, want := range cases {
		_, out, err := s.detectLanguage(context.Background(), nil, DetectLanguageInput{Path: path})
		if err != nil {
			t.Fatalf("detectLanguage(%q): %v", path, err)
		}
		if out.Language != want {
			t.Fatalf("detectLanguage(%q) = %q, want %q", path, out.Language, want)
		}
	}
}

func TestFiles_getSupportedLanguages_Good(t *testing.T) {
	s := newTmpService(t)
	_, out, err := s.getSupportedLanguages(context.Background(), nil, GetSupportedLanguagesInput{})
	if err != nil {
		t.Fatalf("getSupportedLanguages: %v", err)
	}
	if len(out.Languages) == 0 {
		t.Fatal("expected non-empty supported languages")
	}
	// "go" must be present with the .go extension.
	found := false
	for _, l := range out.Languages {
		if l.ID == "go" {
			found = true
			if len(l.Extensions) == 0 || l.Extensions[0] != ".go" {
				t.Fatalf("go language missing .go extension: %+v", l)
			}
		}
	}
	if !found {
		t.Fatal("expected go language in supported set")
	}
}

func TestFiles_countOccurrences_Good_Bad(t *testing.T) {
	if n := countOccurrences("aaa", "a"); n != 3 {
		t.Fatalf("countOccurrences(aaa,a) = %d, want 3", n)
	}
	if n := countOccurrences("ababab", "ab"); n != 3 {
		t.Fatalf("countOccurrences(ababab,ab) = %d, want 3", n)
	}
	// Empty substring returns 0 by contract.
	if n := countOccurrences("anything", ""); n != 0 {
		t.Fatalf("countOccurrences with empty substr = %d, want 0", n)
	}
	if n := countOccurrences("abc", "z"); n != 0 {
		t.Fatalf("countOccurrences(abc,z) = %d, want 0", n)
	}
}

func TestFiles_replaceFirst_Good(t *testing.T) {
	if got := replaceFirst("alpha beta alpha", "alpha", "X"); got != "X beta alpha" {
		t.Fatalf("replaceFirst = %q", got)
	}
	// No match returns the original string.
	if got := replaceFirst("hello", "zzz", "Q"); got != "hello" {
		t.Fatalf("replaceFirst no-match = %q", got)
	}
}
