package brain

import (
	core "dappco.re/go"
	coremcp "dappco.re/go/mcp/pkg/mcp"
)

// moved helpers from ax7_triplets_test.go
type T = core.T

var (
	AssertContains = core.AssertContains
	AssertEqual    = core.AssertEqual
	AssertLen      = core.AssertLen
	AssertNil      = core.AssertNil
	AssertNoError  = core.AssertNoError
	AssertNotEmpty = core.AssertNotEmpty
	AssertNotEqual = core.AssertNotEqual
	AssertNotNil   = core.AssertNotNil
	AssertPanics   = core.AssertPanics
	AssertTrue     = core.AssertTrue
	RequireNoError = core.RequireNoError
)

func brainMCPServiceForTest(t *T) *coremcp.Service {
	t.Helper()
	svc, err := coremcp.New(coremcp.Options{WorkspaceRoot: t.TempDir()})
	RequireNoError(t, err)
	return svc
}
