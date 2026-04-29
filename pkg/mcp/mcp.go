// SPDX-License-Identifier: EUPL-1.2

// Package mcp provides a lightweight MCP (Model Context Protocol) server for CLI use.
// For full GUI integration (display, webview, process management), see core-gui/pkg/mcp.
package mcp

import (
	"cmp"
	"context"
	"iter"
	"net/http"
	"slices"
	"sync"

	core "dappco.re/go"
	"dappco.re/go/process"
	"dappco.re/go/ws"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Service provides a lightweight MCP server with file operations and
// optional subsystems.
// For full GUI features, use the core-gui package.
//
//	svc, err := mcp.New(mcp.Options{WorkspaceRoot: "/home/user/project"})
//	defer svc.Shutdown(ctx)
type Service struct {
	*core.ServiceRuntime[struct{}] // Core access via s.Core()

	server         *mcp.Server
	workspaceRoot  string           // Root directory for file operations (empty = cwd unless Unrestricted)
	medium         *coreMedium      // Filesystem medium for sandboxed operations
	subsystems     []Subsystem      // Additional subsystems registered via Options.Subsystems
	logger         *core.Log        // Logger for tool execution auditing
	processService *process.Service // Process management service (optional)
	wsHub          *ws.Hub          // WebSocket hub for real-time streaming (optional)
	wsServer       *http.Server     // WebSocket HTTP server (optional)
	wsAddr         string           // WebSocket server address
	wsMu           sync.Mutex       // Protects wsServer and wsAddr
	processMu      sync.Mutex       // Protects processMeta
	processMeta    map[string]processRuntime
	tools          []ToolRecord // Parallel tool registry for REST bridge
}

// Options configures a Service.
//
//	svc, err := mcp.New(mcp.Options{
//	    WorkspaceRoot:  "/path/to/project",
//	    ProcessService: ps,
//	    Subsystems:     []Subsystem{brain, agentic, monitor},
//	})
type Options struct {
	WorkspaceRoot  string           // Restrict file ops to this directory (empty = cwd)
	Unrestricted   bool             // Disable sandboxing entirely (not recommended)
	ProcessService *process.Service // Optional process management
	WSHub          *ws.Hub          // Optional WebSocket hub for real-time streaming
	Subsystems     []Subsystem      // Additional tool groups registered at startup
}

// New creates a new MCP service with file operations and optional subsystems.
//
//	svc, err := mcp.New(mcp.Options{WorkspaceRoot: "."})
func New(opts Options) (
	*Service,
	error,
) {
	impl := &mcp.Implementation{
		Name:    "core-cli",
		Version: "0.1.0",
	}

	server := mcp.NewServer(impl, &mcp.ServerOptions{
		Capabilities: &mcp.ServerCapabilities{
			Resources:    &mcp.ResourceCapabilities{ListChanged: false},
			Tools:        &mcp.ToolCapabilities{ListChanged: false},
			Logging:      &mcp.LoggingCapabilities{},
			Experimental: channelCapability(),
		},
	})

	s := &Service{
		server:         server,
		processService: opts.ProcessService,
		wsHub:          opts.WSHub,
		logger:         core.Default(),
		processMeta:    make(map[string]processRuntime),
	}

	// Workspace root: unrestricted, explicit root, or default to cwd
	if opts.Unrestricted {
		s.workspaceRoot = ""
		s.medium = localMedium
	} else {
		root := opts.WorkspaceRoot
		if root == "" {
			cwd := core.Getwd()
			if !cwd.OK {
				err, _ := cwd.Value.(error)
				return nil, core.E("mcp.New", "failed to get working directory", err)
			}
			root = cwd.Value.(string)
		}
		abs := core.PathAbs(root)
		if !abs.OK {
			err, _ := abs.Value.(error)
			return nil, core.E("mcp.New", "failed to resolve workspace root", err)
		}
		s.workspaceRoot = abs.Value.(string)
		s.medium = newCoreMedium(s.workspaceRoot)
	}

	s.registerTools(s.server)

	s.subsystems = make([]Subsystem, 0, len(opts.Subsystems))
	for _, sub := range opts.Subsystems {
		if sub == nil {
			continue
		}
		s.subsystems = append(s.subsystems, sub)
		if sn, ok := sub.(SubsystemWithNotifier); ok {
			sn.SetNotifier(s)
		}
		// Wire channel callback for subsystems that use func-based notification.
		if cw, ok := sub.(SubsystemWithChannelCallback); ok {
			svc := s // capture for closure
			cw.OnChannel(func(ctx context.Context, channel string, data any) {
				svc.ChannelSend(ctx, channel, data)
			})
		}
		sub.RegisterTools(s)
	}

	return s, nil
}

// Subsystems returns the registered subsystems.
//
//	for _, sub := range svc.Subsystems() {
//	    fmt.Println(sub.Name())
//	}
func (s *Service) Subsystems() []Subsystem {
	return slices.Clone(s.subsystems)
}

// SubsystemsSeq returns an iterator over the registered subsystems.
//
//	for sub := range svc.SubsystemsSeq() {
//	    fmt.Println(sub.Name())
//	}
func (s *Service) SubsystemsSeq() iter.Seq[Subsystem] {
	return slices.Values(slices.Clone(s.subsystems))
}

// Tools returns all recorded tool metadata.
//
//	for _, t := range svc.Tools() {
//	    fmt.Printf("%s (%s): %s\n", t.Name, t.Group, t.Description)
//	}
func (s *Service) Tools() []ToolRecord {
	return slices.Clone(s.tools)
}

// ToolsSeq returns an iterator over all recorded tool metadata.
//
//	for rec := range svc.ToolsSeq() {
//	    fmt.Println(rec.Name)
//	}
func (s *Service) ToolsSeq() iter.Seq[ToolRecord] {
	return slices.Values(slices.Clone(s.tools))
}

// Shutdown gracefully shuts down all subsystems that support it.
//
//	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
//	defer cancel()
//	if err := svc.Shutdown(ctx); err != nil { core.Fatal(err) }
func (s *Service) Shutdown(
	ctx context.Context,
) (
	_ error, // result
) {
	var shutdownErr error

	for _, sub := range s.subsystems {
		if sh, ok := sub.(SubsystemWithShutdown); ok {
			if err := sh.Shutdown(ctx); err != nil {
				if shutdownErr == nil {
					shutdownErr = core.E("mcp.Shutdown", "shutdown "+sub.Name(), err)
				}
			}
		}
	}

	if s.wsServer != nil {
		s.wsMu.Lock()
		server := s.wsServer
		s.wsMu.Unlock()

		if err := server.Shutdown(ctx); err != nil && shutdownErr == nil {
			shutdownErr = core.E("mcp.Shutdown", "shutdown websocket server", err)
		}

		s.wsMu.Lock()
		if s.wsServer == server {
			s.wsServer = nil
			s.wsAddr = ""
		}
		s.wsMu.Unlock()
	}

	if err := closeWebviewConnection(); err != nil && shutdownErr == nil {
		shutdownErr = core.E("mcp.Shutdown", "close webview connection", err)
	}

	return shutdownErr
}

// WSHub returns the WebSocket hub, or nil if not configured.
//
//	if hub := svc.WSHub(); hub != nil {
//	    hub.SendProcessOutput("proc-1", "build complete")
//	}
func (s *Service) WSHub() *ws.Hub {
	return s.wsHub
}

// ProcessService returns the process service, or nil if not configured.
//
//	if ps := svc.ProcessService(); ps != nil {
//	    procs := ps.Running()
//	}
func (s *Service) ProcessService() *process.Service {
	return s.processService
}

// resolveWorkspacePath converts a tool path into the filesystem path the
// service actually operates on.
//
// Sandboxed services keep paths anchored under workspaceRoot. Unrestricted
// services preserve absolute paths and clean relative ones against the current
// working directory.
func (s *Service) resolveWorkspacePath(path string) string {
	if path == "" {
		return ""
	}

	if s.workspaceRoot == "" {
		return core.CleanPath(path, "/")
	}

	clean := core.CleanPath(string(core.PathSeparator)+path, "/")
	clean = core.TrimPrefix(clean, string(core.PathSeparator))
	if clean == "." || clean == "" {
		return s.workspaceRoot
	}
	return core.Path(s.workspaceRoot, clean)
}

// registerTools adds the built-in tool groups to the MCP server.
func (s *Service) registerTools(server *mcp.Server) {
	// File operations
	addToolRecorded(s, server, "files", &mcp.Tool{
		Name:        "file_read",
		Description: "Read the contents of a file",
	}, s.readFile)

	addToolRecorded(s, server, "files", &mcp.Tool{
		Name:        "file_write",
		Description: "Write content to a file",
	}, s.writeFile)

	addToolRecorded(s, server, "files", &mcp.Tool{
		Name:        "file_delete",
		Description: "Delete a file or empty directory",
	}, s.deleteFile)

	addToolRecorded(s, server, "files", &mcp.Tool{
		Name:        "file_rename",
		Description: "Rename or move a file",
	}, s.renameFile)

	addToolRecorded(s, server, "files", &mcp.Tool{
		Name:        "file_exists",
		Description: "Check if a file or directory exists",
	}, s.fileExists)

	addToolRecorded(s, server, "files", &mcp.Tool{
		Name:        "file_edit",
		Description: "Edit a file by replacing old_string with new_string. Use replace_all=true to replace all occurrences.",
	}, s.editDiff)

	// Directory operations
	addToolRecorded(s, server, "files", &mcp.Tool{
		Name:        "dir_list",
		Description: "List contents of a directory",
	}, s.listDirectory)

	addToolRecorded(s, server, "files", &mcp.Tool{
		Name:        "dir_create",
		Description: "Create a new directory",
	}, s.createDirectory)

	// Language detection
	addToolRecorded(s, server, "language", &mcp.Tool{
		Name:        "lang_detect",
		Description: "Detect the programming language of a file",
	}, s.detectLanguage)

	addToolRecorded(s, server, "language", &mcp.Tool{
		Name:        "lang_list",
		Description: "Get list of supported programming languages",
	}, s.getSupportedLanguages)

	// Additional built-in tool groups.
	s.registerMetricsTools(server)
	s.registerRAGTools(server)
	s.registerProcessTools(server)
	s.registerWebviewTools(server)
	s.registerWSTools(server)
	s.registerWSClientTools(server)
}

// Tool input/output types for MCP file operations.

// ReadFileInput contains parameters for reading a file.
//
//	input := ReadFileInput{Path: "src/main.go"}
type ReadFileInput struct {
	Path string "json:\"path\"" // e.g. "src/main.go"
}

// ReadFileOutput contains the result of reading a file.
//
//	// Returned by the file_read tool:
//	// out.Content == "package main\n..."
//	// out.Language == "go"
//	// out.Path == "src/main.go"
type ReadFileOutput struct {
	Content  string `json:"content"`  // e.g. "package main\n..."
	Language string `json:"language"` // e.g. "go"
	Path     string "json:\"path\""   // e.g. "src/main.go"
}

// WriteFileInput contains parameters for writing a file.
//
//	input := WriteFileInput{Path: "config/app.yaml", Content: "port: 8080\n"}
type WriteFileInput struct {
	Path    string "json:\"path\""  // e.g. "config/app.yaml"
	Content string `json:"content"` // e.g. "port: 8080\n"
}

// WriteFileOutput contains the result of writing a file.
//
//	// out.Success == true, out.Path == "config/app.yaml"
type WriteFileOutput struct {
	Success bool   `json:"success"` // true when the write succeeded
	Path    string "json:\"path\""  // e.g. "config/app.yaml"
}

// ListDirectoryInput contains parameters for listing a directory.
//
//	input := ListDirectoryInput{Path: "src/"}
type ListDirectoryInput struct {
	Path string "json:\"path\"" // e.g. "src/"
}

// ListDirectoryOutput contains the result of listing a directory.
//
//	// out.Path == "src/", len(out.Entries) == 3
type ListDirectoryOutput struct {
	Entries []DirectoryEntry `json:"entries"` // one entry per file/subdirectory
	Path    string           "json:\"path\""  // e.g. "src/"
}

// DirectoryEntry represents a single entry in a directory listing.
//
//	// entry.Name == "main.go", entry.IsDir == false, entry.Size == 1024
type DirectoryEntry struct {
	Name  string `json:"name"`   // e.g. "main.go"
	Path  string "json:\"path\"" // e.g. "src/main.go"
	IsDir bool   `json:"isDir"`  // true for directories
	Size  int64  `json:"size"`   // file size in bytes
}

// CreateDirectoryInput contains parameters for creating a directory.
//
//	input := CreateDirectoryInput{Path: "src/handlers"}
type CreateDirectoryInput struct {
	Path string "json:\"path\"" // e.g. "src/handlers"
}

// CreateDirectoryOutput contains the result of creating a directory.
//
//	// out.Success == true, out.Path == "src/handlers"
type CreateDirectoryOutput struct {
	Success bool   `json:"success"` // true when creation succeeded
	Path    string "json:\"path\""  // e.g. "src/handlers"
}

// DeleteFileInput contains parameters for deleting a file.
//
//	input := DeleteFileInput{Path: "tmp/debug.log"}
type DeleteFileInput struct {
	Path string "json:\"path\"" // e.g. "tmp/debug.log"
}

// DeleteFileOutput contains the result of deleting a file.
//
//	// out.Success == true, out.Path == "tmp/debug.log"
type DeleteFileOutput struct {
	Success bool   `json:"success"` // true when deletion succeeded
	Path    string "json:\"path\""  // e.g. "tmp/debug.log"
}

// RenameFileInput contains parameters for renaming a file.
//
//	input := RenameFileInput{OldPath: "pkg/util.go", NewPath: "pkg/helpers.go"}
type RenameFileInput struct {
	OldPath string `json:"oldPath"` // e.g. "pkg/util.go"
	NewPath string `json:"newPath"` // e.g. "pkg/helpers.go"
}

// RenameFileOutput contains the result of renaming a file.
//
//	// out.Success == true, out.OldPath == "pkg/util.go", out.NewPath == "pkg/helpers.go"
type RenameFileOutput struct {
	Success bool   `json:"success"` // true when rename succeeded
	OldPath string `json:"oldPath"` // e.g. "pkg/util.go"
	NewPath string `json:"newPath"` // e.g. "pkg/helpers.go"
}

// FileExistsInput contains parameters for checking file existence.
//
//	input := FileExistsInput{Path: "go.mod"}
type FileExistsInput struct {
	Path string "json:\"path\"" // e.g. "go.mod"
}

// FileExistsOutput contains the result of checking file existence.
//
//	// out.Exists == true, out.IsDir == false, out.Path == "go.mod"
type FileExistsOutput struct {
	Exists bool   `json:"exists"` // true when the path exists
	IsDir  bool   `json:"isDir"`  // true when the path is a directory
	Path   string "json:\"path\"" // e.g. "go.mod"
}

// DetectLanguageInput contains parameters for detecting file language.
//
//	input := DetectLanguageInput{Path: "cmd/server/main.go"}
type DetectLanguageInput struct {
	Path string "json:\"path\"" // e.g. "cmd/server/main.go"
}

// DetectLanguageOutput contains the detected programming language.
//
//	// out.Language == "go", out.Path == "cmd/server/main.go"
type DetectLanguageOutput struct {
	Language string `json:"language"` // e.g. "go", "typescript", "python"
	Path     string "json:\"path\""   // e.g. "cmd/server/main.go"
}

// GetSupportedLanguagesInput takes no parameters.
//
//	input := GetSupportedLanguagesInput{}
type GetSupportedLanguagesInput struct{}

// GetSupportedLanguagesOutput contains the list of supported languages.
//
//	// len(out.Languages) == 23
//	// out.Languages[0].ID == "typescript"
type GetSupportedLanguagesOutput struct {
	Languages []LanguageInfo `json:"languages"` // all recognised languages
}

// LanguageInfo describes a supported programming language.
//
//	// info.ID == "go", info.Name == "Go", info.Extensions == [".go"]
type LanguageInfo struct {
	ID         string   `json:"id"`         // e.g. "go"
	Name       string   `json:"name"`       // e.g. "Go"
	Extensions []string `json:"extensions"` // e.g. [".go"]
}

// EditDiffInput contains parameters for editing a file via string replacement.
//
//	input := EditDiffInput{
//	    Path:      "main.go",
//	    OldString: "fmt.Println(\"hello\")",
//	    NewString: "fmt.Println(\"world\")",
//	}
type EditDiffInput struct {
	Path       string "json:\"path\""                // e.g. "main.go"
	OldString  string `json:"old_string"`            // text to find
	NewString  string `json:"new_string"`            // replacement text
	ReplaceAll bool   `json:"replace_all,omitempty"` // replace all occurrences (default: first only)
}

// EditDiffOutput contains the result of a diff-based edit operation.
//
//	// out.Success == true, out.Replacements == 1, out.Path == "main.go"
type EditDiffOutput struct {
	Path         string "json:\"path\""       // e.g. "main.go"
	Success      bool   `json:"success"`      // true when at least one replacement was made
	Replacements int    `json:"replacements"` // number of replacements performed
}

// Tool handlers

func (s *Service) readFile(ctx context.Context, req *mcp.CallToolRequest, input ReadFileInput) (
	*mcp.CallToolResult,
	ReadFileOutput,
	error,
) {
	if s.medium == nil {
		return nil, ReadFileOutput{}, core.E("mcp.readFile", "workspace medium unavailable", nil)
	}

	content, err := s.medium.Read(input.Path)
	if err != nil {
		return nil, ReadFileOutput{}, core.E("mcp.readFile", "failed to read file", err)
	}
	return nil, ReadFileOutput{
		Content:  content,
		Language: detectLanguageFromPath(input.Path),
		Path:     input.Path,
	}, nil
}

func (s *Service) writeFile(ctx context.Context, req *mcp.CallToolRequest, input WriteFileInput) (
	*mcp.CallToolResult,
	WriteFileOutput,
	error,
) {
	if s.medium == nil {
		return nil, WriteFileOutput{}, core.E("mcp.writeFile", "workspace medium unavailable", nil)
	}

	// Medium.Write creates parent directories automatically
	if err := s.medium.Write(input.Path, input.Content); err != nil {
		return nil, WriteFileOutput{}, core.E("mcp.writeFile", "failed to write file", err)
	}
	return nil, WriteFileOutput{Success: true, Path: input.Path}, nil
}

func (s *Service) listDirectory(ctx context.Context, req *mcp.CallToolRequest, input ListDirectoryInput) (
	*mcp.CallToolResult,
	ListDirectoryOutput,
	error,
) {
	if s.medium == nil {
		return nil, ListDirectoryOutput{}, core.E("mcp.listDirectory", "workspace medium unavailable", nil)
	}

	entries, err := s.medium.List(input.Path)
	if err != nil {
		return nil, ListDirectoryOutput{}, core.E("mcp.listDirectory", "failed to list directory", err)
	}
	slices.SortFunc(entries, func(a, b core.FsDirEntry) int {
		return cmp.Compare(a.Name(), b.Name())
	})
	result := make([]DirectoryEntry, 0, len(entries))
	for _, e := range entries {
		info, _ := e.Info()
		var size int64
		if info != nil {
			size = info.Size()
		}
		result = append(result, DirectoryEntry{
			Name:  e.Name(),
			Path:  directoryEntryPath(input.Path, e.Name()),
			IsDir: e.IsDir(),
			Size:  size,
		})
	}
	return nil, ListDirectoryOutput{Entries: result, Path: input.Path}, nil
}

// directoryEntryPath returns the documented display path for a directory entry.
//
// Example:
//
//	directoryEntryPath("src", "main.go") == "src/main.go"
func directoryEntryPath(dir, name string) string {
	if dir == "" {
		return name
	}
	return core.JoinPath(dir, name)
}

func (s *Service) createDirectory(ctx context.Context, req *mcp.CallToolRequest, input CreateDirectoryInput) (
	*mcp.CallToolResult,
	CreateDirectoryOutput,
	error,
) {
	if s.medium == nil {
		return nil, CreateDirectoryOutput{}, core.E("mcp.createDirectory", "workspace medium unavailable", nil)
	}

	if err := s.medium.EnsureDir(input.Path); err != nil {
		return nil, CreateDirectoryOutput{}, core.E("mcp.createDirectory", "failed to create directory", err)
	}
	return nil, CreateDirectoryOutput{Success: true, Path: input.Path}, nil
}

func (s *Service) deleteFile(ctx context.Context, req *mcp.CallToolRequest, input DeleteFileInput) (
	*mcp.CallToolResult,
	DeleteFileOutput,
	error,
) {
	if s.medium == nil {
		return nil, DeleteFileOutput{}, core.E("mcp.deleteFile", "workspace medium unavailable", nil)
	}

	if err := s.medium.Delete(input.Path); err != nil {
		return nil, DeleteFileOutput{}, core.E("mcp.deleteFile", "failed to delete file", err)
	}
	return nil, DeleteFileOutput{Success: true, Path: input.Path}, nil
}

func (s *Service) renameFile(ctx context.Context, req *mcp.CallToolRequest, input RenameFileInput) (
	*mcp.CallToolResult,
	RenameFileOutput,
	error,
) {
	if s.medium == nil {
		return nil, RenameFileOutput{}, core.E("mcp.renameFile", "workspace medium unavailable", nil)
	}

	if err := s.medium.Rename(input.OldPath, input.NewPath); err != nil {
		return nil, RenameFileOutput{}, core.E("mcp.renameFile", "failed to rename file", err)
	}
	return nil, RenameFileOutput{Success: true, OldPath: input.OldPath, NewPath: input.NewPath}, nil
}

func (s *Service) fileExists(ctx context.Context, req *mcp.CallToolRequest, input FileExistsInput) (
	*mcp.CallToolResult,
	FileExistsOutput,
	error,
) {
	if s.medium == nil {
		return nil, FileExistsOutput{}, core.E("mcp.fileExists", "workspace medium unavailable", nil)
	}

	info, err := s.medium.Stat(input.Path)
	if err != nil {
		if core.IsNotExist(err) {
			return nil, FileExistsOutput{Exists: false, IsDir: false, Path: input.Path}, nil
		}
		return nil, FileExistsOutput{}, core.E("mcp.fileExists", "failed to stat path", err)
	}
	return nil, FileExistsOutput{
		Exists: true,
		IsDir:  info.IsDir(),
		Path:   input.Path,
	}, nil
}

func (s *Service) detectLanguage(ctx context.Context, req *mcp.CallToolRequest, input DetectLanguageInput) (
	*mcp.CallToolResult,
	DetectLanguageOutput,
	error,
) {
	lang := detectLanguageFromPath(input.Path)
	return nil, DetectLanguageOutput{Language: lang, Path: input.Path}, nil
}

func (s *Service) getSupportedLanguages(ctx context.Context, req *mcp.CallToolRequest, input GetSupportedLanguagesInput) (
	*mcp.CallToolResult,
	GetSupportedLanguagesOutput,
	error,
) {
	return nil, GetSupportedLanguagesOutput{Languages: supportedLanguages()}, nil
}

func (s *Service) editDiff(ctx context.Context, req *mcp.CallToolRequest, input EditDiffInput) (
	*mcp.CallToolResult,
	EditDiffOutput,
	error,
) {
	if s.medium == nil {
		return nil, EditDiffOutput{}, core.E("mcp.editDiff", "workspace medium unavailable", nil)
	}

	if input.OldString == "" {
		return nil, EditDiffOutput{}, core.E("mcp.editDiff", "old_string cannot be empty", nil)
	}

	content, err := s.medium.Read(input.Path)
	if err != nil {
		return nil, EditDiffOutput{}, core.E("mcp.editDiff", "failed to read file", err)
	}

	count := 0

	if input.ReplaceAll {
		count = countOccurrences(content, input.OldString)
		if count == 0 {
			return nil, EditDiffOutput{}, core.E("mcp.editDiff", "old_string not found in file", nil)
		}
		content = core.Replace(content, input.OldString, input.NewString)
	} else {
		if !core.Contains(content, input.OldString) {
			return nil, EditDiffOutput{}, core.E("mcp.editDiff", "old_string not found in file", nil)
		}
		content = replaceFirst(content, input.OldString, input.NewString)
		count = 1
	}

	if err := s.medium.Write(input.Path, content); err != nil {
		return nil, EditDiffOutput{}, core.E("mcp.editDiff", "failed to write file", err)
	}

	return nil, EditDiffOutput{
		Path:         input.Path,
		Success:      true,
		Replacements: count,
	}, nil
}

// detectLanguageFromPath maps file extensions to language IDs.
func detectLanguageFromPath(path string) string {
	if core.PathBase(path) == "Dockerfile" {
		return "dockerfile"
	}

	ext := core.PathExt(path)
	if lang, ok := languageByExtension[ext]; ok {
		return lang
	}
	return "plaintext"
}

var languageByExtension = map[string]string{
	".ts":       "typescript",
	".tsx":      "typescript",
	".js":       "javascript",
	".jsx":      "javascript",
	".go":       "go",
	".py":       "python",
	".rs":       "rust",
	".rb":       "ruby",
	".java":     "java",
	".php":      "php",
	".c":        "c",
	".h":        "c",
	".cpp":      "cpp",
	".hpp":      "cpp",
	".cc":       "cpp",
	".cxx":      "cpp",
	".cs":       "csharp",
	".html":     "html",
	".htm":      "html",
	".css":      "css",
	".scss":     "scss",
	".json":     `json`,
	".yaml":     "yaml",
	".yml":      "yaml",
	".xml":      "xml",
	".md":       "markdown",
	".markdown": "markdown",
	".sql":      "sql",
	".sh":       "shell",
	".bash":     "shell",
	".swift":    "swift",
	".kt":       "kotlin",
	".kts":      "kotlin",
}

func supportedLanguages() []LanguageInfo {
	return []LanguageInfo{
		{ID: "typescript", Name: "TypeScript", Extensions: []string{".ts", ".tsx"}},
		{ID: "javascript", Name: "JavaScript", Extensions: []string{".js", ".jsx"}},
		{ID: "go", Name: "Go", Extensions: []string{".go"}},
		{ID: "python", Name: "Python", Extensions: []string{".py"}},
		{ID: "rust", Name: "Rust", Extensions: []string{".rs"}},
		{ID: "ruby", Name: "Ruby", Extensions: []string{".rb"}},
		{ID: "java", Name: "Java", Extensions: []string{".java"}},
		{ID: "php", Name: "PHP", Extensions: []string{".php"}},
		{ID: "c", Name: "C", Extensions: []string{".c", ".h"}},
		{ID: "cpp", Name: "C++", Extensions: []string{".cpp", ".hpp", ".cc", ".cxx"}},
		{ID: "csharp", Name: "C#", Extensions: []string{".cs"}},
		{ID: "html", Name: "HTML", Extensions: []string{".html", ".htm"}},
		{ID: "css", Name: "CSS", Extensions: []string{".css"}},
		{ID: "scss", Name: "SCSS", Extensions: []string{".scss"}},
		{ID: `json`, Name: "JSON", Extensions: []string{".json"}},
		{ID: "yaml", Name: "YAML", Extensions: []string{".yaml", ".yml"}},
		{ID: "xml", Name: "XML", Extensions: []string{".xml"}},
		{ID: "markdown", Name: "Markdown", Extensions: []string{".md", ".markdown"}},
		{ID: "sql", Name: "SQL", Extensions: []string{".sql"}},
		{ID: "shell", Name: "Shell", Extensions: []string{".sh", ".bash"}},
		{ID: "swift", Name: "Swift", Extensions: []string{".swift"}},
		{ID: "kotlin", Name: "Kotlin", Extensions: []string{".kt", ".kts"}},
		{ID: "dockerfile", Name: "Dockerfile", Extensions: []string{}},
	}
}

// Run starts the MCP server, auto-selecting transport from environment.
//
//	// Stdio (default):
//	svc.Run(ctx)
//
//	// TCP (set MCP_ADDR):
//	os.Setenv("MCP_ADDR", "127.0.0.1:9100")
//	svc.Run(ctx)
//
//	// Unix socket (set MCP_UNIX_SOCKET):
//	os.Setenv("MCP_UNIX_SOCKET", "/tmp/core-mcp.sock")
//	svc.Run(ctx)
//
//	// HTTP (set MCP_HTTP_ADDR):
//	os.Setenv("MCP_HTTP_ADDR", "127.0.0.1:9101")
//	svc.Run(ctx)
func (s *Service) Run(
	ctx context.Context,
) (
	_ error, // result
) {
	if httpAddr := core.Env("MCP_HTTP_ADDR"); httpAddr != "" {
		return s.ServeHTTP(ctx, httpAddr)
	}
	if addr := core.Env("MCP_ADDR"); addr != "" {
		return s.ServeTCP(ctx, addr)
	}
	if socketPath := core.Env("MCP_UNIX_SOCKET"); socketPath != "" {
		return s.ServeUnix(ctx, socketPath)
	}
	return s.ServeStdio(ctx)
}

// countOccurrences counts non-overlapping instances of substr in s.
func countOccurrences(s, substr string) int {
	if substr == "" {
		return 0
	}
	count := 0
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			count++
			i += len(substr) - 1
		}
	}
	return count
}

// replaceFirst replaces the first occurrence of old with new in s.
func replaceFirst(s, old, new string) string {
	i := 0
	for i <= len(s)-len(old) {
		if s[i:i+len(old)] == old {
			return core.Concat(s[:i], new, s[i+len(old):])
		}
		i++
	}
	return s
}

// Server returns the underlying MCP server for advanced configuration.
//
//	server := svc.Server()
//	mcp.AddTool(server, &mcp.Tool{Name: "custom_tool"}, handler)
func (s *Service) Server() *mcp.Server {
	return s.server
}
