package mcp

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestMcp_New_Good_DefaultWorkspace(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	s, err := New(Options{})
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	if s.workspaceRoot != cwd {
		t.Errorf("Expected default workspace root %s, got %s", cwd, s.workspaceRoot)
	}
	if s.medium == nil {
		t.Error("Expected medium to be set")
	}
}

func TestMcp_New_Good_CustomWorkspace(t *testing.T) {
	tmpDir := t.TempDir()

	s, err := New(Options{WorkspaceRoot: tmpDir})
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	if s.workspaceRoot != tmpDir {
		t.Errorf("Expected workspace root %s, got %s", tmpDir, s.workspaceRoot)
	}
	if s.medium == nil {
		t.Error("Expected medium to be set")
	}
}

func TestMcp_New_Good_NoRestriction(t *testing.T) {
	s, err := New(Options{Unrestricted: true})
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	if s.workspaceRoot != "" {
		t.Errorf("Expected empty workspace root, got %s", s.workspaceRoot)
	}
	if s.medium == nil {
		t.Error("Expected medium to be set (unsandboxed)")
	}
}

func TestMcp_New_Good_RegistersBuiltInTools(t *testing.T) {
	s, err := New(Options{})
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	tools := map[string]bool{}
	for _, rec := range s.Tools() {
		tools[rec.Name] = true
	}

	for _, name := range []string{
		"metrics_record",
		"metrics_query",
		"rag_query",
		"rag_ingest",
		"rag_collections",
		"webview_connect",
		"webview_disconnect",
		"webview_navigate",
		"webview_click",
		"webview_type",
		"webview_query",
		"webview_console",
		"webview_eval",
		"webview_screenshot",
		"webview_wait",
	} {
		if !tools[name] {
			t.Fatalf("expected tool %q to be registered", name)
		}
	}

	for _, name := range []string{"process_start", "ws_start"} {
		if tools[name] {
			t.Fatalf("did not expect tool %q to be registered without dependencies", name)
		}
	}
}

func TestMcp_New_Bad_NilSubsystemIgnored(t *testing.T) {
	s, err := New(Options{Subsystems: []Subsystem{nil}})
	if err != nil {
		t.Fatalf("New failed with nil subsystem: %v", err)
	}
	if len(s.Subsystems()) != 0 {
		t.Fatalf("expected nil subsystem to be ignored, got %d subsystems", len(s.Subsystems()))
	}
}

func TestMcp_New_Ugly_ConcurrentConstruction(t *testing.T) {
	tmpDir := t.TempDir()
	const workers = 8

	var wg sync.WaitGroup
	errs := make(chan error, workers)
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s, err := New(Options{WorkspaceRoot: tmpDir})
			if err != nil {
				errs <- err
				return
			}
			if s.workspaceRoot != tmpDir || s.medium == nil {
				errs <- os.ErrInvalid
			}
		}()
	}
	wg.Wait()
	close(errs)

	for err := range errs {
		if err != nil {
			t.Fatalf("concurrent New failed: %v", err)
		}
	}
}

func TestMcp_GetSupportedLanguages_Good_IncludesAllDetectedLanguages(t *testing.T) {
	s, err := New(Options{})
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	_, out, err := s.getSupportedLanguages(nil, nil, GetSupportedLanguagesInput{})
	if err != nil {
		t.Fatalf("getSupportedLanguages failed: %v", err)
	}

	if got, want := len(out.Languages), 23; got != want {
		t.Fatalf("expected %d supported languages, got %d", want, got)
	}

	got := map[string]bool{}
	for _, lang := range out.Languages {
		got[lang.ID] = true
	}

	for _, want := range []string{
		"typescript",
		"javascript",
		"go",
		"python",
		"rust",
		"ruby",
		"java",
		"php",
		"c",
		"cpp",
		"csharp",
		"html",
		"css",
		"scss",
		"json",
		"yaml",
		"xml",
		"markdown",
		"sql",
		"shell",
		"swift",
		"kotlin",
		"dockerfile",
	} {
		if !got[want] {
			t.Fatalf("expected language %q to be listed", want)
		}
	}
}

func TestMcp_GetSupportedLanguages_Bad_IgnoresUnsupportedInputState(t *testing.T) {
	s := &Service{}

	_, out, err := s.getSupportedLanguages(nil, nil, GetSupportedLanguagesInput{})
	if err != nil {
		t.Fatalf("getSupportedLanguages failed without initialized service state: %v", err)
	}
	if len(out.Languages) == 0 {
		t.Fatal("expected supported languages to be returned")
	}
}

