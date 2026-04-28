package rag

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

type QueryResult struct {
	Text       string
	Source     string
	Section    string
	Category   string
	ChunkIndex int
	Score      float32
}

func QueryDocs(context.Context, string, string, int) ([]QueryResult, error) {
	return nil, errors.New("failed to connect to RAG backend")
}

func FormatResultsContext(results []QueryResult) string {
	var b strings.Builder
	b.WriteString("<retrieved_context>\n")
	for _, result := range results {
		if result.Source != "" {
			b.WriteString(fmt.Sprintf("Source: %s\n", result.Source))
		}
		b.WriteString(result.Text)
		if !strings.HasSuffix(result.Text, "\n") {
			b.WriteByte('\n')
		}
	}
	b.WriteString("</retrieved_context>")
	return b.String()
}

func IngestDirectory(context.Context, string, string, bool) error {
	return errors.New("failed to connect to RAG backend")
}

func IngestSingleFile(context.Context, string, string) (int, error) {
	return 0, errors.New("failed to connect to RAG backend")
}

type QdrantConfig struct{}

func DefaultQdrantConfig() QdrantConfig { return QdrantConfig{} }

type QdrantClient struct{}

func NewQdrantClient(QdrantConfig) (*QdrantClient, error) {
	return nil, errors.New("failed to connect to Qdrant")
}

func (c *QdrantClient) Close() error { return nil }

func (c *QdrantClient) ListCollections(context.Context) ([]string, error) {
	return nil, errors.New("failed to list collections")
}

type CollectionDetails struct {
	PointCount uint64
	Status     string
}

func (c *QdrantClient) CollectionInfo(context.Context, string) (CollectionDetails, error) {
	return CollectionDetails{}, errors.New("failed to get collection info")
}
