package mcp

import (
	"context"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// stubSubsystem is a minimal Subsystem for testing.
type stubSubsystem struct {
	name            string
	toolsRegistered bool
}

func (s *stubSubsystem) Name() string { return s.name }

func (s *stubSubsystem) RegisterTools(server *mcp.Server) {
	s.toolsRegistered = true
}

// notifierSubsystem verifies notifier wiring happens before tool registration.
type notifierSubsystem struct {
	stubSubsystem
	notifierSet               bool
	sawNotifierAtRegistration bool
}

func (s *notifierSubsystem) SetNotifier(n Notifier) {
	s.notifierSet = n != nil
}

func (s *notifierSubsystem) RegisterTools(server *mcp.Server) {
	s.sawNotifierAtRegistration = s.notifierSet
	s.toolsRegistered = true
}

// shutdownSubsystem tracks Shutdown calls.
type shutdownSubsystem struct {
	stubSubsystem
	shutdownCalled bool
	shutdownErr    error
}

func (s *shutdownSubsystem) Shutdown(_ context.Context) error {
	s.shutdownCalled = true
	return s.shutdownErr
}

func TestSubsystem_Good_Registration(t *testing.T) {
	sub := &stubSubsystem{name: "test-sub"}
	svc, err := New(Options{Subsystems: []Subsystem{sub}})
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	if len(svc.Subsystems()) != 1 {
		t.Fatalf("expected 1 subsystem, got %d", len(svc.Subsystems()))
	}
	if svc.Subsystems()[0].Name() != "test-sub" {
		t.Errorf("expected name 'test-sub', got %q", svc.Subsystems()[0].Name())
	}
}

func TestSubsystem_Good_ToolsRegistered(t *testing.T) {
	sub := &stubSubsystem{name: "tools-sub"}
	_, err := New(Options{Subsystems: []Subsystem{sub}})
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	if !sub.toolsRegistered {
		t.Error("expected RegisterTools to have been called")
	}
}

func TestSubsystem_Good_MultipleSubsystems(t *testing.T) {
	sub1 := &stubSubsystem{name: "sub-1"}
	sub2 := &stubSubsystem{name: "sub-2"}
	svc, err := New(Options{Subsystems: []Subsystem{sub1, sub2}})
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	if len(svc.Subsystems()) != 2 {
		t.Fatalf("expected 2 subsystems, got %d", len(svc.Subsystems()))
	}
	if !sub1.toolsRegistered || !sub2.toolsRegistered {
		t.Error("expected all subsystems to have RegisterTools called")
	}
}

func TestSubsystem_Good_NilEntriesIgnoredAndSnapshots(t *testing.T) {
	sub := &stubSubsystem{name: "snap-sub"}
	svc, err := New(Options{Subsystems: []Subsystem{nil, sub}})
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	subs := svc.Subsystems()
	if len(subs) != 1 {
		t.Fatalf("expected 1 subsystem after filtering nil entries, got %d", len(subs))
	}
	if subs[0].Name() != "snap-sub" {
		t.Fatalf("expected snap-sub, got %q", subs[0].Name())
	}

	subs[0] = nil
	if svc.Subsystems()[0] == nil {
		t.Fatal("expected Subsystems() to return a snapshot, not the live slice")
	}
}

func TestSubsystem_Good_NotifierSetBeforeRegistration(t *testing.T) {
	sub := &notifierSubsystem{stubSubsystem: stubSubsystem{name: "notifier-sub"}}
	_, err := New(Options{Subsystems: []Subsystem{sub}})
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	if !sub.notifierSet {
		t.Fatal("expected notifier to be set")
	}
	if !sub.sawNotifierAtRegistration {
		t.Fatal("expected notifier to be available before RegisterTools ran")
	}
}

func TestSubsystemShutdown_Good(t *testing.T) {
	sub := &shutdownSubsystem{stubSubsystem: stubSubsystem{name: "shutdown-sub"}}
	svc, err := New(Options{Subsystems: []Subsystem{sub}})
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	if err := svc.Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown() failed: %v", err)
	}
	if !sub.shutdownCalled {
		t.Error("expected Shutdown to have been called")
	}
}

func TestSubsystemShutdown_Bad_Error(t *testing.T) {
	sub := &shutdownSubsystem{
		stubSubsystem: stubSubsystem{name: "fail-sub"},
		shutdownErr:   context.DeadlineExceeded,
	}
	svc, err := New(Options{Subsystems: []Subsystem{sub}})
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	err = svc.Shutdown(context.Background())
	if err == nil {
		t.Fatal("expected error from Shutdown")
	}
}

func TestSubsystemShutdown_Good_NoShutdownInterface(t *testing.T) {
	sub := &stubSubsystem{name: "plain-sub"}
	svc, err := New(Options{Subsystems: []Subsystem{sub}})
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	if err := svc.Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown() should succeed for non-shutdown subsystem: %v", err)
	}
}
