package process

import (
	"context"
	"os"
	"path/filepath"
	"time"

	core "dappco.re/go"
)

type T = core.T

var (
	AssertContains = core.AssertContains
	AssertEqual    = core.AssertEqual
	AssertError    = core.AssertError
	AssertLen      = core.AssertLen
	AssertNil      = core.AssertNil
	AssertNoError  = core.AssertNoError
	AssertNotEqual = core.AssertNotEqual
	AssertNotNil   = core.AssertNotNil
	AssertPanics   = core.AssertPanics
	AssertTrue     = core.AssertTrue
	RequireNoError = core.RequireNoError
)

func ax7Service() *Service { return &Service{} }

func ax7Start(t *T, args ...string) *Process {
	t.Helper()
	proc, err := ax7Service().StartWithOptions(context.Background(), RunOptions{Command: "/bin/sh", Args: args})
	RequireNoError(t, err)
	return proc
}

func TestAX7_Buffer_Write_Good(t *T) {
	var b captureBuffer
	n, err := b.Write([]byte("hello"))
	AssertNoError(t, err)
	AssertEqual(t, 5, n)
}

func TestAX7_Buffer_Write_Bad(t *T) {
	var b captureBuffer
	n, err := b.Write(nil)
	AssertNoError(t, err)
	AssertEqual(t, 0, n)
}

func TestAX7_Buffer_Write_Ugly(t *T) {
	var b captureBuffer
	_, err := b.Write([]byte("a"))
	AssertNoError(t, err)
	_, err = b.Write([]byte("b"))
	AssertNoError(t, err)
	AssertEqual(t, "ab", b.String())
}

func TestAX7_Buffer_String_Good(t *T) {
	var b captureBuffer
	_, err := b.Write([]byte("hello"))
	AssertNoError(t, err)
	AssertEqual(t, "hello", b.String())
}

func TestAX7_Buffer_String_Bad(t *T) {
	var b captureBuffer
	AssertEqual(t, "", b.String())
	AssertEqual(t, 0, b.buf.Len())
}

func TestAX7_Buffer_String_Ugly(t *T) {
	var b captureBuffer
	AssertEqual(t, "", b.String())
	AssertEqual(t, "", b.String())
}

func TestAX7_NewService_Good(t *T) {
	factory := NewService(Options{})
	got, err := factory(nil)
	AssertNoError(t, err)
	AssertNotNil(t, got)
}

func TestAX7_NewService_Bad(t *T) {
	factory := NewService(Options{})
	got, err := factory(nil)
	AssertNoError(t, err)
	AssertLen(t, got.(*Service).List(), 0)
}

func TestAX7_NewService_Ugly(t *T) {
	got, err := NewService(Options{})(nil)
	AssertNoError(t, err)
	AssertLen(t, got.(*Service).List(), 0)
}

func TestAX7_Service_OnStartup_Good(t *T) {
	s := ax7Service()
	AssertNoError(t, s.OnStartup(context.Background()))
	AssertLen(t, s.List(), 0)
}

func TestAX7_Service_OnStartup_Bad(t *T) {
	s := ax7Service()
	AssertNoError(t, s.OnStartup(nil))
	AssertLen(t, s.List(), 0)
}

func TestAX7_Service_OnStartup_Ugly(t *T) {
	var s *Service
	AssertNoError(t, s.OnStartup(nil))
	AssertNil(t, s)
}

func TestAX7_Service_StartWithOptions_Good(t *T) {
	proc, err := ax7Service().StartWithOptions(context.Background(), RunOptions{Command: "/bin/sh", Args: []string{"-c", "printf ok"}})
	AssertNoError(t, err)
	<-proc.Done()
	AssertEqual(t, "ok", proc.Output())
}

func TestAX7_Service_StartWithOptions_Bad(t *T) {
	proc, err := ax7Service().StartWithOptions(context.Background(), RunOptions{})
	AssertError(t, err)
	AssertNil(t, proc)
}

func TestAX7_Service_StartWithOptions_Ugly(t *T) {
	dir := t.TempDir()
	proc, err := ax7Service().StartWithOptions(context.Background(), RunOptions{
		Command: "/bin/sh",
		Args:    []string{"-c", "pwd"},
		Dir:     dir,
		Env:     []string{"AX7=1"},
	})
	AssertNoError(t, err)
	<-proc.Done()
	AssertContains(t, proc.Output(), dir)
}

