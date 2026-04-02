// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"strings"
	"testing"
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
	if !strings.Contains(strings.ToLower(err.Error()), "repo") {
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
	if !strings.Contains(strings.ToLower(err.Error()), "persona") {
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
	if !strings.Contains(strings.ToLower(err.Error()), "plan_template") {
		t.Fatalf("expected plan template error, got %q", err)
	}
}

func TestSetNotifier_Good_EmitsChannelEvents(t *testing.T) {
	s := NewPrep()
	notifier := &recordingNotifier{}
	s.SetNotifier(notifier)

	s.emitChannel(context.Background(), "agent.status", map[string]any{"status": "running"})

	if notifier.channel != "agent.status" {
		t.Fatalf("expected agent.status channel, got %q", notifier.channel)
	}
	if payload, ok := notifier.data.(map[string]any); !ok || payload["status"] != "running" {
		t.Fatalf("expected payload to include running status, got %#v", notifier.data)
	}
}
