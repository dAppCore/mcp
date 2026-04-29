package process

import (
	"context"
	"time"

	core "dappco.re/go"
)

// moved AX-7 triplet TestProcess_Buffer_Write_Good
func TestProcess_Buffer_Write_Good(t *T) {
	var b captureBuffer
	n, err := b.Write([]byte("hello"))
	AssertNoError(t, err)
	AssertEqual(t, 5, n)
}

// moved AX-7 triplet TestProcess_Buffer_Write_Bad
func TestProcess_Buffer_Write_Bad(t *T) {
	var b captureBuffer
	n, err := b.Write(nil)
	AssertNoError(t, err)
	AssertEqual(t, 0, n)
}

// moved AX-7 triplet TestProcess_Buffer_Write_Ugly
func TestProcess_Buffer_Write_Ugly(t *T) {
	var b captureBuffer
	_, err := b.Write([]byte("a"))
	AssertNoError(t, err)
	_, err = b.Write([]byte("b"))
	AssertNoError(t, err)
	AssertEqual(t, "ab", b.String())
}

// moved AX-7 triplet TestProcess_Buffer_String_Good
func TestProcess_Buffer_String_Good(t *T) {
	var b captureBuffer
	_, err := b.Write([]byte("hello"))
	AssertNoError(t, err)
	AssertEqual(t, "hello", b.String())
}

// moved AX-7 triplet TestProcess_Buffer_String_Bad
func TestProcess_Buffer_String_Bad(t *T) {
	var b captureBuffer
	AssertEqual(t, "", b.String())
	AssertEqual(t, 0, b.buf.Len())
}

// moved AX-7 triplet TestProcess_Buffer_String_Ugly
func TestProcess_Buffer_String_Ugly(t *T) {
	var b captureBuffer
	AssertEqual(t, "", b.String())
	AssertEqual(t, "", b.String())
}

// moved AX-7 triplet TestProcess_NewService_Good
func TestProcess_NewService_Good(t *T) {
	factory := NewService(Options{})
	got, err := factory(nil)
	AssertNoError(t, err)
	AssertNotNil(t, got)
}

// moved AX-7 triplet TestProcess_NewService_Bad
func TestProcess_NewService_Bad(t *T) {
	factory := NewService(Options{})
	got, err := factory(nil)
	AssertNoError(t, err)
	AssertLen(t, got.(*Service).List(), 0)
}

// moved AX-7 triplet TestProcess_NewService_Ugly
func TestProcess_NewService_Ugly(t *T) {
	got, err := NewService(Options{})(nil)
	AssertNoError(t, err)
	AssertLen(t, got.(*Service).List(), 0)
}

// moved AX-7 triplet TestProcess_Service_OnStartup_Good
func TestProcess_Service_OnStartup_Good(t *T) {
	s := processServiceForTest()
	AssertNoError(t, s.OnStartup(context.Background()))
	AssertLen(t, s.List(), 0)
}

// moved AX-7 triplet TestProcess_Service_OnStartup_Bad
func TestProcess_Service_OnStartup_Bad(t *T) {
	s := processServiceForTest()
	AssertNoError(t, s.OnStartup(nil))
	AssertLen(t, s.List(), 0)
}

// moved AX-7 triplet TestProcess_Service_OnStartup_Ugly
func TestProcess_Service_OnStartup_Ugly(t *T) {
	var s *Service
	AssertNoError(t, s.OnStartup(nil))
	AssertNil(t, s)
}

// moved AX-7 triplet TestProcess_Service_StartWithOptions_Good
func TestProcess_Service_StartWithOptions_Good(t *T) {
	proc, err := processServiceForTest().StartWithOptions(context.Background(), RunOptions{Command: "/bin/sh", Args: []string{"-c", "printf ok"}})
	AssertNoError(t, err)
	<-proc.Done()
	AssertEqual(t, "ok", proc.Output())
}

// moved AX-7 triplet TestProcess_Service_StartWithOptions_Bad
func TestProcess_Service_StartWithOptions_Bad(t *T) {
	proc, err := processServiceForTest().StartWithOptions(context.Background(), RunOptions{})
	AssertError(t, err)
	AssertNil(t, proc)
}

