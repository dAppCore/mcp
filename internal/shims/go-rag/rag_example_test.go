package rag

import core "dappco.re/go"

func ExampleQueryDocs() {
	_ = QueryDocs
	core.Println("QueryDocs")
	// Output: QueryDocs
}

func ExampleFormatResultsContext() {
	_ = FormatResultsContext
	core.Println("FormatResultsContext")
	// Output: FormatResultsContext
}

func ExampleIngestDirectory() {
	_ = IngestDirectory
	core.Println("IngestDirectory")
	// Output: IngestDirectory
}

func ExampleIngestSingleFile() {
	_ = IngestSingleFile
	core.Println("IngestSingleFile")
	// Output: IngestSingleFile
}

func ExampleDefaultQdrantConfig() {
	_ = DefaultQdrantConfig
	core.Println("DefaultQdrantConfig")
	// Output: DefaultQdrantConfig
}

func ExampleNewQdrantClient() {
	_ = NewQdrantClient
	core.Println("NewQdrantClient")
	// Output: NewQdrantClient
}

func ExampleQdrantClient_Close() {
	var subject QdrantClient
	_ = subject.Close
	core.Println("QdrantClient.Close")
	// Output: QdrantClient.Close
}

func ExampleQdrantClient_ListCollections() {
	var subject QdrantClient
	_ = subject.ListCollections
	core.Println("QdrantClient.ListCollections")
	// Output: QdrantClient.ListCollections
}

func ExampleQdrantClient_CollectionInfo() {
	var subject QdrantClient
	_ = subject.CollectionInfo
	core.Println("QdrantClient.CollectionInfo")
	// Output: QdrantClient.CollectionInfo
}
