// SPDX-License-Identifier: EUPL-1.2

package mcp

import core "dappco.re/go"

type coreMedium struct {
	fs *core.Fs
}

var localMedium = newCoreMedium("/")

func newCoreMedium(root string) *coreMedium {
	if root != "/" {
		if r := core.PathEvalSymlinks(root); r.OK {
			root = r.Value.(string)
		}
	}
	return &coreMedium{fs: (&core.Fs{}).New(root)}
}

func coreMediumErr(r core.Result) error {
	if r.OK {
		return nil
	}
	if err, ok := r.Value.(error); ok && err != nil {
		return err
	}
	if r.Value != nil {
		return core.E("coreMedium", core.Sprint(r.Value), nil)
	}
	return core.E("coreMedium", "operation failed", nil)
}

func (m *coreMedium) Read(path string) (string, error) {
	r := m.fs.Read(path)
	if !r.OK {
		return "", coreMediumErr(r)
	}
	content, ok := r.Value.(string)
	if !ok {
		return "", core.E("coreMedium.Read", "unexpected read result", nil)
	}
	return content, nil
}

func (m *coreMedium) Write(path, content string) error {
	return coreMediumErr(m.fs.Write(path, content))
}

func (m *coreMedium) WriteMode(path, content string, mode core.FileMode) error {
	return coreMediumErr(m.fs.WriteMode(path, content, mode))
}

func (m *coreMedium) EnsureDir(path string) error {
	return coreMediumErr(m.fs.EnsureDir(path))
}

func (m *coreMedium) IsFile(path string) bool {
	return m.fs.IsFile(path)
}

func (m *coreMedium) Delete(path string) error {
	return coreMediumErr(m.fs.Delete(path))
}

func (m *coreMedium) DeleteAll(path string) error {
	return coreMediumErr(m.fs.DeleteAll(path))
}

func (m *coreMedium) Rename(oldPath, newPath string) error {
	return coreMediumErr(m.fs.Rename(oldPath, newPath))
}

func (m *coreMedium) List(path string) ([]core.FsDirEntry, error) {
	r := m.fs.List(path)
	if !r.OK {
		return nil, coreMediumErr(r)
	}
	entries, ok := r.Value.([]core.FsDirEntry)
	if !ok {
		return nil, core.E("coreMedium.List", "unexpected list result", nil)
	}
	return entries, nil
}

func (m *coreMedium) Stat(path string) (core.FsFileInfo, error) {
	r := m.fs.Stat(path)
	if !r.OK {
		return nil, coreMediumErr(r)
	}
	info, ok := r.Value.(core.FsFileInfo)
	if !ok {
		return nil, core.E("coreMedium.Stat", "unexpected stat result", nil)
	}
	return info, nil
}

func (m *coreMedium) Open(path string) (core.FsFile, error) {
	r := m.fs.Open(path)
	if !r.OK {
		return nil, coreMediumErr(r)
	}
	file, ok := r.Value.(core.FsFile)
	if !ok {
		return nil, core.E("coreMedium.Open", "unexpected open result", nil)
	}
	return file, nil
}

func (m *coreMedium) Create(path string) (core.WriteCloser, error) {
	r := m.fs.Create(path)
	if !r.OK {
		return nil, coreMediumErr(r)
	}
	writer, ok := r.Value.(core.WriteCloser)
	if !ok {
		return nil, core.E("coreMedium.Create", "unexpected create result", nil)
	}
	return writer, nil
}

func (m *coreMedium) Append(path string) (core.WriteCloser, error) {
	r := m.fs.Append(path)
	if !r.OK {
		return nil, coreMediumErr(r)
	}
	writer, ok := r.Value.(core.WriteCloser)
	if !ok {
		return nil, core.E("coreMedium.Append", "unexpected append result", nil)
	}
	return writer, nil
}

func (m *coreMedium) ReadStream(path string) (core.ReadCloser, error) {
	r := m.fs.ReadStream(path)
	if !r.OK {
		return nil, coreMediumErr(r)
	}
	reader, ok := r.Value.(core.ReadCloser)
	if !ok {
		return nil, core.E("coreMedium.ReadStream", "unexpected read stream result", nil)
	}
	return reader, nil
}

func (m *coreMedium) WriteStream(path string) (core.WriteCloser, error) {
	r := m.fs.WriteStream(path)
	if !r.OK {
		return nil, coreMediumErr(r)
	}
	writer, ok := r.Value.(core.WriteCloser)
	if !ok {
		return nil, core.E("coreMedium.WriteStream", "unexpected write stream result", nil)
	}
	return writer, nil
}

func (m *coreMedium) Exists(path string) bool {
	return m.fs.Exists(path)
}

func (m *coreMedium) IsDir(path string) bool {
	return m.fs.IsDir(path)
}
