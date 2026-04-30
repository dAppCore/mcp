// SPDX-License-Identifier: EUPL-1.2

package mcp

import (
	"context"

	core "dappco.re/go"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ServeStdio starts the MCP server over stdin/stdout.
// This is the default transport for CLI integrations.
//
//	if err := svc.ServeStdio(ctx); err != nil {
//	    core.Fatal("stdio transport failed", "err", err)
//	}
func (s *Service) ServeStdio(
	ctx context.Context,
) (
	_ error, // result
) {
	s.logger.Info("MCP Stdio server starting", "user", core.Username())
	reader, ok := core.Stdin().(core.ReadCloser)
	if !ok {
		return core.NewError("stdin is not closable")
	}
	return s.server.Run(ctx, &mcp.IOTransport{
		Reader: reader,
		Writer: sharedStdout,
	})
}
