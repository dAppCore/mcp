package mcp

import (
	"context"
	"net"
	"testing"

	core "dappco.re/go"
	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
)

// moved helpers from ax7_helpers_test.go
func newServiceForTest(t testing.TB, opts Options) *Service {
	t.Helper()
	svc, err := New(opts)
	if err != nil {
		t.Fatalf("New(%+v) failed: %v", opts, err)
	}
	return svc
}

// moved helpers from ax7_triplets_test.go
type T = core.T

var (
	AssertContains  = core.AssertContains
	AssertEqual     = core.AssertEqual
	AssertError     = core.AssertError
	AssertErrorIs   = core.AssertErrorIs
	AssertFalse     = core.AssertFalse
	AssertLen       = core.AssertLen
	AssertNil       = core.AssertNil
	AssertNoError   = core.AssertNoError
	AssertNotEmpty  = core.AssertNotEmpty
	AssertNotEqual  = core.AssertNotEqual
	AssertNotNil    = core.AssertNotNil
	AssertNotPanics = core.AssertNotPanics
	AssertPanics    = core.AssertPanics
	AssertTrue      = core.AssertTrue
	Contains        = core.Contains
	Join            = core.Join
	RequireNoError  = core.RequireNoError
)

func connectionPairForTest(t *T) (*connConnection, net.Conn, func()) {
	t.Helper()
	left, right := net.Pipe()
	transport := &connTransport{conn: left}
	conn, err := transport.Connect(context.Background())
	RequireNoError(t, err)
	return conn.(*connConnection), right, func() {
		if err := left.Close(); err != nil && !core.Is(err, net.ErrClosed) {
			t.Fatalf("close left pipe: %v", err)
		}
		if err := right.Close(); err != nil && !core.Is(err, net.ErrClosed) {
			t.Fatalf("close right pipe: %v", err)
		}
	}
}

func jsonRPCIDForTest(t *T, value any) jsonrpc.ID {
	t.Helper()
	id, err := jsonrpc.MakeID(value)
	RequireNoError(t, err)
	return id
}

func reflectTransformer[T any](got TransformerIn) bool {
	_, ok := got.(T)
	return ok
}

func toolNames(tools []ToolRecord) []string {
	out := make([]string, 0, len(tools))
	for _, tool := range tools {
		out = append(out, tool.Name)
	}
	return out
}

func repeatString(value string, count int) string {
	b := core.NewBuilder()
	for range count {
		b.WriteString(value)
	}
	return b.String()
}
