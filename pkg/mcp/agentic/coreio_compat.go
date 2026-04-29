// SPDX-License-Identifier: EUPL-1.2

package agentic

import core "dappco.re/go"

type localCoreFS struct {
	fs *core.Fs
}

var coreio = struct {
	Local *localCoreFS
}{
	Local: &localCoreFS{fs: (&core.Fs{}).New("/")},
}

func localCoreFSErr(
	r core.Result,
) (
	_ error, // result
) {
	if r.OK {
		return nil
	}
	if err, ok := r.Value.(error); ok && err != nil {
		return err
	}
	if r.Value != nil {
		return core.E("localCoreFS", core.Sprint(r.Value), nil)
	}
	return core.E("localCoreFS", "operation failed", nil)
}

func (l *localCoreFS) Read(path string) (
	string,
	error,
) {
	r := l.fs.Read(path)
	if !r.OK {
		return "", localCoreFSErr(r)
	}
	content, ok := r.Value.(string)
	if !ok {
		return "", core.E("localCoreFS.Read", "unexpected read result", nil)
	}
	return content, nil
}

func (l *localCoreFS) EnsureDir(
	path string,
) (
	_ error, // result
) {
	return localCoreFSErr(l.fs.EnsureDir(path))
}

func (l *localCoreFS) List(path string) (
	[]core.FsDirEntry,
	error,
) {
	r := l.fs.List(path)
	if !r.OK {
		return nil, localCoreFSErr(r)
	}
	entries, ok := r.Value.([]core.FsDirEntry)
	if !ok {
		return nil, core.E("localCoreFS.List", "unexpected list result", nil)
	}
	return entries, nil
}

func (l *localCoreFS) IsFile(path string) bool {
	return l.fs.IsFile(path)
}

func (l *localCoreFS) Delete(
	path string,
) (
	_ error, // result
) {
	return localCoreFSErr(l.fs.Delete(path))
}
