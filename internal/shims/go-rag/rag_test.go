package rag

import (
	"context"

	. "dappco.re/go"
)

// moved AX-7 triplet TestRag_FormatResultsContext_Good
func TestRag_FormatResultsContext_Good(t *T) {
	got := FormatResultsContext([]QueryResult{{Source: "doc.md", Text: "hello"}})
	AssertContains(t, got, "<retrieved_context>")
	AssertContains(t, got, "Source: doc.md")
	AssertContains(t, got, "hello\n")
}

// moved AX-7 triplet TestRag_FormatResultsContext_Bad
func TestRag_FormatResultsContext_Bad(t *T) {
	got := FormatResultsContext(nil)
	AssertContains(t, got, "<retrieved_context>")
	AssertEqual(t, "<retrieved_context>\n</retrieved_context>", got)
}

// moved AX-7 triplet TestRag_FormatResultsContext_Ugly
func TestRag_FormatResultsContext_Ugly(t *T) {
	got := FormatResultsContext([]QueryResult{{Text: "already\n"}})
	AssertContains(t, got, "<retrieved_context>")
	AssertContains(t, got, "already\n</retrieved_context>")
}

// moved AX-7 triplet TestRag_QueryDocs_Good
func TestRag_QueryDocs_Good(t *T) {
	got, err := QueryDocs(context.Background(), "docs", "query", 3)
	AssertError(t, err)
	AssertNil(t, got)
}

// moved AX-7 triplet TestRag_QueryDocs_Bad
func TestRag_QueryDocs_Bad(t *T) {
	got, err := QueryDocs(context.Background(), "", "", 0)
	AssertError(t, err)
	AssertNil(t, got)
}

// moved AX-7 triplet TestRag_QueryDocs_Ugly
func TestRag_QueryDocs_Ugly(t *T) {
	got, err := QueryDocs(nil, "collection", "query", -1)
	AssertError(t, err)
	AssertNil(t, got)
}

// moved AX-7 triplet TestRag_IngestDirectory_Good
func TestRag_IngestDirectory_Good(t *T) {
	err := IngestDirectory(context.Background(), t.TempDir(), "docs", false)
	AssertError(t, err)
	AssertContains(t, err.Error(), "RAG backend")
}

// moved AX-7 triplet TestRag_IngestDirectory_Bad
func TestRag_IngestDirectory_Bad(t *T) {
	err := IngestDirectory(context.Background(), "", "", false)
	AssertError(t, err)
	AssertContains(t, err.Error(), "RAG backend")
}

// moved AX-7 triplet TestRag_IngestDirectory_Ugly
func TestRag_IngestDirectory_Ugly(t *T) {
	err := IngestDirectory(nil, "/", "collection", true)
	AssertError(t, err)
	AssertContains(t, err.Error(), "RAG backend")
}

// moved AX-7 triplet TestRag_IngestSingleFile_Good
func TestRag_IngestSingleFile_Good(t *T) {
	n, err := IngestSingleFile(context.Background(), "doc.md", "docs")
	AssertError(t, err)
	AssertEqual(t, 0, n)
}

// moved AX-7 triplet TestRag_IngestSingleFile_Bad
func TestRag_IngestSingleFile_Bad(t *T) {
	n, err := IngestSingleFile(context.Background(), "", "")
	AssertError(t, err)
	AssertEqual(t, 0, n)
}

// moved AX-7 triplet TestRag_IngestSingleFile_Ugly
func TestRag_IngestSingleFile_Ugly(t *T) {
	n, err := IngestSingleFile(nil, "/tmp/missing", "collection")
	AssertError(t, err)
	AssertEqual(t, 0, n)
}

// moved AX-7 triplet TestRag_DefaultQdrantConfig_Good
func TestRag_DefaultQdrantConfig_Good(t *T) {
	got := DefaultQdrantConfig()
	AssertEqual(t, QdrantConfig{}, got)
	AssertEqual(t, got, DefaultQdrantConfig())
}

