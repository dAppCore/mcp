package mcp

import (
	"context"
	"net"
	"testing"
	"time"

	core "dappco.re/go"
)

// TestServeUnix_Good_SocketPerms asserts S1.4: the socket file is narrowed to
// 0600 (owner read/write only) before connections are accepted, and a same-uid
// peer can connect.
func TestServeUnix_Good_SocketPerms(t *testing.T) {
	s, err := New(Options{})
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	socketPath := shortSocketPath(t, "perms")
	errCh := make(chan error, 1)
	go func() { errCh <- s.ServeUnix(ctx, socketPath) }()

	// Wait for the socket to appear and accept a same-uid connection.
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
		t.Fatalf("same-uid peer failed to connect: %v", err)
	}
	conn.Close()

	info, err := localMedium.Stat(socketPath)
	if err != nil {
		t.Fatalf("stat socket: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("expected socket perms 0600, got %o", perm)
	}

	cancel()
	if err := <-errCh; err != nil {
		t.Fatalf("ServeUnix returned error: %v", err)
	}
}

// TestPeerCredAllowed_Bad_NonSyscallConn asserts the peer-cred check rejects a
// connection that does not expose a syscall handle (where the check is
// compiled in). On the permissive fallback platforms it is a no-op.
func TestPeerCredAllowed_Bad_NonSyscallConn(t *testing.T) {
	err := peerCredAllowed(fakeNonSyscallConn{})
	if peerCredEnforced && err == nil {
		t.Fatal("expected non-syscall conn to be rejected by peer-cred check")
	}
	if !peerCredEnforced && err != nil {
		t.Fatalf("expected permissive fallback to accept, got: %v", err)
	}
}

// fakeNonSyscallConn is a net.Conn that does NOT implement syscall.Conn, so the
// peer-cred check cannot obtain a file descriptor from it.
type fakeNonSyscallConn struct{}

func (fakeNonSyscallConn) Read([]byte) (int, error)         { return 0, nil }
func (fakeNonSyscallConn) Write([]byte) (int, error)        { return 0, nil }
func (fakeNonSyscallConn) Close() error                     { return nil }
func (fakeNonSyscallConn) LocalAddr() net.Addr              { return nil }
func (fakeNonSyscallConn) RemoteAddr() net.Addr             { return nil }
func (fakeNonSyscallConn) SetDeadline(time.Time) error      { return nil }
func (fakeNonSyscallConn) SetReadDeadline(time.Time) error  { return nil }
func (fakeNonSyscallConn) SetWriteDeadline(time.Time) error { return nil }

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
//
// ServeUnix now creates a missing parent directory (S1.4 MkdirAll), so a merely
// absent directory is no longer an error. The genuine failure case is a path
// whose parent is a regular FILE — MkdirAll cannot turn it into a directory.
func TestTransportUnix_Service_ServeUnix_Bad(t *T) {
	svc := newServiceForTest(t, Options{})
	fileAsParent := core.PathJoin(t.TempDir(), "iam-a-file")
	if r := core.WriteFile(fileAsParent, []byte("x"), 0600); !r.OK {
		t.Fatalf("seed file: %v", r.Value)
	}
	err := svc.ServeUnix(context.Background(), core.PathJoin(fileAsParent, "sock"))
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
