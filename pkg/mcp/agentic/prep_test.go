// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"testing"

	. "dappco.re/go"
	coremcp "dappco.re/go/mcp/pkg/mcp"
)

type recordingNotifier struct {
	channel string
	data    any
}

func (r *recordingNotifier) ChannelSend(_ context.Context, channel string, data any) {
	r.channel = channel
	r.data = data
}

func TestSanitizeRepoPathSegment_Good(t *testing.T) {
	t.Run("repo", func(t *testing.T) {
		value, err := sanitizeRepoPathSegment("go-io", "repo", false)
		if err != nil {
			t.Fatalf("expected valid repo name, got error: %v", err)
		}
		if value != "go-io" {
			t.Fatalf("expected normalized value, got: %q", value)
		}
	})

	t.Run("persona", func(t *testing.T) {
		value, err := sanitizeRepoPathSegment("engineering/backend-architect", "persona", true)
		if err != nil {
			t.Fatalf("expected valid persona path, got error: %v", err)
		}
		if value != "engineering/backend-architect" {
			t.Fatalf("expected persona path, got: %q", value)
		}
	})
}

func TestSanitizeRepoPathSegment_Bad(t *testing.T) {
	cases := []struct {
		name      string
		value     string
		allowPath bool
	}{
		{"repo segment traversal", "../repo", false},
		{"repo nested path", "team/repo", false},
		{"plan template traversal", "../secret", false},
		{"persona traversal", "engineering/../../admin", true},
		{"backslash", "org\\repo", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := sanitizeRepoPathSegment(tc.value, tc.name, tc.allowPath)
			if err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestPrepWorkspace_Bad_BadRepoTraversal(t *testing.T) {
	s := &PrepSubsystem{codePath: t.TempDir()}

	_, _, err := s.prepWorkspace(context.Background(), nil, PrepInput{Repo: "../repo"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !Contains(Lower(err.Error()), "repo") {
		t.Fatalf("expected repo error, got %q", err)
	}
}

func TestPrepWorkspace_Bad_BadPersonaTraversal(t *testing.T) {
	s := &PrepSubsystem{codePath: t.TempDir()}

	_, _, err := s.prepWorkspace(context.Background(), nil, PrepInput{
		Repo:    "repo",
		Persona: "engineering/../../admin",
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !Contains(Lower(err.Error()), "persona") {
		t.Fatalf("expected persona error, got %q", err)
	}
}

func TestPrepWorkspace_Bad_BadPlanTemplateTraversal(t *testing.T) {
	s := &PrepSubsystem{codePath: t.TempDir()}

	_, _, err := s.prepWorkspace(context.Background(), nil, PrepInput{
		Repo:         "repo",
		PlanTemplate: "../secret",
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !Contains(Lower(err.Error()), "plan_template") {
		t.Fatalf("expected plan template error, got %q", err)
	}
}

func TestSetNotifier_Good_EmitsChannelEvents(t *testing.T) {
	s := NewPrep()
	notifier := &recordingNotifier{}
	s.SetNotifier(notifier)

	s.emitChannel(context.Background(), coremcp.ChannelAgentStatus, map[string]any{"status": "running"})

	if notifier.channel != coremcp.ChannelAgentStatus {
		t.Fatalf("expected %s channel, got %q", coremcp.ChannelAgentStatus, notifier.channel)
	}
	if payload, ok := notifier.data.(map[string]any); !ok || payload["status"] != "running" {
		t.Fatalf("expected payload to include running status, got %#v", notifier.data)
	}
}

func TestEmitHarvestComplete_Good_EmitsChannelEvents(t *testing.T) {
	s := NewPrep()
	notifier := &recordingNotifier{}
	s.SetNotifier(notifier)

	s.emitHarvestComplete(context.Background(), "go-io-123", "go-io", 4, true)

	if notifier.channel != coremcp.ChannelHarvestComplete {
		t.Fatalf("expected %s channel, got %q", coremcp.ChannelHarvestComplete, notifier.channel)
	}
	payload, ok := notifier.data.(map[string]any)
	if !ok {
		t.Fatalf("expected payload object, got %#v", notifier.data)
	}
	if payload["workspace"] != "go-io-123" {
		t.Fatalf("expected workspace go-io-123, got %#v", payload["workspace"])
	}
	if payload["repo"] != "go-io" {
		t.Fatalf("expected repo go-io, got %#v", payload["repo"])
	}
	if payload["findings"] != 4 {
		t.Fatalf("expected findings 4, got %#v", payload["findings"])
	}
	if payload["issue_created"] != true {
		t.Fatalf("expected issue_created true, got %#v", payload["issue_created"])
	}
}

// moved AX-7 triplet TestPrep_NewPrep_Good
func TestPrep_NewPrep_Good(t *T) {
	t.Setenv("HOME", t.TempDir())
	sub := NewPrep()
	AssertNotNil(t, sub)
	AssertEqual(t, "agentic", sub.Name())
}

// moved AX-7 triplet TestPrep_NewPrep_Bad
func TestPrep_NewPrep_Bad(t *T) {
	t.Setenv("HOME", "")
	sub := NewPrep()
	AssertNotNil(t, sub)
	AssertNil(t, sub.notifier)
}

// moved AX-7 triplet TestPrep_NewPrep_Ugly
func TestPrep_NewPrep_Ugly(t *T) {
	t.Setenv("HOME", t.TempDir())
	first := NewPrep()
	second := NewPrep()
	AssertNotNil(t, first)
	AssertNotNil(t, second)
}

// moved AX-7 triplet TestPrep_PrepSubsystem_Name_Good
func TestPrep_PrepSubsystem_Name_Good(t *T) {
	sub := &PrepSubsystem{}
	AssertEqual(t, "agentic", sub.Name())
	AssertNil(t, sub.notifier)
}

// moved AX-7 triplet TestPrep_PrepSubsystem_Name_Bad
func TestPrep_PrepSubsystem_Name_Bad(t *T) {
	var sub *PrepSubsystem
	AssertEqual(t, "agentic", sub.Name())
	AssertNil(t, sub)
}

// moved AX-7 triplet TestPrep_PrepSubsystem_Name_Ugly
func TestPrep_PrepSubsystem_Name_Ugly(t *T) {
	sub := NewPrep()
	AssertEqual(t, "agentic", sub.Name())
	AssertNil(t, sub.notifier)
}

// moved AX-7 triplet TestPrep_PrepSubsystem_RegisterTools_Good
func TestPrep_PrepSubsystem_RegisterTools_Good(t *T) {
	svc := agenticMCPServiceForTest(t)
	sub := NewPrep()
	sub.RegisterTools(svc)
	AssertTrue(t, len(svc.Tools()) > 0)
}

// moved AX-7 triplet TestPrep_PrepSubsystem_RegisterTools_Bad
func TestPrep_PrepSubsystem_RegisterTools_Bad(t *T) {
	sub := NewPrep()
	AssertPanics(t, func() { sub.RegisterTools(nil) })
	AssertEqual(t, "agentic", sub.Name())
}

// moved AX-7 triplet TestPrep_PrepSubsystem_RegisterTools_Ugly
func TestPrep_PrepSubsystem_RegisterTools_Ugly(t *T) {
	svc := agenticMCPServiceForTest(t)
	sub := &PrepSubsystem{}
	sub.RegisterTools(svc)
	AssertTrue(t, len(svc.Tools()) > 0)
}

// moved AX-7 triplet TestPrep_PrepSubsystem_SetNotifier_Good
func TestPrep_PrepSubsystem_SetNotifier_Good(t *T) {
	sub := NewPrep()
	svc := agenticMCPServiceForTest(t)
	sub.SetNotifier(svc)
	AssertEqual(t, svc, sub.notifier)
}

// moved AX-7 triplet TestPrep_PrepSubsystem_SetNotifier_Bad
func TestPrep_PrepSubsystem_SetNotifier_Bad(t *T) {
	sub := NewPrep()
	sub.SetNotifier(nil)
	AssertNil(t, sub.notifier)
}

// moved AX-7 triplet TestPrep_PrepSubsystem_SetNotifier_Ugly
func TestPrep_PrepSubsystem_SetNotifier_Ugly(t *T) {
	sub := &PrepSubsystem{}
	sub.SetNotifier(agenticMCPServiceForTest(t))
	AssertNotNil(t, sub.notifier)
}

// moved AX-7 triplet TestPrep_PrepSubsystem_Shutdown_Good
func TestPrep_PrepSubsystem_Shutdown_Good(t *T) {
	sub := NewPrep()
	err := sub.Shutdown(context.Background())
	AssertNoError(t, err)
}

// moved AX-7 triplet TestPrep_PrepSubsystem_Shutdown_Bad
func TestPrep_PrepSubsystem_Shutdown_Bad(t *T) {
	sub := NewPrep()
	err := sub.Shutdown(nil)
	AssertNoError(t, err)
}

// moved AX-7 triplet TestPrep_PrepSubsystem_Shutdown_Ugly
func TestPrep_PrepSubsystem_Shutdown_Ugly(t *T) {
	var sub *PrepSubsystem
	err := sub.Shutdown(context.Background())
	AssertNoError(t, err)
}
