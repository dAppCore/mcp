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

func (l *localCoreFS) Stat(path string) (
	core.FsFileInfo,
	error,
) {
	r := l.fs.Stat(path)
	if !r.OK {
		if err, ok := r.Value.(error); ok && err != nil {
			return nil, err
		}
		if r.Value != nil {
			return nil, core.E("localCoreFS", core.Sprint(r.Value), nil)
		}
		return nil, core.E("localCoreFS", "operation failed", nil)
	}
	info, ok := r.Value.(core.FsFileInfo)
	if !ok {
		return nil, core.E("localCoreFS.Stat", "unexpected stat result", nil)
	}
	return info, nil
}

func (l *localCoreFS) Read(path string) (
	string,
	error,
) {
	r := l.fs.Read(path)
	if !r.OK {
		if err, ok := r.Value.(error); ok && err != nil {
			return "", err
		}
		if r.Value != nil {
			return "", core.E("localCoreFS", core.Sprint(r.Value), nil)
		}
		return "", core.E("localCoreFS", "operation failed", nil)
	}
	content, ok := r.Value.(string)
	if !ok {
		return "", core.E("localCoreFS.Read", "unexpected read result", nil)
	}
	return content, nil
}

func (l *localCoreFS) WriteMode(
	path,
	content string,
	mode core.FileMode,
) (
	_ error, // result
) {
	r := l.fs.WriteMode(path, content, mode)
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
