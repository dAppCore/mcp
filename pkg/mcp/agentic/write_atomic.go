// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"os"

	core "dappco.re/go/core"
	coreio "dappco.re/go/core/io"
)

// os.CreateTemp, os.Remove, os.Rename are framework-boundary calls for
// atomic file writes — no core equivalent exists for temp file creation.

// writeAtomic writes content to path by staging it in a temporary file and
// renaming it into place.
//
// This avoids exposing partially written workspace files to agents that may
// read status, prompt, or plan documents while they are being updated.
func writeAtomic(path, content string) error {
	dir := core.PathDir(path)
	if err := coreio.Local.EnsureDir(dir); err != nil {
		return err
	}

	tmp, err := os.CreateTemp(dir, "."+core.PathBase(path)+".*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()

	cleanup := func() {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
	}

	if _, err := tmp.WriteString(content); err != nil {
		cleanup()
		return err
	}
	if err := tmp.Sync(); err != nil {
		cleanup()
		return err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	return nil
}
