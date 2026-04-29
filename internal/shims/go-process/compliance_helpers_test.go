package process

import (
	"context"

	core "dappco.re/go"
)

// moved helpers from ax7_triplets_test.go
type T = core.T

var (
	AssertContains = core.AssertContains
	AssertEqual    = core.AssertEqual
	AssertError    = core.AssertError
	AssertLen      = core.AssertLen
	AssertNil      = core.AssertNil
	AssertNoError  = core.AssertNoError
	AssertNotEqual = core.AssertNotEqual
	AssertNotNil   = core.AssertNotNil
	AssertPanics   = core.AssertPanics
	AssertTrue     = core.AssertTrue
	RequireNoError = core.RequireNoError
)

func processServiceForTest() *Service { return &Service{} }

func startProcessForTest(t *T, args ...string) *Process {
	t.Helper()
	proc, err := processServiceForTest().StartWithOptions(context.Background(), RunOptions{Command: "/bin/sh", Args: args})
	RequireNoError(t, err)
	return proc
}
