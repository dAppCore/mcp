// SPDX-License-Identifier: EUPL-1.2

package mcp

import (
	"context"

	core "dappco.re/go/core"
	"dappco.re/go/core/log"
	"dappco.re/go/core/rag"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Default values for RAG operations.
const (
	DefaultRAGCollection = "hostuk-docs"
	DefaultRAGTopK       = 5
)

// RAGQueryInput contains parameters for querying the RAG vector database.
//
//	input := RAGQueryInput{
//	    Question:   "How do I register a service?",
//	    Collection: "core-docs",
//	    TopK:       3,
//	}
type RAGQueryInput struct {
	Question   string `json:"question"`             // e.g. "How do I register a service?"
	Collection string `json:"collection,omitempty"` // e.g. "core-docs" (default: "hostuk-docs")
	TopK       int    `json:"topK,omitempty"`       // e.g. 3 (default: 5)
}

// RAGQueryResult represents a single query result with relevance score.
//
//	// r.Source == "docs/services.md", r.Score == 0.92
type RAGQueryResult struct {
	Content    string  `json:"content"`              // matched text chunk
	Source     string  `json:"source"`               // e.g. "docs/services.md"
	Section    string  `json:"section,omitempty"`    // e.g. "Service Registration"
	Category   string  `json:"category,omitempty"`   // e.g. "guide"
	ChunkIndex int     `json:"chunkIndex,omitempty"` // chunk position within source
	Score      float32 `json:"score"`                // similarity score (0.0-1.0)
}

// RAGQueryOutput contains the results of a RAG query.
//
//	// len(out.Results) == 3, out.Collection == "core-docs"
type RAGQueryOutput struct {
	Results    []RAGQueryResult `json:"results"`    // ranked by similarity score
	Query      string           `json:"query"`      // the original question
	Collection string           `json:"collection"` // collection that was searched
	Context    string           `json:"context"`    // pre-formatted context string for LLM consumption
}

// RAGIngestInput contains parameters for ingesting documents into the RAG database.
//
//	input := RAGIngestInput{
//	    Path:       "docs/",
//	    Collection: "core-docs",
//	    Recreate:   true,
//	}
type RAGIngestInput struct {
	Path       string `json:"path"`                 // e.g. "docs/" or "docs/services.md"
	Collection string `json:"collection,omitempty"` // e.g. "core-docs" (default: "hostuk-docs")
	Recreate   bool   `json:"recreate,omitempty"`   // true to drop and recreate the collection
}

// RAGIngestOutput contains the result of a RAG ingest operation.
//
//	// out.Success == true, out.Chunks == 42, out.Collection == "core-docs"
type RAGIngestOutput struct {
	Success    bool   `json:"success"`           // true when ingest completed
	Path       string `json:"path"`              // e.g. "docs/"
	Collection string `json:"collection"`        // e.g. "core-docs"
	Chunks     int    `json:"chunks"`            // number of chunks ingested
	Message    string `json:"message,omitempty"` // human-readable summary
}

// RAGCollectionsInput contains parameters for listing collections.
//
//	input := RAGCollectionsInput{ShowStats: true}
type RAGCollectionsInput struct {
	ShowStats bool `json:"show_stats,omitempty"` // true to include point counts and status
}

// CollectionInfo contains information about a Qdrant collection.
//
//	// ci.Name == "core-docs", ci.PointsCount == 1500, ci.Status == "green"
type CollectionInfo struct {
	Name        string `json:"name"`         // e.g. "core-docs"
	PointsCount uint64 `json:"points_count"` // number of vectors stored
	Status      string `json:"status"`       // e.g. "green"
}

// RAGCollectionsOutput contains the list of available collections.
//
//	// len(out.Collections) == 2
type RAGCollectionsOutput struct {
	Collections []CollectionInfo `json:"collections"` // all Qdrant collections
}

// registerRAGTools adds RAG tools to the MCP server.
func (s *Service) registerRAGTools(server *mcp.Server) {
	addToolRecorded(s, server, "rag", &mcp.Tool{
		Name:        "rag_query",
		Description: "Query the RAG vector database for relevant documentation. Returns semantically similar content based on the query.",
	}, s.ragQuery)

	addToolRecorded(s, server, "rag", &mcp.Tool{
		Name:        "rag_ingest",
		Description: "Ingest documents into the RAG vector database. Supports both single files and directories.",
	}, s.ragIngest)

	addToolRecorded(s, server, "rag", &mcp.Tool{
		Name:        "rag_collections",
		Description: "List all available collections in the RAG vector database.",
	}, s.ragCollections)
}

// ragQuery handles the rag_query tool call.
func (s *Service) ragQuery(ctx context.Context, req *mcp.CallToolRequest, input RAGQueryInput) (*mcp.CallToolResult, RAGQueryOutput, error) {
	// Apply defaults
	collection := input.Collection
	if collection == "" {
		collection = DefaultRAGCollection
	}
	topK := input.TopK
	if topK <= 0 {
		topK = DefaultRAGTopK
	}

	s.logger.Info("MCP tool execution", "tool", "rag_query", "question", input.Question, "collection", collection, "topK", topK, "user", log.Username())

	// Validate input
	if input.Question == "" {
		return nil, RAGQueryOutput{}, log.E("ragQuery", "question cannot be empty", nil)
	}

	// Call the RAG query function
	results, err := rag.QueryDocs(ctx, input.Question, collection, topK)
	if err != nil {
		log.Error("mcp: rag query failed", "question", input.Question, "collection", collection, "err", err)
		return nil, RAGQueryOutput{}, log.E("ragQuery", "failed to query RAG", err)
	}

	// Convert results
	output := RAGQueryOutput{
		Results:    make([]RAGQueryResult, len(results)),
		Query:      input.Question,
		Collection: collection,
		Context:    rag.FormatResultsContext(results),
	}
	for i, r := range results {
		output.Results[i] = RAGQueryResult{
			Content:    r.Text,
			Source:     r.Source,
			Section:    r.Section,
			Category:   r.Category,
			ChunkIndex: r.ChunkIndex,
			Score:      r.Score,
		}
	}

	return nil, output, nil
}

// ragIngest handles the rag_ingest tool call.
func (s *Service) ragIngest(ctx context.Context, req *mcp.CallToolRequest, input RAGIngestInput) (*mcp.CallToolResult, RAGIngestOutput, error) {
	// Apply defaults
	collection := input.Collection
	if collection == "" {
		collection = DefaultRAGCollection
	}

	s.logger.Security("MCP tool execution", "tool", "rag_ingest", "path", input.Path, "collection", collection, "recreate", input.Recreate, "user", log.Username())

	// Validate input
	if input.Path == "" {
		return nil, RAGIngestOutput{}, log.E("ragIngest", "path cannot be empty", nil)
	}

	// Check if path is a file or directory using the medium
	info, err := s.medium.Stat(input.Path)
	if err != nil {
		log.Error("mcp: rag ingest stat failed", "path", input.Path, "err", err)
		return nil, RAGIngestOutput{}, log.E("ragIngest", "failed to access path", err)
	}
	resolvedPath := s.resolveWorkspacePath(input.Path)

	var message string
	var chunks int
	if info.IsDir() {
		// Ingest directory
		err = rag.IngestDirectory(ctx, resolvedPath, collection, input.Recreate)
		if err != nil {
			log.Error("mcp: rag ingest directory failed", "path", input.Path, "collection", collection, "err", err)
			return nil, RAGIngestOutput{}, log.E("ragIngest", "failed to ingest directory", err)
		}
		message = core.Sprintf("Successfully ingested directory %s into collection %s", input.Path, collection)
	} else {
		// Ingest single file
		chunks, err = rag.IngestSingleFile(ctx, resolvedPath, collection)
		if err != nil {
			log.Error("mcp: rag ingest file failed", "path", input.Path, "collection", collection, "err", err)
			return nil, RAGIngestOutput{}, log.E("ragIngest", "failed to ingest file", err)
		}
		message = core.Sprintf("Successfully ingested file %s (%d chunks) into collection %s", input.Path, chunks, collection)
	}

	return nil, RAGIngestOutput{
		Success:    true,
		Path:       input.Path,
		Collection: collection,
		Chunks:     chunks,
		Message:    message,
	}, nil
}

// ragCollections handles the rag_collections tool call.
func (s *Service) ragCollections(ctx context.Context, req *mcp.CallToolRequest, input RAGCollectionsInput) (*mcp.CallToolResult, RAGCollectionsOutput, error) {
	s.logger.Info("MCP tool execution", "tool", "rag_collections", "show_stats", input.ShowStats, "user", log.Username())

	// Create Qdrant client with default config
	qdrantClient, err := rag.NewQdrantClient(rag.DefaultQdrantConfig())
	if err != nil {
		log.Error("mcp: rag collections connect failed", "err", err)
		return nil, RAGCollectionsOutput{}, log.E("ragCollections", "failed to connect to Qdrant", err)
	}
	defer func() { _ = qdrantClient.Close() }()

	// List collections
	collectionNames, err := qdrantClient.ListCollections(ctx)
	if err != nil {
		log.Error("mcp: rag collections list failed", "err", err)
		return nil, RAGCollectionsOutput{}, log.E("ragCollections", "failed to list collections", err)
	}

	// Build collection info list
	collections := make([]CollectionInfo, len(collectionNames))
	for i, name := range collectionNames {
		collections[i] = CollectionInfo{Name: name}

		// Fetch stats if requested
		if input.ShowStats {
			info, err := qdrantClient.CollectionInfo(ctx, name)
			if err != nil {
				log.Error("mcp: rag collection info failed", "collection", name, "err", err)
				// Continue with defaults on error
				continue
			}
			collections[i].PointsCount = info.PointCount
			collections[i].Status = info.Status
		}
	}

	return nil, RAGCollectionsOutput{
		Collections: collections,
	}, nil
}