func TestMcp_GetSupportedLanguages_Ugly_ReturnsIndependentSnapshots(t *testing.T) {
	s, err := New(Options{})
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	_, first, err := s.getSupportedLanguages(nil, nil, GetSupportedLanguagesInput{})
	if err != nil {
		t.Fatalf("getSupportedLanguages failed: %v", err)
	}
	first.Languages[0].ID = "mutated"

	_, second, err := s.getSupportedLanguages(nil, nil, GetSupportedLanguagesInput{})
	if err != nil {
		t.Fatalf("getSupportedLanguages failed on second call: %v", err)
	}
	if second.Languages[0].ID == "mutated" {
		t.Fatal("expected a fresh supported languages snapshot")
	}
}

func TestMcp_DetectLanguageFromPath_Good_KnownExtensions(t *testing.T) {
	cases := map[string]string{
		"main.go":           "go",
		"index.tsx":         "typescript",
		"style.scss":        "scss",
		"Program.cs":        "csharp",
		"module.kt":         "kotlin",
		"docker/Dockerfile": "dockerfile",
	}

	for path, want := range cases {
		if got := detectLanguageFromPath(path); got != want {
			t.Fatalf("detectLanguageFromPath(%q) = %q, want %q", path, got, want)
		}
	}
}

func TestMcp_DetectLanguageFromPath_Bad_UnsupportedExtensionDefaultsPlaintext(t *testing.T) {
	if got := detectLanguageFromPath("archive.unknown"); got != "plaintext" {
		t.Fatalf("expected unsupported extension to be plaintext, got %q", got)
	}
}

func TestMcp_DetectLanguageFromPath_Ugly_BoundaryPaths(t *testing.T) {
	cases := map[string]string{
		"":                 "plaintext",
		"Dockerfile":       "dockerfile",
		"nested/Makefile":  "plaintext",
		"nested/file.TSX":  "plaintext",
		"nested/.env":      "plaintext",
		"nested/file.bash": "shell",
	}

	for path, want := range cases {
		if got := detectLanguageFromPath(path); got != want {
			t.Fatalf("detectLanguageFromPath(%q) = %q, want %q", path, got, want)
		}
	}
}

func TestMcp_Medium_Good_ReadWrite(t *testing.T) {
	tmpDir := t.TempDir()
	s, err := New(Options{WorkspaceRoot: tmpDir})
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	// Write a file
	testContent := "hello world"
	err = s.medium.Write("test.txt", testContent)
	if err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}

	// Read it back
	content, err := s.medium.Read("test.txt")
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}
	if content != testContent {
		t.Errorf("Expected content %q, got %q", testContent, content)
	}

	// Verify file exists on disk
	diskPath := filepath.Join(tmpDir, "test.txt")
	if _, err := os.Stat(diskPath); os.IsNotExist(err) {
		t.Error("File should exist on disk")
	}
}

func TestMcp_Medium_Bad_ReadMissingFile(t *testing.T) {
	tmpDir := t.TempDir()
	s, err := New(Options{WorkspaceRoot: tmpDir})
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	if _, err := s.medium.Read("missing.txt"); err == nil {
		t.Fatal("expected reading a missing file to fail")
	}
}

func TestMcp_Medium_Ugly_ConcurrentReadWrite(t *testing.T) {
	tmpDir := t.TempDir()
	s, err := New(Options{WorkspaceRoot: tmpDir})
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	const workers = 8
	var wg sync.WaitGroup
	errs := make(chan error, workers)
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			path := filepath.Join("concurrent", string(rune('a'+i))+".txt")
			if err := s.medium.Write(path, "content"); err != nil {
				errs <- err
				return
			}
			if _, err := s.medium.Read(path); err != nil {
				errs <- err
			}
		}(i)
	}
	wg.Wait()
	close(errs)

	for err := range errs {
		if err != nil {
			t.Fatalf("concurrent medium access failed: %v", err)
		}
	}
}

func TestMcp_Medium_Good_EnsureDir(t *testing.T) {
	tmpDir := t.TempDir()
	s, err := New(Options{WorkspaceRoot: tmpDir})
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	err = s.medium.EnsureDir("subdir/nested")
	if err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}

	// Verify directory exists
	diskPath := filepath.Join(tmpDir, "subdir", "nested")
	info, err := os.Stat(diskPath)
	if os.IsNotExist(err) {
		t.Error("Directory should exist on disk")
	}
	if err == nil && !info.IsDir() {
		t.Error("Path should be a directory")
	}
}

func TestMcp_Medium_Bad_EnsureDirOverFile(t *testing.T) {
	tmpDir := t.TempDir()
	s, err := New(Options{WorkspaceRoot: tmpDir})
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}
	if err := s.medium.Write("same", "content"); err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}

	if err := s.medium.EnsureDir("same"); err == nil {
		t.Fatal("expected EnsureDir over an existing file to fail")
	}
}

