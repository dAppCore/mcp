// SPDX-License-Identifier: EUPL-1.2

// Package mcp provides a lightweight MCP (Model Context Protocol) server for CLI use.
// For full GUI integration (display, webview, process management), see core-gui/pkg/mcp.
package mcp

import (
	"context"
	"iter"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"sync"

	core "dappco.re/go/core"
	"forge.lthn.ai/core/go-io"
	"forge.lthn.ai/core/go-log"
	"forge.lthn.ai/core/go-process"
	"forge.lthn.ai/core/go-ws"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Service provides a lightweight MCP server with file operations only.
// For full GUI features, use the core-gui package.
//
//	svc, err := mcp.New(mcp.Options{WorkspaceRoot: "/home/user/project"})
//	defer svc.Shutdown(ctx)
type Service struct {
	*core.ServiceRuntime[McpOptions] // Core access via s.Core()

	server         *mcp.Server
	workspaceRoot  string           // Root directory for file operations (empty = unrestricted)
	medium         io.Medium        // Filesystem medium for sandboxed operations
	subsystems     []Subsystem      // Additional subsystems registered via Options.Subsystems
	logger         *log.Logger      // Logger for tool execution auditing
	processService *process.Service // Process management service (optional)
	wsHub          *ws.Hub          // WebSocket hub for real-time streaming (optional)
	wsServer       *http.Server     // WebSocket HTTP server (optional)
	wsAddr         string           // WebSocket server address
	wsMu           sync.Mutex       // Protects wsServer and wsAddr
	tools          []ToolRecord     // Parallel tool registry for REST bridge
}

// McpOptions configures the MCP service runtime.
type McpOptions struct{}

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

// New creates a new MCP service with file operations.
//
//	svc, err := mcp.New(mcp.Options{WorkspaceRoot: "."})
func New(opts Options) (*Service, error) {
	impl := &mcp.Implementation{
		Name:    "core-cli",
		Version: "0.1.0",
	}

	server := mcp.NewServer(impl, &mcp.ServerOptions{
		Capabilities: &mcp.ServerCapabilities{
			Tools:        &mcp.ToolCapabilities{ListChanged: true},
			Logging:      &mcp.LoggingCapabilities{},
			Experimental: channelCapability(),
		},
	})

	s := &Service{
		server:         server,
		processService: opts.ProcessService,
		wsHub:          opts.WSHub,
		subsystems:     opts.Subsystems,
		logger:         log.Default(),
	}

	// Workspace root: unrestricted, explicit root, or default to cwd
	if opts.Unrestricted {
		s.workspaceRoot = ""
		s.medium = io.Local
	} else {
		root := opts.WorkspaceRoot
		if root == "" {
			cwd, err := os.Getwd()
			if err != nil {
				return nil, core.E("mcp.New", "failed to get working directory", err)
			}
			root = cwd
		}
		abs, err := filepath.Abs(root)
		if err != nil {
			return nil, core.E("mcp.New", "failed to resolve workspace root", err)
		}
		s.workspaceRoot = abs
		m, merr := io.NewSandboxed(abs)
		if merr != nil {
			return nil, core.E("mcp.New", "failed to create workspace medium", merr)
		}
		s.medium = m
	}

	s.registerTools(s.server)

	for _, sub := range s.subsystems {
		sub.RegisterTools(s.server)
		if sn, ok := sub.(SubsystemWithNotifier); ok {
			sn.SetNotifier(s)
		}
		// Wire channel callback for subsystems that use func-based notification
		type channelWirer interface {
			OnChannel(func(ctx context.Context, channel string, data any))
		}
		if cw, ok := sub.(channelWirer); ok {
			svc := s // capture for closure
			cw.OnChannel(func(ctx context.Context, channel string, data any) {
				svc.ChannelSend(ctx, channel, data)
			})
		}
	}

	return s, nil
}

// Subsystems returns the registered subsystems.
//
//	for _, sub := range svc.Subsystems() {
//	    fmt.Println(sub.Name())
//	}
func (s *Service) Subsystems() []Subsystem {
	return s.subsystems
}

// SubsystemsSeq returns an iterator over the registered subsystems.
//
//	for sub := range svc.SubsystemsSeq() {
//	    fmt.Println(sub.Name())
//	}
func (s *Service) SubsystemsSeq() iter.Seq[Subsystem] {
	return slices.Values(s.subsystems)
}

// Tools returns all recorded tool metadata.
//
//	for _, t := range svc.Tools() {
//	    fmt.Printf("%s (%s): %s\n", t.Name, t.Group, t.Description)
//	}
func (s *Service) Tools() []ToolRecord {
	return s.tools
}

// ToolsSeq returns an iterator over all recorded tool metadata.
//
//	for rec := range svc.ToolsSeq() {
//	    fmt.Println(rec.Name)
//	}
func (s *Service) ToolsSeq() iter.Seq[ToolRecord] {
	return slices.Values(s.tools)
}

// Shutdown gracefully shuts down all subsystems that support it.
//
//	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
//	defer cancel()
//	if err := svc.Shutdown(ctx); err != nil { log.Fatal(err) }
func (s *Service) Shutdown(ctx context.Context) error {
	for _, sub := range s.subsystems {
		if sh, ok := sub.(SubsystemWithShutdown); ok {
			if err := sh.Shutdown(ctx); err != nil {
				return log.E("mcp.Shutdown", "shutdown "+sub.Name(), err)
			}
		}
	}
	return nil
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

// registerTools adds file operation tools to the MCP server.
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
}

// Tool input/output types for MCP file operations.

// ReadFileInput contains parameters for reading a file.
//
//	input := ReadFileInput{Path: "src/main.go"}
type ReadFileInput struct {
	Path string `json:"path"` // e.g. "src/main.go"
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
	Path     string `json:"path"`     // e.g. "src/main.go"
}

// WriteFileInput contains parameters for writing a file.
//
//	input := WriteFileInput{Path: "config/app.yaml", Content: "port: 8080\n"}
type WriteFileInput struct {
	Path    string `json:"path"`    // e.g. "config/app.yaml"
	Content string `json:"content"` // e.g. "port: 8080\n"
}

// WriteFileOutput contains the result of writing a file.
//
//	// out.Success == true, out.Path == "config/app.yaml"
type WriteFileOutput struct {
	Success bool   `json:"success"` // true when the write succeeded
	Path    string `json:"path"`    // e.g. "config/app.yaml"
}

// ListDirectoryInput contains parameters for listing a directory.
//
//	input := ListDirectoryInput{Path: "src/"}
type ListDirectoryInput struct {
	Path string `json:"path"` // e.g. "src/"
}

// ListDirectoryOutput contains the result of listing a directory.
//
//	// out.Path == "src/", len(out.Entries) == 3
type ListDirectoryOutput struct {
	Entries []DirectoryEntry `json:"entries"` // one entry per file/subdirectory
	Path    string           `json:"path"`    // e.g. "src/"
}

// DirectoryEntry represents a single entry in a directory listing.
//
//	// entry.Name == "main.go", entry.IsDir == false, entry.Size == 1024
type DirectoryEntry struct {
	Name  string `json:"name"`  // e.g. "main.go"
	Path  string `json:"path"`  // e.g. "src/main.go"
	IsDir bool   `json:"isDir"` // true for directories
	Size  int64  `json:"size"`  // file size in bytes
}

// CreateDirectoryInput contains parameters for creating a directory.
//
//	input := CreateDirectoryInput{Path: "src/handlers"}
type CreateDirectoryInput struct {
	Path string `json:"path"` // e.g. "src/handlers"
}

// CreateDirectoryOutput contains the result of creating a directory.
//
//	// out.Success == true, out.Path == "src/handlers"
type CreateDirectoryOutput struct {
	Success bool   `json:"success"` // true when creation succeeded
	Path    string `json:"path"`    // e.g. "src/handlers"
}

// DeleteFileInput contains parameters for deleting a file.
//
//	input := DeleteFileInput{Path: "tmp/debug.log"}
type DeleteFileInput struct {
	Path string `json:"path"` // e.g. "tmp/debug.log"
}

// DeleteFileOutput contains the result of deleting a file.
//
//	// out.Success == true, out.Path == "tmp/debug.log"
type DeleteFileOutput struct {
	Success bool   `json:"success"` // true when deletion succeeded
	Path    string `json:"path"`    // e.g. "tmp/debug.log"
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
	Path string `json:"path"` // e.g. "go.mod"
}

// FileExistsOutput contains the result of checking file existence.
//
//	// out.Exists == true, out.IsDir == false, out.Path == "go.mod"
type FileExistsOutput struct {
	Exists bool   `json:"exists"` // true when the path exists
	IsDir  bool   `json:"isDir"`  // true when the path is a directory
	Path   string `json:"path"`   // e.g. "go.mod"
}

// DetectLanguageInput contains parameters for detecting file language.
//
//	input := DetectLanguageInput{Path: "cmd/server/main.go"}
type DetectLanguageInput struct {
	Path string `json:"path"` // e.g. "cmd/server/main.go"
}

// DetectLanguageOutput contains the detected programming language.
//
//	// out.Language == "go", out.Path == "cmd/server/main.go"
type DetectLanguageOutput struct {
	Language string `json:"language"` // e.g. "go", "typescript", "python"
	Path     string `json:"path"`     // e.g. "cmd/server/main.go"
}

// GetSupportedLanguagesInput takes no parameters.
//
//	input := GetSupportedLanguagesInput{}
type GetSupportedLanguagesInput struct{}

// GetSupportedLanguagesOutput contains the list of supported languages.
//
//	// len(out.Languages) == 15
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
	Path       string `json:"path"`                  // e.g. "main.go"
	OldString  string `json:"old_string"`            // text to find
	NewString  string `json:"new_string"`            // replacement text
	ReplaceAll bool   `json:"replace_all,omitempty"` // replace all occurrences (default: first only)
}

