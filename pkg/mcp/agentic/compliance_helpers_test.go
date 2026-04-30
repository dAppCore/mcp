package agentic

import (
	. "dappco.re/go"
	coremcp "dappco.re/go/mcp/pkg/mcp"
)

// moved helpers from ax7_triplets_test.go
func agenticFSForTest(t *T) *localCoreFS {
	t.Helper()
	r := PathEvalSymlinks(t.TempDir())
	AssertTrue(t, r.OK)
	root := r.Value.(string)
	return &localCoreFS{fs: (&Fs{}).New(root)}
}

func agenticMCPServiceForTest(t *T) *coremcp.Service {
	t.Helper()
	svc, err := coremcp.New(coremcp.Options{WorkspaceRoot: t.TempDir()})
	RequireNoError(t, err)
	return svc
}
