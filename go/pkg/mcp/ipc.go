// SPDX-License-Identifier: EUPL-1.2

package mcp

import (
	"context"

	core "dappco.re/go"
)

func (s *Service) handleChannelPushIPC(ctx context.Context, ev ChannelPush) core.Result {
	if core.Trim(ev.Channel) == "" {
		return core.Fail(core.E("mcp.HandleIPCEvents", "channel is required", nil))
	}

	s.ChannelSend(ctx, ev.Channel, ev.Data)
	return core.Ok(nil)
}
