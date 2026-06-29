// SPDX-License-Identifier: EUPL-1.2

//go:build !linux && !darwin

package mcp

import (
	"net"
)

// peerCredEnforced reports whether the peer-credential check is a real check on
// this platform (true) or the permissive fallback (false).
const peerCredEnforced = false

// peerCredAllowed is a permissive fallback on platforms without a portable
// peer-credential syscall. The socket-perm narrowing in ServeUnix (0700 dir,
// 0600 socket) remains the effective access control on these platforms.
//
//	if err := peerCredAllowed(conn); err != nil { conn.Close() }
func peerCredAllowed(_ net.Conn) error {
	return nil
}
