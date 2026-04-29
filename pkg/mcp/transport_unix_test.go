package mcp

import (
	"context"
	"net"
	"testing"
	"time"

	core "dappco.re/go"
)

func TestRun_Good_UnixTrigger(t *testing.T) {
	s, err := New(Options{})
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	socketPath := shortSocketPath(t, "run")
	t.Setenv("MCP_UNIX_SOCKET", socketPath)
	t.Setenv("MCP_HTTP_ADDR", "")
	t.Setenv("MCP_ADDR", "")

	errCh := make(chan error, 1)
	go func() {
		errCh <- s.Run(ctx)
	}()

	var conn net.Conn
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		conn, err = net.DialTimeout("unix", socketPath, 200*time.Millisecond)
		if err == nil {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if err != nil {
		t.Fatalf("Failed to connect to Unix socket at %s: %v", socketPath, err)
	}
	conn.Close()

	cancel()
	if err := <-errCh; err != nil {
		t.Fatalf("Run failed: %v", err)
	}
}

// moved AX-7 triplet TestTransportUnix_Service_ServeUnix_Good
func TestTransportUnix_Service_ServeUnix_Good(t *T) {
	svc := newServiceForTest(t, Options{})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := svc.ServeUnix(ctx, shortSocketPath(t, "serve"))
	AssertNoError(t, err)
}

// moved AX-7 triplet TestTransportUnix_Service_ServeUnix_Bad
func TestTransportUnix_Service_ServeUnix_Bad(t *T) {
	svc := newServiceForTest(t, Options{})
	err := svc.ServeUnix(context.Background(), core.PathJoin(t.TempDir(), "missing", "sock"))
	AssertError(t, err)
}

// moved AX-7 triplet TestTransportUnix_Service_ServeUnix_Ugly
func TestTransportUnix_Service_ServeUnix_Ugly(t *T) {
	svc := newServiceForTest(t, Options{})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := svc.ServeUnix(ctx, "")
	AssertError(t, err)
}
