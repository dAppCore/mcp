// SPDX-License-Identifier: EUPL-1.2

//go:build linux

package mcp

import (
	"net"
	"syscall"

	core "dappco.re/go"
	"golang.org/x/sys/unix"
)

// peerCredEnforced reports whether the peer-credential check is a real check on
// this platform (true) or the permissive fallback (false).
const peerCredEnforced = true

// peerCredAllowed rejects a Unix-socket connection whose peer uid does not match
// the server process owner (defence-in-depth). On Linux the peer uid is read
// via SO_PEERCRED.
//
//	if err := peerCredAllowed(conn); err != nil { conn.Close() }
func peerCredAllowed(conn net.Conn) error {
	sysConn, ok := conn.(syscall.Conn)
	if !ok {
		return core.E("mcp.peerCredAllowed", "connection does not expose a syscall handle", nil)
	}
	raw, err := sysConn.SyscallConn()
	if err != nil {
		return core.E("mcp.peerCredAllowed", "failed to obtain syscall conn", err)
	}

	var peerUID uint32
	var credErr error
	ctrlErr := raw.Control(func(fd uintptr) {
		cred, e := unix.GetsockoptUcred(int(fd), unix.SOL_SOCKET, unix.SO_PEERCRED)
		if e != nil {
			credErr = e
			return
		}
		peerUID = cred.Uid
	})
	if ctrlErr != nil {
		return core.E("mcp.peerCredAllowed", "failed to read peer credentials", ctrlErr)
	}
	if credErr != nil {
		return core.E("mcp.peerCredAllowed", "failed to read peer credentials", credErr)
	}

	if owner := uint32(syscall.Getuid()); peerUID != owner {
		return core.E("mcp.peerCredAllowed", core.Sprintf("peer uid %d does not match owner uid %d", peerUID, owner), nil)
	}
	return nil
}
