package mcp

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMCP_New_Good(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	t.Run("default workspace", func(t *testing.T) {
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
	})

	t.Run("custom workspace", func(t *testing.T) {
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
	})

	t.Run("built in tools", func(t *testing.T) {
		s, err := New(Options{})
		if err != nil {
			t.Fatalf("Failed to create service: %v", err)
		}

		tools := map[string]bool{}
		for _, rec := range s.Tools() {
			tools[rec.Name] = true
		}

		for _, name := range []string{
			"file_read",
			"file_write",
			"file_delete",
			"file_rename",
			"file_exists",
			"file_edit",
			"dir_list",
			"dir_create",
			"lang_detect",
			"lang_list",
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
	})
}

func TestMCP_New_Bad(t *testing.T) {
	s, err := New(Options{Subsystems: []Subsystem{nil}})
	if err != nil {
		t.Fatalf("Failed to create service with nil subsystem entry: %v", err)
	}
	if got := len(s.Subsystems()); got != 0 {
		t.Fatalf("expected nil subsystem entry to be ignored, got %d subsystem(s)", got)
	}
}

func TestMCP_New_Ugly(t *testing.T) {
	t.Run("unrestricted ignores workspace root", func(t *testing.T) {
		tmpDir := t.TempDir()

		s, err := New(Options{WorkspaceRoot: tmpDir, Unrestricted: true})
		if err != nil {
			t.Fatalf("Failed to create unrestricted service: %v", err)
		}

		if s.workspaceRoot != "" {
			t.Errorf("Expected empty workspace root, got %s", s.workspaceRoot)
		}
		if s.medium == nil {
			t.Error("Expected medium to be set")
		}
	})

	t.Run("relative workspace root is made absolute", func(t *testing.T) {
		oldWD, err := os.Getwd()
		if err != nil {
			t.Fatalf("Failed to get working directory: %v", err)
		}
		tmpDir := t.TempDir()
		if err := os.Chdir(tmpDir); err != nil {
			t.Fatalf("Failed to chdir: %v", err)
		}
		t.Cleanup(func() {
			if err := os.Chdir(oldWD); err != nil {
				t.Fatalf("Failed to restore working directory: %v", err)
			}
		})

		s, err := New(Options{WorkspaceRoot: "."})
		if err != nil {
			t.Fatalf("Failed to create service with relative workspace: %v", err)
		}

		want, err := filepath.Abs(".")
		if err != nil {
			t.Fatalf("Failed to resolve expected workspace root: %v", err)
		}
		if s.workspaceRoot != want {
			t.Fatalf("expected relative workspace root %q to resolve to %q", s.workspaceRoot, want)
		}
	})
}

func TestMCP_GetSupportedLanguages_Good(t *testing.T) {
	s := newTestMCPService(t)

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

func TestMCP_GetSupportedLanguages_Bad(t *testing.T) {
	s := newTestMCPService(t)

	_, out, err := s.getSupportedLanguages(nil, nil, GetSupportedLanguagesInput{})
	if err != nil {
		t.Fatalf("getSupportedLanguages failed: %v", err)
	}

	ids := map[string]bool{}
	extensions := map[string]string{}
	for _, lang := range out.Languages {
		if lang.ID == "" {
			t.Fatal("supported language has empty ID")
		}
		if lang.Name == "" {
			t.Fatalf("supported language %q has empty display name", lang.ID)
		}
		if ids[lang.ID] {
			t.Fatalf("duplicate supported language ID %q", lang.ID)
		}
		ids[lang.ID] = true

		for _, ext := range lang.Extensions {
			if ext == "" {
				t.Fatalf("language %q has empty extension", lang.ID)
			}
			if ext[0] != '.' {
				t.Fatalf("language %q has extension %q without dot prefix", lang.ID, ext)
			}
			if got := languageByExtension[ext]; got != lang.ID {
				t.Fatalf("extension %q maps to %q, want %q", ext, got, lang.ID)
			}
			if owner, ok := extensions[ext]; ok {
				t.Fatalf("extension %q is registered for both %q and %q", ext, owner, lang.ID)
			}
			extensions[ext] = lang.ID
		}
	}
}

func TestMCP_GetSupportedLanguages_Ugly(t *testing.T) {
	s := newTestMCPService(t)

	_, out, err := s.getSupportedLanguages(nil, nil, GetSupportedLanguagesInput{})
	if err != nil {
		t.Fatalf("getSupportedLanguages failed: %v", err)
	}
	out.Languages[0].ID = "mutated"
	out.Languages[0].Extensions[0] = ".mutated"

	_, fresh, err := s.getSupportedLanguages(nil, nil, GetSupportedLanguagesInput{})
	if err != nil {
		t.Fatalf("getSupportedLanguages failed after caller mutation: %v", err)
	}
	if fresh.Languages[0].ID != "typescript" {
		t.Fatalf("caller mutation leaked into fresh language list: %q", fresh.Languages[0].ID)
	}
	if fresh.Languages[0].Extensions[0] != ".ts" {
		t.Fatalf("caller mutation leaked into fresh extension list: %q", fresh.Languages[0].Extensions[0])
	}
}

func TestMCP_DetectLanguageFromPath_Good(t *testing.T) {
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

func TestMCP_DetectLanguageFromPath_Bad(t *testing.T) {
	for _, path := range []string{"notes.unknown", "Makefile", "dockerfile"} {
		if got := detectLanguageFromPath(path); got != "plaintext" {
			t.Fatalf("detectLanguageFromPath(%q) = %q, want plaintext", path, got)
		}
	}
}

func TestMCP_DetectLanguageFromPath_Ugly(t *testing.T) {
	cases := map[string]string{
		"":                        "plaintext",
		".gitignore":              "plaintext",
		"archive.tar.gz":          "plaintext",
		"nested/.config/app.yaml": "yaml",
	}

	for path, want := range cases {
		if got := detectLanguageFromPath(path); got != want {
			t.Fatalf("detectLanguageFromPath(%q) = %q, want %q", path, got, want)
		}
	}
}

func TestMCP_Medium_Good(t *testing.T) {
	t.Run("read write", func(t *testing.T) {
		tmpDir := t.TempDir()
		s := newTestMCPServiceWithRoot(t, tmpDir)

		testContent := "hello world"
		if err := s.medium.Write("test.txt", testContent); err != nil {
			t.Fatalf("Failed to write file: %v", err)
		}

		content, err := s.medium.Read("test.txt")
		if err != nil {
			t.Fatalf("Failed to read file: %v", err)
		}
		if content != testContent {
			t.Errorf("Expected content %q, got %q", testContent, content)
		}

		diskPath := filepath.Join(tmpDir, "test.txt")
		if _, err := os.Stat(diskPath); os.IsNotExist(err) {
			t.Error("File should exist on disk")
		}
	})

	t.Run("ensure dir", func(t *testing.T) {
		tmpDir := t.TempDir()
		s := newTestMCPServiceWithRoot(t, tmpDir)

		if err := s.medium.EnsureDir("subdir/nested"); err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}

		diskPath := filepath.Join(tmpDir, "subdir", "nested")
		info, err := os.Stat(diskPath)
		if os.IsNotExist(err) {
			t.Error("Directory should exist on disk")
		}
		if err == nil && !info.IsDir() {
			t.Error("Path should be a directory")
		}
	})

	t.Run("is file", func(t *testing.T) {
		s := newTestMCPService(t)

		if s.medium.IsFile("test.txt") {
			t.Error("File should not exist yet")
		}

		if err := s.medium.Write("test.txt", "content"); err != nil {
			t.Fatalf("Failed to write file: %v", err)
		}

		if !s.medium.IsFile("test.txt") {
			t.Error("File should exist after write")
		}
	})
}

func TestMCP_Medium_Bad(t *testing.T) {
	t.Run("missing read", func(t *testing.T) {
		s := newTestMCPService(t)

		if _, err := s.medium.Read("missing.txt"); err == nil {
			t.Fatal("expected missing file read to fail")
		}
	})

	t.Run("file blocks directory creation", func(t *testing.T) {
		s := newTestMCPService(t)

		if err := s.medium.Write("already-file", "content"); err != nil {
			t.Fatalf("Failed to write file: %v", err)
		}
		if err := s.medium.EnsureDir("already-file"); err == nil {
			t.Fatal("expected directory creation over a file to fail")
		}
	})

	t.Run("symlink escape blocked", func(t *testing.T) {
		tmpDir := t.TempDir()
		outsideDir := t.TempDir()

		targetFile := filepath.Join(outsideDir, "secret.txt")
		if err := os.WriteFile(targetFile, []byte("secret"), 0644); err != nil {
			t.Fatalf("Failed to create target file: %v", err)
		}

		symlinkPath := filepath.Join(tmpDir, "link")
		if err := os.Symlink(targetFile, symlinkPath); err != nil {
			t.Skipf("Symlinks not supported: %v", err)
		}

		s := newTestMCPServiceWithRoot(t, tmpDir)

		if _, err := s.medium.Read("link"); err == nil {
			t.Error("Expected permission denied for symlink escaping sandbox, but read succeeded")
		}
	})
}

func TestMCP_Medium_Ugly(t *testing.T) {
	tmpDir := t.TempDir()
	s := newTestMCPServiceWithRoot(t, tmpDir)

	if err := s.medium.Write("../notes.txt", "inside workspace"); err != nil {
		t.Fatalf("Failed to write traversal path: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(tmpDir, "notes.txt"))
	if err != nil {
		t.Fatalf("expected traversal path to be sanitized inside workspace: %v", err)
	}
	if string(content) != "inside workspace" {
		t.Fatalf("expected sanitized traversal content, got %q", content)
	}
}

func TestMCP_FileExists_Good(t *testing.T) {
	s := newTestMCPService(t)

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

func TestMCP_FileExists_Bad(t *testing.T) {
	s := newTestMCPService(t)

	_, out, err := s.fileExists(nil, nil, FileExistsInput{Path: "missing.txt"})
	if err != nil {
		t.Fatalf("fileExists(missing) failed: %v", err)
	}
	if out.Exists {
		t.Fatal("expected missing file to not exist")
	}
	if out.IsDir {
		t.Fatal("expected missing file to not be reported as a directory")
	}
}

func TestMCP_FileExists_Ugly(t *testing.T) {
	s := newTestMCPService(t)

	_, out, err := s.fileExists(nil, nil, FileExistsInput{Path: ""})
	if err != nil {
		t.Fatalf("fileExists(empty path) failed: %v", err)
	}
	if !out.Exists {
		t.Fatal("expected empty path to resolve to existing workspace root")
	}
	if !out.IsDir {
		t.Fatal("expected empty path to report workspace root as a directory")
	}
}

func TestMCP_ListDirectory_Good(t *testing.T) {
	s := newTestMCPService(t)

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

func TestMCP_ListDirectory_Bad(t *testing.T) {
	s := newTestMCPService(t)

	if _, _, err := s.listDirectory(nil, nil, ListDirectoryInput{Path: "missing"}); err == nil {
		t.Fatal("expected missing directory list to fail")
	}
}

func TestMCP_ListDirectory_Ugly(t *testing.T) {
	s := newTestMCPService(t)

	if err := s.medium.Write("z.txt", "z"); err != nil {
		t.Fatalf("Failed to write z.txt: %v", err)
	}
	if err := s.medium.Write("a.txt", "a"); err != nil {
		t.Fatalf("Failed to write a.txt: %v", err)
	}
	if err := s.medium.EnsureDir("dir"); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}

	_, out, err := s.listDirectory(nil, nil, ListDirectoryInput{Path: ""})
	if err != nil {
		t.Fatalf("listDirectory(root) failed: %v", err)
	}

	got := make([]string, 0, len(out.Entries))
	for _, entry := range out.Entries {
		got = append(got, entry.Path)
	}
	want := []string{"a.txt", "dir", "z.txt"}
	if len(got) != len(want) {
		t.Fatalf("expected entries %v, got %v", want, got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("expected entries %v, got %v", want, got)
		}
	}
}

func TestMCP_ResolveWorkspacePath_Good(t *testing.T) {
	tmpDir := t.TempDir()
	s := newTestMCPServiceWithRoot(t, tmpDir)

	cases := map[string]string{
		"docs/readme.md":  filepath.Join(tmpDir, "docs", "readme.md"),
		"/docs/readme.md": filepath.Join(tmpDir, "docs", "readme.md"),
	}
	for input, want := range cases {
		if got := s.resolveWorkspacePath(input); got != want {
			t.Fatalf("resolveWorkspacePath(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestMCP_ResolveWorkspacePath_Bad(t *testing.T) {
	tmpDir := t.TempDir()
	s := newTestMCPServiceWithRoot(t, tmpDir)

	got := s.resolveWorkspacePath("../escape/notes.md")
	want := filepath.Join(tmpDir, "escape", "notes.md")
	if got != want {
		t.Fatalf("resolveWorkspacePath(traversal) = %q, want %q", got, want)
	}
}

func TestMCP_ResolveWorkspacePath_Ugly(t *testing.T) {
	t.Run("empty path", func(t *testing.T) {
		s := newTestMCPService(t)

		if got := s.resolveWorkspacePath(""); got != "" {
			t.Fatalf("resolveWorkspacePath(empty) = %q, want empty", got)
		}
	})

	t.Run("unrestricted", func(t *testing.T) {
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
	})
}

func newTestMCPService(t *testing.T) *Service {
	t.Helper()
	return newTestMCPServiceWithRoot(t, t.TempDir())
}

func newTestMCPServiceWithRoot(t *testing.T, root string) *Service {
	t.Helper()

	s, err := New(Options{WorkspaceRoot: root})
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}
	return s
}