func TestMcp_Medium_Ugly_EnsureDirIdempotentNestedBoundary(t *testing.T) {
	tmpDir := t.TempDir()
	s, err := New(Options{WorkspaceRoot: tmpDir})
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	for i := 0; i < 2; i++ {
		if err := s.medium.EnsureDir("subdir/nested"); err != nil {
			t.Fatalf("EnsureDir call %d failed: %v", i+1, err)
		}
	}
}

func TestMcp_FileExists_Good_FileAndDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	s, err := New(Options{WorkspaceRoot: tmpDir})
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	if err := s.medium.EnsureDir("nested"); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}
	if err := s.medium.Write("nested/file.txt", "content"); err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}

	_, fileOut, err := s.fileExists(nil, nil, FileExistsInput{Path: "nested/file.txt"})
	if err != nil {
		t.Fatalf("fileExists(file) failed: %v", err)
	}
	if !fileOut.Exists {
		t.Fatal("expected file to exist")
	}
	if fileOut.IsDir {
		t.Fatal("expected file to not be reported as a directory")
	}

	_, dirOut, err := s.fileExists(nil, nil, FileExistsInput{Path: "nested"})
	if err != nil {
		t.Fatalf("fileExists(dir) failed: %v", err)
	}
	if !dirOut.Exists {
		t.Fatal("expected directory to exist")
	}
	if !dirOut.IsDir {
		t.Fatal("expected directory to be reported as a directory")
	}
}

func TestMcp_FileExists_Bad_MissingPath(t *testing.T) {
	tmpDir := t.TempDir()
	s, err := New(Options{WorkspaceRoot: tmpDir})
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	_, out, err := s.fileExists(nil, nil, FileExistsInput{Path: "missing.txt"})
	if err != nil {
		t.Fatalf("fileExists(missing) failed: %v", err)
	}
	if out.Exists || out.IsDir {
		t.Fatalf("expected missing path to be reported absent, got %+v", out)
	}
}

func TestMcp_FileExists_Ugly_NilMedium(t *testing.T) {
	s := &Service{}

	if _, _, err := s.fileExists(nil, nil, FileExistsInput{Path: "anything"}); err == nil {
		t.Fatal("expected fileExists to fail when medium is nil")
	}
}

func TestMcp_ListDirectory_Good_ReturnsDocumentedEntryPaths(t *testing.T) {
	tmpDir := t.TempDir()
	s, err := New(Options{WorkspaceRoot: tmpDir})
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	if err := s.medium.EnsureDir("nested"); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}
	if err := s.medium.Write("nested/file.txt", "content"); err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}

	_, out, err := s.listDirectory(nil, nil, ListDirectoryInput{Path: "nested"})
	if err != nil {
		t.Fatalf("listDirectory failed: %v", err)
	}
	if len(out.Entries) != 1 {
		t.Fatalf("expected one entry, got %d", len(out.Entries))
	}

	want := filepath.Join("nested", "file.txt")
	if out.Entries[0].Path != want {
		t.Fatalf("expected entry path %q, got %q", want, out.Entries[0].Path)
	}
}

func TestMcp_ListDirectory_Bad_MissingDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	s, err := New(Options{WorkspaceRoot: tmpDir})
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	if _, _, err := s.listDirectory(nil, nil, ListDirectoryInput{Path: "missing"}); err == nil {
		t.Fatal("expected listing a missing directory to fail")
	}
}

func TestMcp_ListDirectory_Ugly_SortsEntries(t *testing.T) {
	tmpDir := t.TempDir()
	s, err := New(Options{WorkspaceRoot: tmpDir})
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}
	for _, name := range []string{"b.txt", "a.txt", "c.txt"} {
		if err := s.medium.Write(filepath.Join("nested", name), "content"); err != nil {
			t.Fatalf("Failed to write %s: %v", name, err)
		}
	}

	_, out, err := s.listDirectory(nil, nil, ListDirectoryInput{Path: "nested"})
	if err != nil {
		t.Fatalf("listDirectory failed: %v", err)
	}
	if len(out.Entries) != 3 {
		t.Fatalf("expected three entries, got %d", len(out.Entries))
	}
	for i, want := range []string{"a.txt", "b.txt", "c.txt"} {
		if out.Entries[i].Name != want {
			t.Fatalf("entry %d = %q, want %q", i, out.Entries[i].Name, want)
		}
	}
}

func TestMcp_Medium_Good_IsFile(t *testing.T) {
	tmpDir := t.TempDir()
	s, err := New(Options{WorkspaceRoot: tmpDir})
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	// File doesn't exist yet
	if s.medium.IsFile("test.txt") {
		t.Error("File should not exist yet")
	}

	// Create the file
	_ = s.medium.Write("test.txt", "content")

	// Now it should exist
	if !s.medium.IsFile("test.txt") {
		t.Error("File should exist after write")
	}
}

