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
	"strings"

	"forge.lthn.ai/core/go-io"
	"forge.lthn.ai/core/go-log"
	"forge.lthn.ai/core/go-process"
	"forge.lthn.ai/core/go-ws"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Service provides a lightweight MCP server with file operations only.
// For full GUI features, use the core-gui package.
type Service struct {
	server         *mcp.Server
	workspaceRoot  string           // Root directory for file operations (empty = unrestricted)
	medium         io.Medium        // Filesystem medium for sandboxed operations
	subsystems     []Subsystem      // Additional subsystems registered via WithSubsystem
	logger         *log.Logger      // Logger for tool execution auditing
	processService *process.Service // Process management service (optional)
	wsHub          *ws.Hub          // WebSocket hub for real-time streaming (optional)
	wsServer       *http.Server     // WebSocket HTTP server (optional)
	wsAddr         string           // WebSocket server address
	tools          []ToolRecord     // Parallel tool registry for REST bridge
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
				return nil, log.E("mcp.New", "failed to get working directory", err)
			}
			root = cwd
		}
		abs, err := filepath.Abs(root)
		if err != nil {
			return nil, log.E("mcp.New", "invalid workspace root", err)
		}
		m, merr := io.NewSandboxed(abs)
		if merr != nil {
			return nil, log.E("mcp.New", "failed to create workspace medium", merr)
		}
		s.workspaceRoot = abs
		s.medium = m
	}

	s.registerTools(s.server)

	for _, sub := range s.subsystems {
		sub.RegisterTools(s.server)
	}

	return s, nil
}

// Subsystems returns the registered subsystems.
func (s *Service) Subsystems() []Subsystem {
	return s.subsystems
}

// SubsystemsSeq returns an iterator over the registered subsystems.
func (s *Service) SubsystemsSeq() iter.Seq[Subsystem] {
	return slices.Values(s.subsystems)
}

// Tools returns all recorded tool metadata.
func (s *Service) Tools() []ToolRecord {
	return s.tools
}

// ToolsSeq returns an iterator over all recorded tool metadata.
func (s *Service) ToolsSeq() iter.Seq[ToolRecord] {
	return slices.Values(s.tools)
}

// Shutdown gracefully shuts down all subsystems that support it.
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


// WSHub returns the WebSocket hub.
func (s *Service) WSHub() *ws.Hub {
	return s.wsHub
}

// ProcessService returns the process service.
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
type ReadFileInput struct {
	Path string `json:"path"`
}

// ReadFileOutput contains the result of reading a file.
type ReadFileOutput struct {
	Content  string `json:"content"`
	Language string `json:"language"`
	Path     string `json:"path"`
}

// WriteFileInput contains parameters for writing a file.
type WriteFileInput struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

// WriteFileOutput contains the result of writing a file.
type WriteFileOutput struct {
	Success bool   `json:"success"`
	Path    string `json:"path"`
}

// ListDirectoryInput contains parameters for listing a directory.
type ListDirectoryInput struct {
	Path string `json:"path"`
}

// ListDirectoryOutput contains the result of listing a directory.
type ListDirectoryOutput struct {
	Entries []DirectoryEntry `json:"entries"`
	Path    string           `json:"path"`
}

// DirectoryEntry represents a single entry in a directory listing.
type DirectoryEntry struct {
	Name  string `json:"name"`
	Path  string `json:"path"`
	IsDir bool   `json:"isDir"`
	Size  int64  `json:"size"`
}

// CreateDirectoryInput contains parameters for creating a directory.
type CreateDirectoryInput struct {
	Path string `json:"path"`
}

// CreateDirectoryOutput contains the result of creating a directory.
type CreateDirectoryOutput struct {
	Success bool   `json:"success"`
	Path    string `json:"path"`
}

// DeleteFileInput contains parameters for deleting a file.
type DeleteFileInput struct {
	Path string `json:"path"`
}

// DeleteFileOutput contains the result of deleting a file.
type DeleteFileOutput struct {
	Success bool   `json:"success"`
	Path    string `json:"path"`
}

// RenameFileInput contains parameters for renaming a file.
type RenameFileInput struct {
	OldPath string `json:"oldPath"`
	NewPath string `json:"newPath"`
}

// RenameFileOutput contains the result of renaming a file.
type RenameFileOutput struct {
	Success bool   `json:"success"`
	OldPath string `json:"oldPath"`
	NewPath string `json:"newPath"`
}

// FileExistsInput contains parameters for checking file existence.
type FileExistsInput struct {
	Path string `json:"path"`
}

// FileExistsOutput contains the result of checking file existence.
type FileExistsOutput struct {
	Exists bool   `json:"exists"`
	IsDir  bool   `json:"isDir"`
	Path   string `json:"path"`
}

// DetectLanguageInput contains parameters for detecting file language.
type DetectLanguageInput struct {
	Path string `json:"path"`
}

// DetectLanguageOutput contains the detected programming language.
type DetectLanguageOutput struct {
	Language string `json:"language"`
	Path     string `json:"path"`
}

// GetSupportedLanguagesInput is an empty struct for the languages query.
type GetSupportedLanguagesInput struct{}

// GetSupportedLanguagesOutput contains the list of supported languages.
type GetSupportedLanguagesOutput struct {
	Languages []LanguageInfo `json:"languages"`
}

// LanguageInfo describes a supported programming language.
type LanguageInfo struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	Extensions []string `json:"extensions"`
}

// EditDiffInput contains parameters for editing a file via diff.
type EditDiffInput struct {
	Path       string `json:"path"`
	OldString  string `json:"old_string"`
	NewString  string `json:"new_string"`
	ReplaceAll bool   `json:"replace_all,omitempty"`
}

// EditDiffOutput contains the result of a diff-based edit operation.
type EditDiffOutput struct {
	Path         string `json:"path"`
	Success      bool   `json:"success"`
	Replacements int    `json:"replacements"`
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
			Path: filepath.Join(input.Path, e.Name()), // Note: This might be relative path, client might expect absolute?
			// Issue 103 says "Replace ... with local.Medium sandboxing".
			// Previous code returned `filepath.Join(input.Path, e.Name())`.
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
		count = strings.Count(content, input.OldString)
		if count == 0 {
			return nil, EditDiffOutput{}, log.E("mcp.editDiff", "old_string not found in file", nil)
		}
		content = strings.ReplaceAll(content, input.OldString, input.NewString)
	} else {
		if !strings.Contains(content, input.OldString) {
			return nil, EditDiffOutput{}, log.E("mcp.editDiff", "old_string not found in file", nil)
		}
		content = strings.Replace(content, input.OldString, input.NewString, 1)
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
	ext := filepath.Ext(path)
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
		if filepath.Base(path) == "Dockerfile" {
			return "dockerfile"
		}
		return "plaintext"
	}
}

// Run starts the MCP server.
// Transport selection:
//   - MCP_HTTP_ADDR set → Streamable HTTP (with optional MCP_AUTH_TOKEN)
//   - MCP_ADDR set → TCP
//   - Otherwise → Stdio
func (s *Service) Run(ctx context.Context) error {
	if httpAddr := os.Getenv("MCP_HTTP_ADDR"); httpAddr != "" {
		return s.ServeHTTP(ctx, httpAddr)
	}
	if addr := os.Getenv("MCP_ADDR"); addr != "" {
		return s.ServeTCP(ctx, addr)
	}
	return s.server.Run(ctx, &mcp.StdioTransport{})
}


// Server returns the underlying MCP server for advanced configuration.
func (s *Service) Server() *mcp.Server {
	return s.server
}
