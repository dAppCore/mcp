// SPDX-License-Identifier: EUPL-1.2

package mcp

import (
	"context"
	"net"

	core "dappco.re/go"
)

// ServeUnix starts a Unix domain socket server for the MCP service.
// The socket file is created at the given path and removed on shutdown.
//
//	if err := svc.ServeUnix(ctx, "/tmp/core-mcp.sock"); err != nil {
//	    core.Fatal("unix transport failed", "err", err)
//	}
func (s *Service) ServeUnix(ctx context.Context, socketPath string) error {
	// Clean up any stale socket file
	if err := localMedium.Delete(socketPath); err != nil {
		s.logger.Warn("Failed to remove stale socket", "path", socketPath, "err", err)
	}

	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		return err
	}
	defer func() {
		_ = listener.Close()
		_ = localMedium.Delete(socketPath)
	}()

	// Close listener when context is cancelled to unblock Accept
	go func() {
		<-ctx.Done()
		_ = listener.Close()
	}()

	s.logger.Security("MCP Unix server listening", "path", socketPath, "user", core.Username())

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

		s.logger.Security("MCP Unix connection accepted", "user", core.Username())
		go s.handleConnection(ctx, conn)
	}
}