// moved AX-7 triplet TestRag_DefaultQdrantConfig_Bad
func TestRag_DefaultQdrantConfig_Bad(t *T) {
	got := DefaultQdrantConfig()
	want := QdrantConfig{}
	AssertEqual(t, want, got)
	AssertNotNil(t, &got)
}

// moved AX-7 triplet TestRag_DefaultQdrantConfig_Ugly
func TestRag_DefaultQdrantConfig_Ugly(t *T) {
	AssertNotPanics(t, func() {
		got := DefaultQdrantConfig()
		AssertEqual(t, QdrantConfig{}, got)
	})
}

// moved AX-7 triplet TestRag_NewQdrantClient_Good
func TestRag_NewQdrantClient_Good(t *T) {
	client, err := NewQdrantClient(QdrantConfig{})
	AssertError(t, err)
	AssertNil(t, client)
}

// moved AX-7 triplet TestRag_NewQdrantClient_Bad
func TestRag_NewQdrantClient_Bad(t *T) {
	cfg := DefaultQdrantConfig()
	client, err := NewQdrantClient(cfg)
	AssertError(t, err)
	AssertNil(t, client)
	AssertEqual(t, QdrantConfig{}, cfg)
}

// moved AX-7 triplet TestRag_NewQdrantClient_Ugly
func TestRag_NewQdrantClient_Ugly(t *T) {
	client, err := NewQdrantClient(DefaultQdrantConfig())
	AssertError(t, err)
	AssertNil(t, client)
}

// moved AX-7 triplet TestRag_QdrantClient_Close_Good
func TestRag_QdrantClient_Close_Good(t *T) {
	c := &QdrantClient{}
	AssertNoError(t, c.Close())
	AssertNotNil(t, c)
}

// moved AX-7 triplet TestRag_QdrantClient_Close_Bad
func TestRag_QdrantClient_Close_Bad(t *T) {
	c := &QdrantClient{}
	AssertNoError(t, c.Close())
	AssertNoError(t, c.Close())
}

// moved AX-7 triplet TestRag_QdrantClient_Close_Ugly
func TestRag_QdrantClient_Close_Ugly(t *T) {
	var c *QdrantClient
	AssertNoError(t, c.Close())
	AssertNil(t, c)
}

// moved AX-7 triplet TestRag_QdrantClient_ListCollections_Good
func TestRag_QdrantClient_ListCollections_Good(t *T) {
	got, err := (&QdrantClient{}).ListCollections(context.Background())
	AssertError(t, err)
	AssertNil(t, got)
}

// moved AX-7 triplet TestRag_QdrantClient_ListCollections_Bad
func TestRag_QdrantClient_ListCollections_Bad(t *T) {
	client := &QdrantClient{}
	got, err := client.ListCollections(nil)
	AssertError(t, err)
	AssertNil(t, got)
	AssertNotNil(t, client)
}

// moved AX-7 triplet TestRag_QdrantClient_ListCollections_Ugly
func TestRag_QdrantClient_ListCollections_Ugly(t *T) {
	got, err := ((*QdrantClient)(nil)).ListCollections(nil)
	AssertError(t, err)
	AssertNil(t, got)
}

// moved AX-7 triplet TestRag_QdrantClient_CollectionInfo_Good
func TestRag_QdrantClient_CollectionInfo_Good(t *T) {
	got, err := (&QdrantClient{}).CollectionInfo(context.Background(), "docs")
	AssertError(t, err)
	AssertEqual(t, CollectionDetails{}, got)
}

// moved AX-7 triplet TestRag_QdrantClient_CollectionInfo_Bad
func TestRag_QdrantClient_CollectionInfo_Bad(t *T) {
	got, err := (&QdrantClient{}).CollectionInfo(context.Background(), "")
	AssertError(t, err)
	AssertEqual(t, CollectionDetails{}, got)
}

// moved AX-7 triplet TestRag_QdrantClient_CollectionInfo_Ugly
func TestRag_QdrantClient_CollectionInfo_Ugly(t *T) {
	got, err := ((*QdrantClient)(nil)).CollectionInfo(nil, "collection")
	AssertError(t, err)
	AssertEqual(t, CollectionDetails{}, got)
}