// EditDiffOutput contains the result of a diff-based edit operation.
//
//	// out.Success == true, out.Replacements == 1, out.Path == "main.go"
type EditDiffOutput struct {
	Path         string `json:"path"`         // e.g. "main.go"
	Success      bool   `json:"success"`      // true when at least one replacement was made
	Replacements int    `json:"replacements"` // number of replacements performed
}

// Tool handlers

func (s *Service) readFile(ctx context.Context, req *mcp.CallToolRequest, input ReadFileInput) (*mcp.CallToolResult, ReadFileOutput, error) {
	content, err := s.medium.Read(input.Path)
	if err != nil {
		return nil, ReadFileOutput{}, log.E("mcp.readFile", "failed to read file", err)
	}
	return nil, ReadFileOutput{
		Content:  content,
		Language: detectLanguageFromPath(input.Path),
		Path:     input.Path,
	}, nil
}

func (s *Service) writeFile(ctx context.Context, req *mcp.CallToolRequest, input WriteFileInput) (*mcp.CallToolResult, WriteFileOutput, error) {
	// Medium.Write creates parent directories automatically
	if err := s.medium.Write(input.Path, input.Content); err != nil {
		return nil, WriteFileOutput{}, log.E("mcp.writeFile", "failed to write file", err)
	}
	return nil, WriteFileOutput{Success: true, Path: input.Path}, nil
}

