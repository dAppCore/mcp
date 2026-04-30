package ide

import (
	core "dappco.re/go"
	coremcp "dappco.re/go/mcp/pkg/mcp"
)

// moved helpers from ax7_triplets_test.go
type T = core.T

var (
	AssertEqual     = core.AssertEqual
	AssertError     = core.AssertError
	AssertFalse     = core.AssertFalse
	AssertLen       = core.AssertLen
	AssertNil       = core.AssertNil
	AssertNoError   = core.AssertNoError
	AssertNotEmpty  = core.AssertNotEmpty
	AssertNotNil    = core.AssertNotNil
	AssertNotPanics = core.AssertNotPanics
	AssertPanics    = core.AssertPanics
	AssertTrue      = core.AssertTrue
	RequireNoError  = core.RequireNoError
)

func coremcpTestService(t *T) *coremcp.Service {
	t.Helper()
	svc, err := coremcp.New(coremcp.Options{WorkspaceRoot: t.TempDir()})
	RequireNoError(t, err)
	return svc
}