// moved AX-7 triplet TestProcess_Service_StartWithOptions_Ugly
func TestProcess_Service_StartWithOptions_Ugly(t *T) {
	dir := t.TempDir()
	proc, err := processServiceForTest().StartWithOptions(context.Background(), RunOptions{
		Command: "/bin/sh",
		Args:    []string{"-c", "pwd"},
		Dir:     dir,
		Env:     []string{"AX7=1"},
	})
	AssertNoError(t, err)
	<-proc.Done()
	AssertContains(t, proc.Output(), dir)
}

// moved AX-7 triplet TestProcess_Service_Get_Good
func TestProcess_Service_Get_Good(t *T) {
	s := processServiceForTest()
	proc, err := s.StartWithOptions(context.Background(), RunOptions{Command: "/bin/sh", Args: []string{"-c", "printf ok"}})
	AssertNoError(t, err)
	got, err := s.Get(proc.ID)
	AssertNoError(t, err)
	AssertEqual(t, proc.ID, got.ID)
}

// moved AX-7 triplet TestProcess_Service_Get_Bad
func TestProcess_Service_Get_Bad(t *T) {
	_, err := processServiceForTest().Get("missing")
	AssertContains(t, err.Error(), "not found")
	AssertError(t, err)
}

// moved AX-7 triplet TestProcess_Service_Get_Ugly
func TestProcess_Service_Get_Ugly(t *T) {
	_, err := (&Service{}).Get("")
	AssertContains(t, err.Error(), "not found")
	AssertError(t, err)
}

// moved AX-7 triplet TestProcess_Service_List_Good
func TestProcess_Service_List_Good(t *T) {
	s := processServiceForTest()
	_, err := s.StartWithOptions(context.Background(), RunOptions{Command: "/bin/sh", Args: []string{"-c", "printf ok"}})
	AssertNoError(t, err)
	AssertLen(t, s.List(), 1)
}

// moved AX-7 triplet TestProcess_Service_List_Bad
func TestProcess_Service_List_Bad(t *T) {
	s := processServiceForTest()
	got := s.List()
	AssertLen(t, got, 0)
}

// moved AX-7 triplet TestProcess_Service_List_Ugly
func TestProcess_Service_List_Ugly(t *T) {
	s := &Service{}
	got := s.List()
	AssertLen(t, got, 0)
}

// moved AX-7 triplet TestProcess_Service_Running_Good
func TestProcess_Service_Running_Good(t *T) {
	s := processServiceForTest()
	proc, err := s.StartWithOptions(context.Background(), RunOptions{Command: "/bin/sh", Args: []string{"-c", "sleep 1"}})
	AssertNoError(t, err)
	defer proc.Shutdown()
	AssertLen(t, s.Running(), 1)
}

// moved AX-7 triplet TestProcess_Service_Running_Bad
func TestProcess_Service_Running_Bad(t *T) {
	s := processServiceForTest()
	got := s.Running()
	AssertLen(t, got, 0)
}

// moved AX-7 triplet TestProcess_Service_Running_Ugly
func TestProcess_Service_Running_Ugly(t *T) {
	s := processServiceForTest()
	proc, err := s.StartWithOptions(context.Background(), RunOptions{Command: "/bin/sh", Args: []string{"-c", "true"}})
	AssertNoError(t, err)
	<-proc.Done()
	AssertLen(t, s.Running(), 0)
}

// moved AX-7 triplet TestProcess_Service_Kill_Good
func TestProcess_Service_Kill_Good(t *T) {
	s := processServiceForTest()
	proc, err := s.StartWithOptions(context.Background(), RunOptions{Command: "/bin/sh", Args: []string{"-c", "sleep 5"}})
	AssertNoError(t, err)
	AssertNoError(t, s.Kill(proc.ID))
}

