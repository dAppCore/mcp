// SPDX-License-Identifier: EUPL-1.2

package mcp

import (
	"context"
	"net"
	"syscall"

	core "dappco.re/go"
)

// unixSocketDirMode is the permission applied to the parent directory of an MCP
// Unix socket — owner-only, so a sibling user cannot enumerate or plant sockets.
const unixSocketDirMode = core.FileMode(0o700)

// unixSocketFileMode is the permission applied to the MCP Unix socket itself —
// owner read/write only, so only the owning user can connect.
const unixSocketFileMode = core.FileMode(0o600)

// ServeUnix starts a Unix domain socket server for the MCP service.
// The socket file is created at the given path and removed on shutdown.
//
// The raw Unix transport hands each connection straight to handleConnection
// with NO per-request auth surface — it cannot carry per-request credentials
// the way the HTTP+SSE served transport does (see ServeHTTP). It is therefore
// suitable only for single-trust, same-user local use; auth-needing callers
// must use the HTTP+SSE transport. As defence-in-depth this transport narrows
// the socket directory to 0700, the socket file to 0600, and rejects
// connections whose peer uid does not match the server process owner.
//
//	if err := svc.ServeUnix(ctx, "/tmp/core-mcp.sock"); err != nil {
//	    core.Fatal("unix transport failed", "err", err)
//	}
func (s *Service) ServeUnix(
	ctx context.Context,
	socketPath string,
) (
	_ error, // result
) {
	if core.Trim(socketPath) == "" {
		return core.E("mcp.ServeUnix", "socket path is required", nil)
	}

	// Narrow the parent directory to owner-only BEFORE binding, so the socket
	// is never briefly reachable by a sibling user.
	if dir := core.PathDir(socketPath); dir != "" && dir != "." && dir != "/" {
		if r := core.MkdirAll(dir, unixSocketDirMode); !r.OK {
			return core.E("mcp.ServeUnix", "failed to create socket directory "+dir, nil)
		}
	}

	// Clean up any stale socket file
	if err := localMedium.Delete(socketPath); err != nil {
		s.logger.Warn("Failed to remove stale socket", `path`, socketPath, "err", err)
	}

	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		return err
	}

	// Narrow the socket file to owner read/write before the first Accept.
	if err := syscall.Chmod(socketPath, uint32(unixSocketFileMode)); err != nil {
		if cerr := listener.Close(); cerr != nil {
			s.logger.Warn("Failed to close Unix listener after chmod failure", `path`, socketPath, "err", cerr)
		}
		if derr := localMedium.Delete(socketPath); derr != nil {
			s.logger.Warn("Failed to remove Unix socket after chmod failure", `path`, socketPath, "err", derr)
		}
		return core.E("mcp.ServeUnix", "failed to chmod socket "+socketPath, err)
	}

	defer func() {
		if err := listener.Close(); err != nil {
			s.logger.Warn("Failed to close Unix listener", `path`, socketPath, "err", err)
		}
		if err := localMedium.Delete(socketPath); err != nil {
			s.logger.Warn("Failed to remove Unix socket", `path`, socketPath, "err", err)
		}
	}()

	// Close listener when context is cancelled to unblock Accept
	go func() {
		<-ctx.Done()
		if err := listener.Close(); err != nil {
			s.logger.Warn("Failed to close Unix listener on cancellation", `path`, socketPath, "err", err)
		}
	}()

	s.logger.Security("MCP Unix server listening", `path`, socketPath, "user", core.Username())

	for {
		conn, err := listener.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				return nil
			default:
				s.logger.Error("MCP Unix accept error", "err", err, "user", core.Username())
				continue
			}
		}

		if err := peerCredAllowed(conn); err != nil {
			s.logger.Security("MCP Unix connection rejected by peer-cred check", "err", err, "user", core.Username())
			if cerr := conn.Close(); cerr != nil {
				s.logger.Warn("Failed to close rejected Unix connection", "err", cerr)
			}
			continue
		}

		s.logger.Security("MCP Unix connection accepted", "user", core.Username())
		go s.handleConnection(ctx, conn)
	}
}
