// SPDX-License-Identifier: EUPL-1.2

package client

import core "dappco.re/go"

type localCoreFS struct {
	fs *core.Fs
}

var coreio = struct {
	Local *localCoreFS
}{
	Local: &localCoreFS{fs: (&core.Fs{}).New("/")},
}

func localCoreFSErr(r core.Result) error {
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

func (l *localCoreFS) Stat(path string) (core.FsFileInfo, error) {
	r := l.fs.Stat(path)
	if !r.OK {
		return nil, localCoreFSErr(r)
	}
	info, ok := r.Value.(core.FsFileInfo)
	if !ok {
		return nil, core.E("localCoreFS.Stat", "unexpected stat result", nil)
	}
	return info, nil
}

func (l *localCoreFS) Read(path string) (string, error) {
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

func (l *localCoreFS) WriteMode(path, content string, mode core.FileMode) error {
	return localCoreFSErr(l.fs.WriteMode(path, content, mode))
}
