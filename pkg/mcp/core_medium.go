// SPDX-License-Identifier: EUPL-1.2

package mcp

import (
	core "dappco.re/go"
	coreio "dappco.re/go/io"
)

type coreMedium struct {
	medium coreio.Medium
}

var localMedium = newCoreMedium("/")

func newCoreMedium(root string) *coreMedium {
	if root != "/" {
		if r := core.PathEvalSymlinks(root); r.OK {
			root = r.Value.(string)
		}
	}
	medium, err := coreio.NewSandboxed(root)
	if err != nil {
		core.Warn("mcp: filesystem medium unavailable", "root", root, "err", err)
		medium = coreio.NewMemoryMedium()
	}
	return &coreMedium{medium: medium}
}

func (m *coreMedium) Read(path string) (
	string,
	error,
) {
	if m == nil || m.medium == nil {
		return "", core.E("coreMedium.Read", "medium unavailable", nil)
	}
	return m.medium.Read(path)
}

func (m *coreMedium) Write(
	path,
	content string,
) (
	_ error, // result
) {
	if m == nil || m.medium == nil {
		return core.E("coreMedium.Write", "medium unavailable", nil)
	}
	return m.medium.Write(path, content)
}

func (m *coreMedium) WriteMode(
	path,
	content string,
	mode core.FileMode,
) (
	_ error, // result
) {
	if m == nil || m.medium == nil {
		return core.E("coreMedium.WriteMode", "medium unavailable", nil)
	}
	return m.medium.WriteMode(path, content, mode)
}

func (m *coreMedium) EnsureDir(
	path string,
) (
	_ error, // result
) {
	if m == nil || m.medium == nil {
		return core.E("coreMedium.EnsureDir", "medium unavailable", nil)
	}
	return m.medium.EnsureDir(path)
}

func (m *coreMedium) IsFile(path string) bool {
	if m == nil || m.medium == nil {
		return false
	}
	return m.medium.IsFile(path)
}

func (m *coreMedium) Delete(
	path string,
) (
	_ error, // result
) {
	if m == nil || m.medium == nil {
		return core.E("coreMedium.Delete", "medium unavailable", nil)
	}
	if core.Trim(path) == "" {
		return nil
	}
	return m.medium.Delete(path)
}

func (m *coreMedium) DeleteAll(
	path string,
) (
	_ error, // result
) {
	if m == nil || m.medium == nil {
		return core.E("coreMedium.DeleteAll", "medium unavailable", nil)
	}
	return m.medium.DeleteAll(path)
}

func (m *coreMedium) Rename(
	oldPath,
	newPath string,
) (
	_ error, // result
) {
	if m == nil || m.medium == nil {
		return core.E("coreMedium.Rename", "medium unavailable", nil)
	}
	return m.medium.Rename(oldPath, newPath)
}

func (m *coreMedium) List(path string) (
	[]core.FsDirEntry,
	error,
) {
	if m == nil || m.medium == nil {
		return nil, core.E("coreMedium.List", "medium unavailable", nil)
	}
	return m.medium.List(path)
}

func (m *coreMedium) Stat(path string) (
	core.FsFileInfo,
	error,
) {
	if m == nil || m.medium == nil {
		return nil, core.E("coreMedium.Stat", "medium unavailable", nil)
	}
	return m.medium.Stat(path)
}

func (m *coreMedium) Open(path string) (
	core.FsFile,
	error,
) {
	if m == nil || m.medium == nil {
		return nil, core.E("coreMedium.Open", "medium unavailable", nil)
	}
	return m.medium.Open(path)
}

func (m *coreMedium) Create(path string) (
	core.WriteCloser,
	error,
) {
	if m == nil || m.medium == nil {
		return nil, core.E("coreMedium.Create", "medium unavailable", nil)
	}
	return m.medium.Create(path)
}

func (m *coreMedium) Append(path string) (
	core.WriteCloser,
	error,
) {
	if m == nil || m.medium == nil {
		return nil, core.E("coreMedium.Append", "medium unavailable", nil)
	}
	return m.medium.Append(path)
}

func (m *coreMedium) ReadStream(path string) (
	core.ReadCloser,
	error,
) {
	if m == nil || m.medium == nil {
		return nil, core.E("coreMedium.ReadStream", "medium unavailable", nil)
	}
	return m.medium.ReadStream(path)
}

func (m *coreMedium) WriteStream(path string) (
	core.WriteCloser,
	error,
) {
	if m == nil || m.medium == nil {
		return nil, core.E("coreMedium.WriteStream", "medium unavailable", nil)
	}
	return m.medium.WriteStream(path)
}

func (m *coreMedium) Exists(path string) bool {
	if m == nil || m.medium == nil {
		return false
	}
	return m.medium.Exists(path)
}

func (m *coreMedium) IsDir(path string) bool {
	if m == nil || m.medium == nil {
		return false
	}
	return m.medium.IsDir(path)
}
