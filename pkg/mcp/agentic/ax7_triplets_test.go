// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"path/filepath"

	. "dappco.re/go"
	coremcp "dappco.re/go/mcp/pkg/mcp"
)

func ax7FS(t *T) *localCoreFS {
	t.Helper()
	root, err := filepath.EvalSymlinks(t.TempDir())
	RequireNoError(t, err)
	return &localCoreFS{fs: (&Fs{}).New(root)}
}

func ax7MCPService(t *T) *coremcp.Service {
	t.Helper()
	svc, err := coremcp.New(coremcp.Options{WorkspaceRoot: t.TempDir()})
	RequireNoError(t, err)
	return svc
}

func TestAX7_CoreFS_Read_Good(t *T) {
	fs := ax7FS(t)
	AssertTrue(t, fs.fs.Write("a.txt", "ok").OK)
	got, err := fs.Read("a.txt")
	AssertNoError(t, err)
	AssertEqual(t, "ok", got)
}
func TestAX7_CoreFS_Read_Bad(t *T) {
	fs := ax7FS(t)
	got, err := fs.Read("missing")
	AssertError(t, err)
	AssertEqual(t, "", got)
}
func TestAX7_CoreFS_Read_Ugly(t *T) {
	fs := ax7FS(t)
	AssertTrue(t, fs.fs.Write("empty.txt", "").OK)
	got, err := fs.Read("empty.txt")
	AssertNoError(t, err)
	AssertEqual(t, "", got)
}
func TestAX7_CoreFS_EnsureDir_Good(t *T) {
	fs := ax7FS(t)
	AssertNoError(t, fs.EnsureDir("a/b"))
	AssertTrue(t, fs.IsFile("a/b") == false)
}
func TestAX7_CoreFS_EnsureDir_Bad(t *T) {
	fs := ax7FS(t)
	AssertTrue(t, fs.fs.Write("file", "ok").OK)
	err := fs.EnsureDir("file")
	AssertError(t, err)
}
func TestAX7_CoreFS_EnsureDir_Ugly(t *T) {
	fs := ax7FS(t)
	AssertNoError(t, fs.EnsureDir("a/b"))
	AssertNoError(t, fs.EnsureDir("a/b"))
}
func TestAX7_CoreFS_List_Good(t *T) {
	fs := ax7FS(t)
	AssertTrue(t, fs.fs.Write("a.txt", "ok").OK)
	got, err := fs.List(".")
	AssertNoError(t, err)
	AssertTrue(t, len(got) > 0)
}
func TestAX7_CoreFS_List_Bad(t *T) {
	fs := ax7FS(t)
	got, err := fs.List("missing")
	AssertError(t, err)
	AssertNil(t, got)
}
func TestAX7_CoreFS_List_Ugly(t *T) {
	fs := ax7FS(t)
	AssertNoError(t, fs.EnsureDir("empty"))
	got, err := fs.List("empty")
	AssertNoError(t, err)
	AssertLen(t, got, 0)
}
func TestAX7_CoreFS_IsFile_Good(t *T) {
	fs := ax7FS(t)
	AssertTrue(t, fs.fs.Write("a.txt", "ok").OK)
	AssertTrue(t, fs.IsFile("a.txt"))
}
func TestAX7_CoreFS_IsFile_Bad(t *T) {
	fs := ax7FS(t)
	AssertFalse(t, fs.IsFile("missing"))
	AssertFalse(t, fs.IsFile(""))
}
func TestAX7_CoreFS_IsFile_Ugly(t *T) {
	fs := ax7FS(t)
	AssertNoError(t, fs.EnsureDir("dir"))
	AssertFalse(t, fs.IsFile("dir"))
}
func TestAX7_CoreFS_Delete_Good(t *T) {
	fs := ax7FS(t)
	AssertTrue(t, fs.fs.Write("a.txt", "ok").OK)
	AssertNoError(t, fs.Delete("a.txt"))
}
func TestAX7_CoreFS_Delete_Bad(t *T) {
	fs := ax7FS(t)
	err := fs.Delete("missing")
	AssertError(t, err)
}
func TestAX7_CoreFS_Delete_Ugly(t *T) {
	fs := ax7FS(t)
	AssertTrue(t, fs.fs.Write("nested/a.txt", "").OK)
	AssertNoError(t, fs.Delete("nested/a.txt"))
}
func TestAX7_NewPrep_Good(t *T) {
	t.Setenv("HOME", t.TempDir())
	sub := NewPrep()
	AssertNotNil(t, sub)
	AssertEqual(t, "agentic", sub.Name())
}
func TestAX7_NewPrep_Bad(t *T) {
	t.Setenv("HOME", "")
	sub := NewPrep()
	AssertNotNil(t, sub)
	AssertNil(t, sub.notifier)
}
func TestAX7_NewPrep_Ugly(t *T) {
	t.Setenv("HOME", t.TempDir())
	first := NewPrep()
	second := NewPrep()
	AssertNotNil(t, first)
	AssertNotNil(t, second)
}
func TestAX7_PrepSubsystem_Name_Good(t *T) {
	sub := &PrepSubsystem{}
	AssertEqual(t, "agentic", sub.Name())
	AssertNil(t, sub.notifier)
}
func TestAX7_PrepSubsystem_Name_Bad(t *T) {
	var sub *PrepSubsystem
	AssertEqual(t, "agentic", sub.Name())
	AssertNil(t, sub)
}
func TestAX7_PrepSubsystem_Name_Ugly(t *T) {
	sub := NewPrep()
	AssertEqual(t, "agentic", sub.Name())
	AssertNil(t, sub.notifier)
}
func TestAX7_PrepSubsystem_RegisterTools_Good(t *T) {
	svc := ax7MCPService(t)
	sub := NewPrep()
	sub.RegisterTools(svc)
	AssertTrue(t, len(svc.Tools()) > 0)
}
func TestAX7_PrepSubsystem_RegisterTools_Bad(t *T) {
	sub := NewPrep()
	AssertPanics(t, func() { sub.RegisterTools(nil) })
	AssertEqual(t, "agentic", sub.Name())
}
func TestAX7_PrepSubsystem_RegisterTools_Ugly(t *T) {
	svc := ax7MCPService(t)
	sub := &PrepSubsystem{}
	sub.RegisterTools(svc)
	AssertTrue(t, len(svc.Tools()) > 0)
}
func TestAX7_PrepSubsystem_SetNotifier_Good(t *T) {
	sub := NewPrep()
	svc := ax7MCPService(t)
	sub.SetNotifier(svc)
	AssertEqual(t, svc, sub.notifier)
}
func TestAX7_PrepSubsystem_SetNotifier_Bad(t *T) {
	sub := NewPrep()
	sub.SetNotifier(nil)
	AssertNil(t, sub.notifier)
}
func TestAX7_PrepSubsystem_SetNotifier_Ugly(t *T) {
	sub := &PrepSubsystem{}
	sub.SetNotifier(ax7MCPService(t))
	AssertNotNil(t, sub.notifier)
}
func TestAX7_PrepSubsystem_Shutdown_Good(t *T) {
	sub := NewPrep()
	err := sub.Shutdown(context.Background())
	AssertNoError(t, err)
}
func TestAX7_PrepSubsystem_Shutdown_Bad(t *T) {
	sub := NewPrep()
	err := sub.Shutdown(nil)
	AssertNoError(t, err)
}
func TestAX7_PrepSubsystem_Shutdown_Ugly(t *T) {
	var sub *PrepSubsystem
	err := sub.Shutdown(context.Background())
	AssertNoError(t, err)
}