func TestMcp_Medium_Bad_IsFileEmptyPath(t *testing.T) {
	tmpDir := t.TempDir()
	s, err := New(Options{WorkspaceRoot: tmpDir})
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	if s.medium.IsFile("") {
		t.Fatal("empty path should not be a file")
	}
}

func TestMcp_Medium_Ugly_IsFileDirectoryBoundary(t *testing.T) {
	tmpDir := t.TempDir()
	s, err := New(Options{WorkspaceRoot: tmpDir})
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}
	if err := s.medium.EnsureDir("nested"); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}

	if s.medium.IsFile("nested") {
		t.Fatal("directory should not be reported as a file")
	}
}

func TestMcp_ResolveWorkspacePath_Good(t *testing.T) {
	tmpDir := t.TempDir()
	s, err := New(Options{WorkspaceRoot: tmpDir})
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	cases := map[string]string{
		"docs/readme.md":     filepath.Join(tmpDir, "docs", "readme.md"),
		"/docs/readme.md":    filepath.Join(tmpDir, "docs", "readme.md"),
		"../escape/notes.md": filepath.Join(tmpDir, "escape", "notes.md"),
		"":                   "",
	}
	for input, want := range cases {
		if got := s.resolveWorkspacePath(input); got != want {
			t.Fatalf("resolveWorkspacePath(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestMcp_ResolveWorkspacePath_Good_Unrestricted(t *testing.T) {
	s, err := New(Options{Unrestricted: true})
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	if got, want := s.resolveWorkspacePath("docs/readme.md"), filepath.Clean("docs/readme.md"); got != want {
		t.Fatalf("resolveWorkspacePath(relative) = %q, want %q", got, want)
	}
	if got, want := s.resolveWorkspacePath("/tmp/readme.md"), filepath.Clean("/tmp/readme.md"); got != want {
		t.Fatalf("resolveWorkspacePath(absolute) = %q, want %q", got, want)
	}
}

func TestMcp_ResolveWorkspacePath_Bad_EmptyPath(t *testing.T) {
	tmpDir := t.TempDir()
	s, err := New(Options{WorkspaceRoot: tmpDir})
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	if got := s.resolveWorkspacePath(""); got != "" {
		t.Fatalf("resolveWorkspacePath(empty) = %q, want empty", got)
	}
}

func TestMcp_ResolveWorkspacePath_Ugly_TraversalSanitized(t *testing.T) {
	tmpDir := t.TempDir()
	s, err := New(Options{WorkspaceRoot: tmpDir})
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	got := s.resolveWorkspacePath("../../secret.txt")
	want := filepath.Join(tmpDir, "secret.txt")
	if got != want {
		t.Fatalf("resolveWorkspacePath(traversal) = %q, want %q", got, want)
	}
}

func TestMcp_Medium_Ugly_TraversalSanitized(t *testing.T) {
	tmpDir := t.TempDir()
	s, err := New(Options{WorkspaceRoot: tmpDir})
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	// Path traversal is sanitized (.. becomes .), so ../secret.txt becomes
	// ./secret.txt in the workspace. Since that file doesn't exist, we get
	// a file not found error (not a traversal error).
	_, err = s.medium.Read("../secret.txt")
	if err == nil {
		t.Error("Expected error (file not found)")
	}

	// Absolute paths are allowed through - they access the real filesystem.
	// This is intentional for full filesystem access. Callers wanting sandboxing
	// should validate inputs before calling Medium.
}

func TestMcp_Medium_Ugly_SymlinksBlocked(t *testing.T) {
	tmpDir := t.TempDir()
	outsideDir := t.TempDir()

	// Create a target file outside workspace
	targetFile := filepath.Join(outsideDir, "secret.txt")
	if err := os.WriteFile(targetFile, []byte("secret"), 0644); err != nil {
		t.Fatalf("Failed to create target file: %v", err)
	}

	// Create symlink inside workspace pointing outside
	symlinkPath := filepath.Join(tmpDir, "link")
	if err := os.Symlink(targetFile, symlinkPath); err != nil {
		t.Skipf("Symlinks not supported: %v", err)
	}

	s, err := New(Options{WorkspaceRoot: tmpDir})
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	// Symlinks pointing outside the sandbox root are blocked (security feature).
	// The sandbox resolves the symlink target and rejects it because it escapes
	// the workspace boundary.
	_, err = s.medium.Read("link")
	if err == nil {
		t.Error("Expected permission denied for symlink escaping sandbox, but read succeeded")
	}
}