func TestAX7_Service_Get_Good(t *T) {
	s := ax7Service()
	proc, err := s.StartWithOptions(context.Background(), RunOptions{Command: "/bin/sh", Args: []string{"-c", "printf ok"}})
	AssertNoError(t, err)
	got, err := s.Get(proc.ID)
	AssertNoError(t, err)
	AssertEqual(t, proc.ID, got.ID)
}

func TestAX7_Service_Get_Bad(t *T) {
	_, err := ax7Service().Get("missing")
	AssertContains(t, err.Error(), "not found")
	AssertError(t, err)
}

func TestAX7_Service_Get_Ugly(t *T) {
	_, err := (&Service{}).Get("")
	AssertContains(t, err.Error(), "not found")
	AssertError(t, err)
}

func TestAX7_Service_List_Good(t *T) {
	s := ax7Service()
	_, err := s.StartWithOptions(context.Background(), RunOptions{Command: "/bin/sh", Args: []string{"-c", "printf ok"}})
	AssertNoError(t, err)
	AssertLen(t, s.List(), 1)
}

func TestAX7_Service_List_Bad(t *T) {
	s := ax7Service()
	got := s.List()
	AssertLen(t, got, 0)
}

func TestAX7_Service_List_Ugly(t *T) {
	s := &Service{}
	got := s.List()
	AssertLen(t, got, 0)
}

func TestAX7_Service_Running_Good(t *T) {
	s := ax7Service()
	proc, err := s.StartWithOptions(context.Background(), RunOptions{Command: "/bin/sh", Args: []string{"-c", "sleep 1"}})
	AssertNoError(t, err)
	defer proc.Shutdown()
	AssertLen(t, s.Running(), 1)
}

func TestAX7_Service_Running_Bad(t *T) {
	s := ax7Service()
	got := s.Running()
	AssertLen(t, got, 0)
}

func TestAX7_Service_Running_Ugly(t *T) {
	s := ax7Service()
	proc, err := s.StartWithOptions(context.Background(), RunOptions{Command: "/bin/sh", Args: []string{"-c", "true"}})
	AssertNoError(t, err)
	<-proc.Done()
	AssertLen(t, s.Running(), 0)
}

func TestAX7_Service_Kill_Good(t *T) {
	s := ax7Service()
	proc, err := s.StartWithOptions(context.Background(), RunOptions{Command: "/bin/sh", Args: []string{"-c", "sleep 5"}})
	AssertNoError(t, err)
	AssertNoError(t, s.Kill(proc.ID))
}

func TestAX7_Service_Kill_Bad(t *T) {
	err := ax7Service().Kill("missing")
	AssertError(t, err)
	AssertContains(t, err.Error(), "not found")
}

func TestAX7_Service_Kill_Ugly(t *T) {
	s := ax7Service()
	s.procs = map[string]*Process{"manual": {ID: "manual"}}
	AssertNoError(t, s.Kill("manual"))
}

func TestAX7_Service_Output_Good(t *T) {
	s := ax7Service()
	proc, err := s.StartWithOptions(context.Background(), RunOptions{Command: "/bin/sh", Args: []string{"-c", "printf ok"}})
	AssertNoError(t, err)
	<-proc.Done()
	got, err := s.Output(proc.ID)
	AssertNoError(t, err)
	AssertEqual(t, "ok", got)
}

func TestAX7_Service_Output_Bad(t *T) {
	got, err := ax7Service().Output("missing")
	AssertError(t, err)
	AssertEqual(t, "", got)
}

func TestAX7_Service_Output_Ugly(t *T) {
	s := ax7Service()
	proc, err := s.StartWithOptions(context.Background(), RunOptions{Command: "/bin/sh", Args: []string{"-c", "true"}})
	AssertNoError(t, err)
	<-proc.Done()
	got, err := s.Output(proc.ID)
	AssertNoError(t, err)
	AssertEqual(t, "", got)
}

