// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"time"

	core "dappco.re/go"
)

// writeAtomic writes content to path by staging it in a temporary file and
// renaming it into place.
//
// This avoids exposing partially written workspace files to agents that may
// read status, prompt, or plan documents while they are being updated.
func writeAtomic(
	path,
	content string,
) (
	_ error, // result
) {
	dir := core.PathDir(path)
	if err := coreio.Local.EnsureDir(dir); err != nil {
		return err
	}

	tmpPath := core.Path(dir, core.Sprintf(".%s.%d.tmp", core.PathBase(path), time.Now().UnixNano()))
	r := core.OpenFile(tmpPath, core.O_CREATE|core.O_EXCL|core.O_WRONLY, 0o600)
	if !r.OK {
		return resultError(r)
	}
	tmp := r.Value.(*core.OSFile)

	cleanup := func() {
		if err := tmp.Close(); err != nil {
			core.Error("agentic: close temporary file failed", `path`, tmpPath, "err", err)
		}
		if remove := core.Remove(tmpPath); !remove.OK && !core.IsNotExist(resultError(remove)) {
			core.Error("agentic: remove temporary file failed", `path`, tmpPath, "err", resultError(remove))
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
		if remove := core.Remove(tmpPath); !remove.OK && !core.IsNotExist(resultError(remove)) {
			core.Error("agentic: remove temporary file after close failure failed", `path`, tmpPath, "err", resultError(remove))
		}
		return err
	}
	if rename := core.Rename(tmpPath, path); !rename.OK {
		if remove := core.Remove(tmpPath); !remove.OK && !core.IsNotExist(resultError(remove)) {
			core.Error("agentic: remove temporary file after rename failure failed", `path`, tmpPath, "err", resultError(remove))
		}
		return resultError(rename)
	}
	return nil
}

func writeAtomicBestEffort(path, content string) {
	if err := writeAtomic(path, content); err != nil {
		core.Error("agentic: atomic write failed", `path`, path, "err", err)
	}
}
