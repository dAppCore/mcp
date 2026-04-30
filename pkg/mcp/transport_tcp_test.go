package mcp

import (
	"context"
	"io"
	"net"
	"testing"
	"time"

	core "dappco.re/go"
	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
)

func TestNewTCPTransport_Defaults(t *testing.T) {
	// Test that empty string gets replaced with default address constant
	// Note: We can't actually bind to 9100 as it may be in use,
	// so we verify the address is set correctly before Listen is called
	if DefaultTCPAddr != "127.0.0.1:9100" {
		t.Errorf("Expected default constant 127.0.0.1:9100, got %s", DefaultTCPAddr)
	}

	// Test with a dynamic port to verify transport creation works
	tr, err := NewTCPTransport("127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to create transport with dynamic port: %v", err)
	}
	defer tr.listener.Close()

	// Verify we got a valid address
	if tr.addr != "127.0.0.1:0" {
		t.Errorf("Expected address to be set, got %s", tr.addr)
	}
}

func TestNormalizeTCPAddr_Good_Defaults(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "empty", in: "", want: DefaultTCPAddr},
		{name: "missing host", in: ":9100", want: "127.0.0.1:9100"},
		{name: "explicit host", in: "127.0.0.1:9100", want: "127.0.0.1:9100"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizeTCPAddr(tt.in); got != tt.want {
				t.Fatalf("normalizeTCPAddr(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestNewTCPTransport_Warning(t *testing.T) {
	// Capture warning output via setDiagWriter (mutex-protected, no race).
	buf := core.NewBuffer()
	old := setDiagWriter(buf)
	defer setDiagWriter(old)

	// Trigger warning — use port 0 (OS assigns free port)
	tr, err := NewTCPTransport("0.0.0.0:0")
	if err != nil {
		t.Fatalf("Failed to create transport: %v", err)
	}
	defer tr.listener.Close()

	output := buf.String()
	if !core.Contains(output, "WARNING") {
		t.Error("Expected warning for binding to 0.0.0.0, but didn't find it in stderr")
	}
}

func TestServeTCP_Connection(t *testing.T) {
	s, err := New(Options{})
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Use a random port for testing to avoid collisions
	addr := "127.0.0.1:0"

	// Create transport first to get the actual address if we use :0
	tr, err := NewTCPTransport(addr)
	if err != nil {
		t.Fatalf("Failed to create transport: %v", err)
	}
	actualAddr := tr.listener.Addr().String()
	tr.listener.Close() // Close it so ServeTCP can re-open it or use the same address

	// Start server in background
	errCh := make(chan error, 1)
	go func() {
		errCh <- s.ServeTCP(ctx, actualAddr)
	}()

	// Give it a moment to start
	time.Sleep(100 * time.Millisecond)

	// Connect to the server
	conn, err := net.Dial("tcp", actualAddr)
	if err != nil {
		t.Fatalf("Failed to connect to server: %v", err)
	}
	defer conn.Close()

	// Verify we can write to it
	_, err = conn.Write([]byte("{}\n"))
	if err != nil {
		t.Errorf("Failed to write to connection: %v", err)
	}

	// Shutdown server
	cancel()
	err = <-errCh
	if err != nil {
		t.Errorf("ServeTCP returned error: %v", err)
	}
}

func TestRun_TCPTrigger(t *testing.T) {
	s, err := New(Options{})
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set MCP_ADDR to empty to trigger default TCP
	t.Setenv("MCP_ADDR", "")

	// We use a random port for testing, but Run will try to use 127.0.0.1:9100 by default if we don't override.
	// Since 9100 might be in use, we'll set MCP_ADDR to use :0 (random port)
	t.Setenv("MCP_ADDR", "127.0.0.1:0")

	errCh := make(chan error, 1)
	go func() {
		errCh <- s.Run(ctx)
	}()

	// Give it a moment to start
	time.Sleep(100 * time.Millisecond)

	// Since we can't easily get the actual port used by Run (it's internal),
	// we just verify it didn't immediately fail.
	select {
	case err := <-errCh:
		t.Fatalf("Run failed immediately: %v", err)
	default:
		// still running, which is good
	}

	cancel()
	_ = <-errCh
}

func TestServeTCP_MultipleConnections(t *testing.T) {
	s, err := New(Options{})
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	addr := "127.0.0.1:0"
	tr, err := NewTCPTransport(addr)
	if err != nil {
		t.Fatalf("Failed to create transport: %v", err)
	}
	actualAddr := tr.listener.Addr().String()
	tr.listener.Close()

	errCh := make(chan error, 1)
	go func() {
		errCh <- s.ServeTCP(ctx, actualAddr)
	}()

	time.Sleep(100 * time.Millisecond)

	// Connect multiple clients
	const numClients = 3
	for i := range numClients {
		conn, err := net.Dial("tcp", actualAddr)
		if err != nil {
			t.Fatalf("Client %d failed to connect: %v", i, err)
		}
		defer conn.Close()
		_, err = conn.Write([]byte("{}\n"))
		if err != nil {
			t.Errorf("Client %d failed to write: %v", i, err)
		}
	}

	cancel()
	err = <-errCh
	if err != nil {
		t.Errorf("ServeTCP returned error: %v", err)
	}
}

// moved AX-7 triplet TestTransportTcp_Connection_Close_Good
func TestTransportTcp_Connection_Close_Good(t *T) {
	c, _, cleanup := connectionPairForTest(t)
	defer cleanup()
	AssertNoError(t, c.Close())
}

// moved AX-7 triplet TestTransportTcp_Connection_Close_Bad
func TestTransportTcp_Connection_Close_Bad(t *T) {
	var c *connConnection
	AssertPanics(t, func() { _ = c.Close() })
	AssertNil(t, c)
}

// moved AX-7 triplet TestTransportTcp_Connection_Close_Ugly
func TestTransportTcp_Connection_Close_Ugly(t *T) {
	c, _, cleanup := connectionPairForTest(t)
	defer cleanup()
	AssertNoError(t, c.Close())
	AssertNoError(t, c.Close())
}

// moved AX-7 triplet TestTransportTcp_Connection_Read_Good
func TestTransportTcp_Connection_Read_Good(t *T) {
	c, right, cleanup := connectionPairForTest(t)
	defer cleanup()
	go func() { _, _ = right.Write([]byte(`{"jsonrpc":"2.0","id":1,"method":"x"}` + "\n")) }()
	msg, err := c.Read(context.Background())
	AssertNoError(t, err)
	AssertNotNil(t, msg)
}

// moved AX-7 triplet TestTransportTcp_Connection_Read_Bad
func TestTransportTcp_Connection_Read_Bad(t *T) {
	c, right, cleanup := connectionPairForTest(t)
	defer cleanup()
	go func() { _, _ = right.Write([]byte(`bad` + "\n")) }()
	msg, err := c.Read(context.Background())
	AssertError(t, err)
	AssertNil(t, msg)
}

// moved AX-7 triplet TestTransportTcp_Connection_Read_Ugly
func TestTransportTcp_Connection_Read_Ugly(t *T) {
	c, right, cleanup := connectionPairForTest(t)
	defer cleanup()
	AssertNoError(t, right.Close())
	_, err := c.Read(context.Background())
	AssertErrorIs(t, err, io.EOF)
}

// moved AX-7 triplet TestTransportTcp_Connection_SessionID_Good
func TestTransportTcp_Connection_SessionID_Good(t *T) {
	c, _, cleanup := connectionPairForTest(t)
	defer cleanup()
	AssertContains(t, c.SessionID(), "tcp-")
}

// moved AX-7 triplet TestTransportTcp_Connection_SessionID_Bad
func TestTransportTcp_Connection_SessionID_Bad(t *T) {
	var c *connConnection
	AssertPanics(t, func() { _ = c.SessionID() })
	AssertNil(t, c)
}

// moved AX-7 triplet TestTransportTcp_Connection_SessionID_Ugly
func TestTransportTcp_Connection_SessionID_Ugly(t *T) {
	c, _, cleanup := connectionPairForTest(t)
	defer cleanup()
	AssertNotEmpty(t, c.SessionID())
}

// moved AX-7 triplet TestTransportTcp_Connection_Write_Good
func TestTransportTcp_Connection_Write_Good(t *T) {
	c, right, cleanup := connectionPairForTest(t)
	defer cleanup()
	done := make(chan string, 1)
	go func() {
		buf := make([]byte, 512)
		n, _ := right.Read(buf)
		done <- string(buf[:n])
	}()
	err := c.Write(context.Background(), &jsonrpc.Request{ID: jsonRPCIDForTest(t, "1"), Method: "x"})
	AssertNoError(t, err)
	AssertContains(t, <-done, `"method":"x"`)
}

// moved AX-7 triplet TestTransportTcp_Connection_Write_Bad
func TestTransportTcp_Connection_Write_Bad(t *T) {
	c, _, cleanup := connectionPairForTest(t)
	defer cleanup()
	err := c.Write(context.Background(), nil)
	AssertError(t, err)
}

// moved AX-7 triplet TestTransportTcp_Connection_Write_Ugly
func TestTransportTcp_Connection_Write_Ugly(t *T) {
	c, right, cleanup := connectionPairForTest(t)
	defer cleanup()
	AssertNoError(t, right.Close())
	err := c.Write(context.Background(), &jsonrpc.Request{ID: jsonRPCIDForTest(t, "1"), Method: "x"})
	AssertError(t, err)
}

// moved AX-7 triplet TestTransportTcp_NewTCPTransport_Good
func TestTransportTcp_NewTCPTransport_Good(t *T) {
	tr, err := NewTCPTransport("127.0.0.1:0")
	AssertNoError(t, err)
	defer tr.listener.Close()
	AssertNotNil(t, tr.listener)
}

// moved AX-7 triplet TestTransportTcp_NewTCPTransport_Bad
func TestTransportTcp_NewTCPTransport_Bad(t *T) {
	tr, err := NewTCPTransport("127.0.0.1:bad")
	AssertError(t, err)
	AssertNil(t, tr)
}

// moved AX-7 triplet TestTransportTcp_NewTCPTransport_Ugly
func TestTransportTcp_NewTCPTransport_Ugly(t *T) {
	tr, err := NewTCPTransport(":0")
	AssertNoError(t, err)
	defer tr.listener.Close()
	AssertNotEmpty(t, tr.listener.Addr().String())
}

// moved AX-7 triplet TestTransportTcp_Service_ServeTCP_Good
func TestTransportTcp_Service_ServeTCP_Good(t *T) {
	svc := newServiceForTest(t, Options{})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := svc.ServeTCP(ctx, "127.0.0.1:0")
	AssertNoError(t, err)
}

// moved AX-7 triplet TestTransportTcp_Service_ServeTCP_Bad
func TestTransportTcp_Service_ServeTCP_Bad(t *T) {
	svc := newServiceForTest(t, Options{})
	err := svc.ServeTCP(context.Background(), "127.0.0.1:bad")
	AssertError(t, err)
}

// moved AX-7 triplet TestTransportTcp_Service_ServeTCP_Ugly
func TestTransportTcp_Service_ServeTCP_Ugly(t *T) {
	svc := newServiceForTest(t, Options{})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := svc.ServeTCP(ctx, "")
	AssertNoError(t, err)
}

// moved AX-7 triplet TestTransportTcp_Transport_Connect_Good
func TestTransportTcp_Transport_Connect_Good(t *T) {
	left, right := net.Pipe()
	defer left.Close()
	defer right.Close()
	conn, err := (&connTransport{conn: left}).Connect(context.Background())
	AssertNoError(t, err)
	AssertNotNil(t, conn)
}

// moved AX-7 triplet TestTransportTcp_Transport_Connect_Bad
func TestTransportTcp_Transport_Connect_Bad(t *T) {
	conn, err := (&connTransport{}).Connect(context.Background())
	AssertNoError(t, err)
	AssertNotNil(t, conn)
}

// moved AX-7 triplet TestTransportTcp_Transport_Connect_Ugly
func TestTransportTcp_Transport_Connect_Ugly(t *T) {
	left, right := net.Pipe()
	right.Close()
	conn, err := (&connTransport{conn: left}).Connect(nil)
	AssertNoError(t, err)
	AssertNotNil(t, conn)
}