func (s *Service) listDirectory(ctx context.Context, req *mcp.CallToolRequest, input ListDirectoryInput) (*mcp.CallToolResult, ListDirectoryOutput, error) {
	entries, err := s.medium.List(input.Path)
	if err != nil {
		return nil, ListDirectoryOutput{}, log.E("mcp.listDirectory", "failed to list directory", err)
	}
	result := make([]DirectoryEntry, 0, len(entries))
	for _, e := range entries {
		info, _ := e.Info()
		var size int64
		if info != nil {
			size = info.Size()
		}
		result = append(result, DirectoryEntry{
			Name: e.Name(),
			Path: core.JoinPath(input.Path, e.Name()), // Note: This might be relative path, client might expect absolute?
			// Issue 103 says "Replace ... with local.Medium sandboxing".
			// Previous code returned `core.JoinPath(input.Path, e.Name())`.
			// If input.Path is relative, this preserves it.
			IsDir: e.IsDir(),
			Size:  size,
		})
	}
	return nil, ListDirectoryOutput{Entries: result, Path: input.Path}, nil
}

func (s *Service) createDirectory(ctx context.Context, req *mcp.CallToolRequest, input CreateDirectoryInput) (*mcp.CallToolResult, CreateDirectoryOutput, error) {
	if err := s.medium.EnsureDir(input.Path); err != nil {
		return nil, CreateDirectoryOutput{}, log.E("mcp.createDirectory", "failed to create directory", err)
	}
	return nil, CreateDirectoryOutput{Success: true, Path: input.Path}, nil
}