func TestAX7_Service_OnShutdown_Good(t *T) {
	s := ax7Service()
	_, err := s.StartWithOptions(context.Background(), RunOptions{Command: "/bin/sh", Args: []string{"-c", "sleep 5"}})
	AssertNoError(t, err)
	AssertNoError(t, s.OnShutdown(context.Background()))
}

func TestAX7_Service_OnShutdown_Bad(t *T) {
	s := ax7Service()
	AssertNoError(t, s.OnShutdown(context.Background()))
	AssertLen(t, s.List(), 0)
}

func TestAX7_Service_OnShutdown_Ugly(t *T) {
	s := &Service{}
	AssertNoError(t, s.OnShutdown(nil))
	AssertLen(t, s.List(), 0)
}

func TestAX7_Process_Info_Good(t *T) {
	proc := ax7Start(t, "-c", "sleep 1")
	defer proc.Shutdown()
	info := proc.Info()
	AssertEqual(t, StatusRunning, info.Status)
	AssertTrue(t, info.PID > 0)
}

func TestAX7_Process_Info_Bad(t *T) {
	var proc *Process
	AssertNil(t, proc)
	AssertPanics(t, func() { _ = proc.Info() })
}

func TestAX7_Process_Info_Ugly(t *T) {
	proc := &Process{ID: "manual"}
	info := proc.Info()
	AssertEqual(t, "manual", info.ID)
	AssertEqual(t, Status(""), info.Status)
}

func TestAX7_Process_Done_Good(t *T) {
	proc := ax7Start(t, "-c", "true")
	<-proc.Done()
	AssertEqual(t, StatusExited, proc.Info().Status)
}

func TestAX7_Process_Done_Bad(t *T) {
	var proc *Process
	AssertNil(t, proc)
	AssertPanics(t, func() { _ = proc.Done() })
}

func TestAX7_Process_Done_Ugly(t *T) {
	ch := (&Process{}).Done()
	select {
	case <-ch:
	case <-time.After(time.Second):
		t.Fatal("nil done channel fallback did not close")
	}
}

func TestAX7_Process_Output_Good(t *T) {
	proc := ax7Start(t, "-c", "printf ok")
	<-proc.Done()
	AssertEqual(t, "ok", proc.Output())
}

func TestAX7_Process_Output_Bad(t *T) {
	var proc *Process
	AssertNil(t, proc)
	AssertPanics(t, func() { _ = proc.Output() })
}

func TestAX7_Process_Output_Ugly(t *T) {
	proc := &Process{}
	AssertEqual(t, "", proc.Output())
	AssertEqual(t, "", proc.ID)
}

func TestAX7_Process_Shutdown_Good(t *T) {
	proc := ax7Start(t, "-c", "sleep 5")
	AssertNoError(t, proc.Shutdown())
	AssertNotEqual(t, StatusRunning, proc.Info().Status)
}

func TestAX7_Process_Shutdown_Bad(t *T) {
	proc := &Process{}
	AssertNoError(t, proc.Shutdown())
	AssertEqual(t, Status(""), proc.Info().Status)
}

func TestAX7_Process_Shutdown_Ugly(t *T) {
	proc := &Process{}
	AssertNoError(t, proc.Shutdown())
	AssertNoError(t, proc.Shutdown())
}

func TestAX7_Process_SendInput_Good(t *T) {
	path := filepath.Join(t.TempDir(), "input")
	proc := ax7Start(t, "-c", "cat > "+path)
	AssertNoError(t, proc.SendInput("ok"))
	AssertNoError(t, proc.stdin.Close())
	<-proc.Done()
	data, err := os.ReadFile(path)
	AssertNoError(t, err)
	AssertEqual(t, "ok", string(data))
}

func TestAX7_Process_SendInput_Bad(t *T) {
	err := (&Process{}).SendInput("ok")
	AssertError(t, err)
	AssertContains(t, err.Error(), "stdin unavailable")
}

func TestAX7_Process_SendInput_Ugly(t *T) {
	path := filepath.Join(t.TempDir(), "input")
	proc := ax7Start(t, "-c", "cat > "+path)
	AssertNoError(t, proc.SendInput(""))
	AssertNoError(t, proc.stdin.Close())
	<-proc.Done()
	data, err := os.ReadFile(path)
	AssertNoError(t, err)
	AssertEqual(t, "", string(data))
}