// moved AX-7 triplet TestProcess_Service_Kill_Bad
func TestProcess_Service_Kill_Bad(t *T) {
	err := processServiceForTest().Kill("missing")
	AssertError(t, err)
	AssertContains(t, err.Error(), "not found")
}

// moved AX-7 triplet TestProcess_Service_Kill_Ugly
func TestProcess_Service_Kill_Ugly(t *T) {
	s := processServiceForTest()
	s.procs = map[string]*Process{"manual": {ID: "manual"}}
	AssertNoError(t, s.Kill("manual"))
}

// moved AX-7 triplet TestProcess_Service_Output_Good
func TestProcess_Service_Output_Good(t *T) {
	s := processServiceForTest()
	proc, err := s.StartWithOptions(context.Background(), RunOptions{Command: "/bin/sh", Args: []string{"-c", "printf ok"}})
	AssertNoError(t, err)
	<-proc.Done()
	got, err := s.Output(proc.ID)
	AssertNoError(t, err)
	AssertEqual(t, "ok", got)
}

// moved AX-7 triplet TestProcess_Service_Output_Bad
func TestProcess_Service_Output_Bad(t *T) {
	got, err := processServiceForTest().Output("missing")
	AssertError(t, err)
	AssertEqual(t, "", got)
}

// moved AX-7 triplet TestProcess_Service_Output_Ugly
func TestProcess_Service_Output_Ugly(t *T) {
	s := processServiceForTest()
	proc, err := s.StartWithOptions(context.Background(), RunOptions{Command: "/bin/sh", Args: []string{"-c", "true"}})
	AssertNoError(t, err)
	<-proc.Done()
	got, err := s.Output(proc.ID)
	AssertNoError(t, err)
	AssertEqual(t, "", got)
}

// moved AX-7 triplet TestProcess_Service_OnShutdown_Good
func TestProcess_Service_OnShutdown_Good(t *T) {
	s := processServiceForTest()
	_, err := s.StartWithOptions(context.Background(), RunOptions{Command: "/bin/sh", Args: []string{"-c", "sleep 5"}})
	AssertNoError(t, err)
	AssertNoError(t, s.OnShutdown(context.Background()))
}

// moved AX-7 triplet TestProcess_Service_OnShutdown_Bad
func TestProcess_Service_OnShutdown_Bad(t *T) {
	s := processServiceForTest()
	AssertNoError(t, s.OnShutdown(context.Background()))
	AssertLen(t, s.List(), 0)
}

// moved AX-7 triplet TestProcess_Service_OnShutdown_Ugly
func TestProcess_Service_OnShutdown_Ugly(t *T) {
	s := &Service{}
	AssertNoError(t, s.OnShutdown(nil))
	AssertLen(t, s.List(), 0)
}

// moved AX-7 triplet TestProcess_Process_Info_Good
func TestProcess_Process_Info_Good(t *T) {
	proc := startProcessForTest(t, "-c", "sleep 1")
	defer proc.Shutdown()
	info := proc.Info()
	AssertEqual(t, StatusRunning, info.Status)
	AssertTrue(t, info.PID > 0)
}

// moved AX-7 triplet TestProcess_Process_Info_Bad
func TestProcess_Process_Info_Bad(t *T) {
	var proc *Process
	AssertNil(t, proc)
	AssertPanics(t, func() { _ = proc.Info() })
}

// moved AX-7 triplet TestProcess_Process_Info_Ugly
func TestProcess_Process_Info_Ugly(t *T) {
	proc := &Process{ID: "manual"}
	info := proc.Info()
	AssertEqual(t, "manual", info.ID)
	AssertEqual(t, Status(""), info.Status)
}

// moved AX-7 triplet TestProcess_Process_Done_Good
func TestProcess_Process_Done_Good(t *T) {
	proc := startProcessForTest(t, "-c", "true")
	<-proc.Done()
	AssertEqual(t, StatusExited, proc.Info().Status)
}

// moved AX-7 triplet TestProcess_Process_Done_Bad
func TestProcess_Process_Done_Bad(t *T) {
	var proc *Process
	AssertNil(t, proc)
	AssertPanics(t, func() { _ = proc.Done() })
}

