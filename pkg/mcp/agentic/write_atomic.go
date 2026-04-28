// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"os"

	core "dappco.re/go"
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
		if err := tmp.Close(); err != nil {
			core.Error("agentic: close temporary file failed", "path", tmpPath, "err", err)
		}
		if err := os.Remove(tmpPath); err != nil && !os.IsNotExist(err) {
			core.Error("agentic: remove temporary file failed", "path", tmpPath, "err", err)
		}
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
		if removeErr := os.Remove(tmpPath); removeErr != nil && !os.IsNotExist(removeErr) {
			core.Error("agentic: remove temporary file after close failure failed", "path", tmpPath, "err", removeErr)
		}
		return err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		if removeErr := os.Remove(tmpPath); removeErr != nil && !os.IsNotExist(removeErr) {
			core.Error("agentic: remove temporary file after rename failure failed", "path", tmpPath, "err", removeErr)
		}
		return err
	}
	return nil
}

func writeAtomicBestEffort(path, content string) {
	if err := writeAtomic(path, content); err != nil {
		core.Error("agentic: atomic write failed", "path", path, "err", err)
	}
}
