package ws

import (
	core "dappco.re/go"
)

// moved helpers from ax7_triplets_test.go
type T = core.T

var (
	AssertEqual    = core.AssertEqual
	AssertLen      = core.AssertLen
	AssertNil      = core.AssertNil
	AssertNoError  = core.AssertNoError
	AssertNotEqual = core.AssertNotEqual
	AssertNotNil   = core.AssertNotNil
	AssertPanics   = core.AssertPanics
)
