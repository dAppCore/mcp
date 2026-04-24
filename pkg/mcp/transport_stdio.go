// SPDX-License-Identifier: EUPL-1.2

package mcp

import (
	"context"
	"os"

	"dappco.re/go/log"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ServeStdio starts the MCP server over stdin/stdout.
// This is the default transport for CLI integrations.
//
//	if err := svc.ServeStdio(ctx); err != nil {
//	    log.Fatal("stdio transport failed", "err", err)
//	}
func (s *Service) ServeStdio(ctx context.Context) error {
	s.logger.Info("MCP Stdio server starting", "user", log.Username())
	return s.server.Run(ctx, &mcp.IOTransport{
		Reader: os.Stdin,
		Writer: sharedStdout,
	})
}
