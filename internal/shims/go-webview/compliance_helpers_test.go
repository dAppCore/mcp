package webview

import (
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
	AssertPanics   = core.AssertPanics
)