func (s *Service) deleteFile(ctx context.Context, req *mcp.CallToolRequest, input DeleteFileInput) (*mcp.CallToolResult, DeleteFileOutput, error) {
	if err := s.medium.Delete(input.Path); err != nil {
		return nil, DeleteFileOutput{}, log.E("mcp.deleteFile", "failed to delete file", err)
	}
	return nil, DeleteFileOutput{Success: true, Path: input.Path}, nil
}

func (s *Service) renameFile(ctx context.Context, req *mcp.CallToolRequest, input RenameFileInput) (*mcp.CallToolResult, RenameFileOutput, error) {
	if err := s.medium.Rename(input.OldPath, input.NewPath); err != nil {
		return nil, RenameFileOutput{}, log.E("mcp.renameFile", "failed to rename file", err)
	}
	return nil, RenameFileOutput{Success: true, OldPath: input.OldPath, NewPath: input.NewPath}, nil
}

func (s *Service) fileExists(ctx context.Context, req *mcp.CallToolRequest, input FileExistsInput) (*mcp.CallToolResult, FileExistsOutput, error) {
	exists := s.medium.IsFile(input.Path)
	if exists {
		return nil, FileExistsOutput{Exists: true, IsDir: false, Path: input.Path}, nil
	}
	// Check if it's a directory by attempting to list it
	// List might fail if it's a file too (but we checked IsFile) or if doesn't exist.
	_, err := s.medium.List(input.Path)
	isDir := err == nil

	// If List failed, it might mean it doesn't exist OR it's a special file or permissions.
	// Assuming if List works, it's a directory.

	// Refinement: If it doesn't exist, List returns error.

	return nil, FileExistsOutput{Exists: isDir, IsDir: isDir, Path: input.Path}, nil
}

func (s *Service) detectLanguage(ctx context.Context, req *mcp.CallToolRequest, input DetectLanguageInput) (*mcp.CallToolResult, DetectLanguageOutput, error) {
	lang := detectLanguageFromPath(input.Path)
	return nil, DetectLanguageOutput{Language: lang, Path: input.Path}, nil
}

