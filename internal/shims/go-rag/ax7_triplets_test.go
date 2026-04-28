package rag

import (
	"context"

	. "dappco.re/go"
)

func TestAX7_FormatResultsContext_Good(t *T) {
	got := FormatResultsContext([]QueryResult{{Source: "doc.md", Text: "hello"}})
	AssertContains(t, got, "<retrieved_context>")
	AssertContains(t, got, "Source: doc.md")
	AssertContains(t, got, "hello\n")
}

func TestAX7_FormatResultsContext_Bad(t *T) {
	got := FormatResultsContext(nil)
	AssertContains(t, got, "<retrieved_context>")
	AssertEqual(t, "<retrieved_context>\n</retrieved_context>", got)
}

func TestAX7_FormatResultsContext_Ugly(t *T) {
	got := FormatResultsContext([]QueryResult{{Text: "already\n"}})
	AssertContains(t, got, "<retrieved_context>")
	AssertContains(t, got, "already\n</retrieved_context>")
}

func TestAX7_QueryDocs_Good(t *T) {
	got, err := QueryDocs(context.Background(), "docs", "query", 3)
	AssertError(t, err)
	AssertNil(t, got)
}

func TestAX7_QueryDocs_Bad(t *T) {
	got, err := QueryDocs(context.Background(), "", "", 0)
	AssertError(t, err)
	AssertNil(t, got)
}

func TestAX7_QueryDocs_Ugly(t *T) {
	got, err := QueryDocs(nil, "collection", "query", -1)
	AssertError(t, err)
	AssertNil(t, got)
}

func TestAX7_IngestDirectory_Good(t *T) {
	err := IngestDirectory(context.Background(), t.TempDir(), "docs", false)
	AssertError(t, err)
	AssertContains(t, err.Error(), "RAG backend")
}

func TestAX7_IngestDirectory_Bad(t *T) {
	err := IngestDirectory(context.Background(), "", "", false)
	AssertError(t, err)
	AssertContains(t, err.Error(), "RAG backend")
}

func TestAX7_IngestDirectory_Ugly(t *T) {
	err := IngestDirectory(nil, "/", "collection", true)
	AssertError(t, err)
	AssertContains(t, err.Error(), "RAG backend")
}

func TestAX7_IngestSingleFile_Good(t *T) {
	n, err := IngestSingleFile(context.Background(), "doc.md", "docs")
	AssertError(t, err)
	AssertEqual(t, 0, n)
}

func TestAX7_IngestSingleFile_Bad(t *T) {
	n, err := IngestSingleFile(context.Background(), "", "")
	AssertError(t, err)
	AssertEqual(t, 0, n)
}

func TestAX7_IngestSingleFile_Ugly(t *T) {
	n, err := IngestSingleFile(nil, "/tmp/missing", "collection")
	AssertError(t, err)
	AssertEqual(t, 0, n)
}

func TestAX7_DefaultQdrantConfig_Good(t *T) {
	got := DefaultQdrantConfig()
	AssertEqual(t, QdrantConfig{}, got)
	AssertEqual(t, got, DefaultQdrantConfig())
}

func TestAX7_DefaultQdrantConfig_Bad(t *T) {
	got := DefaultQdrantConfig()
	AssertEqual(t, QdrantConfig{}, got)
	AssertEqual(t, got, DefaultQdrantConfig())
}

func TestAX7_DefaultQdrantConfig_Ugly(t *T) {
	got := DefaultQdrantConfig()
	AssertEqual(t, QdrantConfig{}, got)
	AssertEqual(t, got, DefaultQdrantConfig())
}

func TestAX7_NewQdrantClient_Good(t *T) {
	client, err := NewQdrantClient(QdrantConfig{})
	AssertError(t, err)
	AssertNil(t, client)
}

func TestAX7_NewQdrantClient_Bad(t *T) {
	client, err := NewQdrantClient(QdrantConfig{})
	AssertError(t, err)
	AssertNil(t, client)
}

func TestAX7_NewQdrantClient_Ugly(t *T) {
	client, err := NewQdrantClient(DefaultQdrantConfig())
	AssertError(t, err)
	AssertNil(t, client)
}

func TestAX7_QdrantClient_Close_Good(t *T) {
	c := &QdrantClient{}
	AssertNoError(t, c.Close())
	AssertNotNil(t, c)
}

func TestAX7_QdrantClient_Close_Bad(t *T) {
	c := &QdrantClient{}
	AssertNoError(t, c.Close())
	AssertNoError(t, c.Close())
}

func TestAX7_QdrantClient_Close_Ugly(t *T) {
	var c *QdrantClient
	AssertNoError(t, c.Close())
	AssertNil(t, c)
}

func TestAX7_QdrantClient_ListCollections_Good(t *T) {
	got, err := (&QdrantClient{}).ListCollections(context.Background())
	AssertError(t, err)
	AssertNil(t, got)
}

func TestAX7_QdrantClient_ListCollections_Bad(t *T) {
	got, err := (&QdrantClient{}).ListCollections(context.Background())
	AssertError(t, err)
	AssertNil(t, got)
}

func TestAX7_QdrantClient_ListCollections_Ugly(t *T) {
	got, err := ((*QdrantClient)(nil)).ListCollections(nil)
	AssertError(t, err)
	AssertNil(t, got)
}

func TestAX7_QdrantClient_CollectionInfo_Good(t *T) {
	got, err := (&QdrantClient{}).CollectionInfo(context.Background(), "docs")
	AssertError(t, err)
	AssertEqual(t, CollectionDetails{}, got)
}

func TestAX7_QdrantClient_CollectionInfo_Bad(t *T) {
	got, err := (&QdrantClient{}).CollectionInfo(context.Background(), "")
	AssertError(t, err)
	AssertEqual(t, CollectionDetails{}, got)
}

func TestAX7_QdrantClient_CollectionInfo_Ugly(t *T) {
	got, err := ((*QdrantClient)(nil)).CollectionInfo(nil, "collection")
	AssertError(t, err)
	AssertEqual(t, CollectionDetails{}, got)
}
