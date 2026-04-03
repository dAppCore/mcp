package mcp

import (
	"context"
	"net"
	"testing"
	"time"
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