func (s *Service) getSupportedLanguages(ctx context.Context, req *mcp.CallToolRequest, input GetSupportedLanguagesInput) (*mcp.CallToolResult, GetSupportedLanguagesOutput, error) {
	languages := []LanguageInfo{
		{ID: "typescript", Name: "TypeScript", Extensions: []string{".ts", ".tsx"}},
		{ID: "javascript", Name: "JavaScript", Extensions: []string{".js", ".jsx"}},
		{ID: "go", Name: "Go", Extensions: []string{".go"}},
		{ID: "python", Name: "Python", Extensions: []string{".py"}},
		{ID: "rust", Name: "Rust", Extensions: []string{".rs"}},
		{ID: "java", Name: "Java", Extensions: []string{".java"}},
		{ID: "php", Name: "PHP", Extensions: []string{".php"}},
		{ID: "ruby", Name: "Ruby", Extensions: []string{".rb"}},
		{ID: "html", Name: "HTML", Extensions: []string{".html", ".htm"}},
		{ID: "css", Name: "CSS", Extensions: []string{".css"}},
		{ID: "json", Name: "JSON", Extensions: []string{".json"}},
		{ID: "yaml", Name: "YAML", Extensions: []string{".yaml", ".yml"}},
		{ID: "markdown", Name: "Markdown", Extensions: []string{".md", ".markdown"}},
		{ID: "sql", Name: "SQL", Extensions: []string{".sql"}},
		{ID: "shell", Name: "Shell", Extensions: []string{".sh", ".bash"}},
	}
	return nil, GetSupportedLanguagesOutput{Languages: languages}, nil
}

func (s *Service) editDiff(ctx context.Context, req *mcp.CallToolRequest, input EditDiffInput) (*mcp.CallToolResult, EditDiffOutput, error) {
	if input.OldString == "" {
		return nil, EditDiffOutput{}, log.E("mcp.editDiff", "old_string cannot be empty", nil)
	}

	content, err := s.medium.Read(input.Path)
	if err != nil {
		return nil, EditDiffOutput{}, log.E("mcp.editDiff", "failed to read file", err)
	}

	count := 0

	if input.ReplaceAll {
		count = countOccurrences(content, input.OldString)
		if count == 0 {
			return nil, EditDiffOutput{}, log.E("mcp.editDiff", "old_string not found in file", nil)
		}
		content = core.Replace(content, input.OldString, input.NewString)
	} else {
		if !core.Contains(content, input.OldString) {
			return nil, EditDiffOutput{}, log.E("mcp.editDiff", "old_string not found in file", nil)
		}
		content = replaceFirst(content, input.OldString, input.NewString)
		count = 1
	}

	if err := s.medium.Write(input.Path, content); err != nil {
		return nil, EditDiffOutput{}, log.E("mcp.editDiff", "failed to write file", err)
	}

	return nil, EditDiffOutput{
		Path:         input.Path,
		Success:      true,
		Replacements: count,
	}, nil
}

// detectLanguageFromPath maps file extensions to language IDs.
func detectLanguageFromPath(path string) string {
	ext := core.PathExt(path)
	switch ext {
	case ".ts", ".tsx":
		return "typescript"
	case ".js", ".jsx":
		return "javascript"
	case ".go":
		return "go"
	case ".py":
		return "python"
	case ".rs":
		return "rust"
	case ".rb":
		return "ruby"
	case ".java":
		return "java"
	case ".php":
		return "php"
	case ".c", ".h":
		return "c"
	case ".cpp", ".hpp", ".cc", ".cxx":
		return "cpp"
	case ".cs":
		return "csharp"
	case ".html", ".htm":
		return "html"
	case ".css":
		return "css"
	case ".scss":
		return "scss"
	case ".json":
		return "json"
	case ".yaml", ".yml":
		return "yaml"
	case ".xml":
		return "xml"
	case ".md", ".markdown":
		return "markdown"
	case ".sql":
		return "sql"
	case ".sh", ".bash":
		return "shell"
	case ".swift":
		return "swift"
	case ".kt", ".kts":
		return "kotlin"
	default:
		if core.PathBase(path) == "Dockerfile" {
			return "dockerfile"
		}
		return "plaintext"
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
//	// HTTP (set MCP_HTTP_ADDR):
//	os.Setenv("MCP_HTTP_ADDR", "127.0.0.1:9101")
//	svc.Run(ctx)
func (s *Service) Run(ctx context.Context) error {
	if httpAddr := core.Env("MCP_HTTP_ADDR"); httpAddr != "" {
		return s.ServeHTTP(ctx, httpAddr)
	}
	if addr := core.Env("MCP_ADDR"); addr != "" {
		return s.ServeTCP(ctx, addr)
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