// moved AX-7 triplet TestProcess_Process_Done_Ugly
func TestProcess_Process_Done_Ugly(t *T) {
	ch := (&Process{}).Done()
	select {
	case <-ch:
	case <-time.After(time.Second):
		t.Fatal("nil done channel fallback did not close")
	}
}

// moved AX-7 triplet TestProcess_Process_Output_Good
func TestProcess_Process_Output_Good(t *T) {
	proc := startProcessForTest(t, "-c", "printf ok")
	<-proc.Done()
	AssertEqual(t, "ok", proc.Output())
}

// moved AX-7 triplet TestProcess_Process_Output_Bad
func TestProcess_Process_Output_Bad(t *T) {
	var proc *Process
	AssertNil(t, proc)
	AssertPanics(t, func() { _ = proc.Output() })
}

// moved AX-7 triplet TestProcess_Process_Output_Ugly
func TestProcess_Process_Output_Ugly(t *T) {
	proc := &Process{}
	AssertEqual(t, "", proc.Output())
	AssertEqual(t, "", proc.ID)
}

// moved AX-7 triplet TestProcess_Process_Shutdown_Good
func TestProcess_Process_Shutdown_Good(t *T) {
	proc := startProcessForTest(t, "-c", "sleep 5")
	AssertNoError(t, proc.Shutdown())
	AssertNotEqual(t, StatusRunning, proc.Info().Status)
}

// moved AX-7 triplet TestProcess_Process_Shutdown_Bad
func TestProcess_Process_Shutdown_Bad(t *T) {
	proc := &Process{}
	AssertNoError(t, proc.Shutdown())
	AssertEqual(t, Status(""), proc.Info().Status)
}

// moved AX-7 triplet TestProcess_Process_Shutdown_Ugly
func TestProcess_Process_Shutdown_Ugly(t *T) {
	proc := &Process{}
	AssertNoError(t, proc.Shutdown())
	AssertNoError(t, proc.Shutdown())
}

// moved AX-7 triplet TestProcess_Process_SendInput_Good
func TestProcess_Process_SendInput_Good(t *T) {
	path := core.Path(t.TempDir(), "input")
	proc := startProcessForTest(t, "-c", "cat > "+path)
	AssertNoError(t, proc.SendInput("ok"))
	AssertNoError(t, proc.stdin.Close())
	<-proc.Done()
	r := core.ReadFile(path)
	AssertTrue(t, r.OK)
	data := r.Value.([]byte)
	AssertEqual(t, "ok", string(data))
}

// moved AX-7 triplet TestProcess_Process_SendInput_Bad
func TestProcess_Process_SendInput_Bad(t *T) {
	err := (&Process{}).SendInput("ok")
	AssertError(t, err)
	AssertContains(t, err.Error(), "stdin unavailable")
}

// moved AX-7 triplet TestProcess_Process_SendInput_Ugly
func TestProcess_Process_SendInput_Ugly(t *T) {
	path := core.Path(t.TempDir(), "input")
	proc := startProcessForTest(t, "-c", "cat > "+path)
	AssertNoError(t, proc.SendInput(""))
	AssertNoError(t, proc.stdin.Close())
	<-proc.Done()
	r := core.ReadFile(path)
	AssertTrue(t, r.OK)
	data := r.Value.([]byte)
	AssertEqual(t, "", string(data))
}

func TestProcess_Bytes_Len_Good(t *T) {
	buf := captureBytes("abc")
	got := buf.Len()
	AssertEqual(t, 3, got)
	AssertEqual(t, len(buf), got)
}

func TestProcess_Bytes_Len_Bad(t *T) {
	var buf captureBytes
	got := buf.Len()
	AssertEqual(t, 0, got)
	AssertEqual(t, len(buf), got)
}

func TestProcess_Bytes_Len_Ugly(t *T) {
	buf := captureBytes([]byte{0, 1, 2, 3})
	got := buf.Len()
	AssertEqual(t, 4, got)
	AssertEqual(t, len(buf), got)
}
