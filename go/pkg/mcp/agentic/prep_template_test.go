// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"testing"

	core "dappco.re/go"
)

func TestPrep_writePromptTemplate_Good_KnownTemplates(t *testing.T) {
	sub := newPlanSub(t)
	cases := map[string]string{
		"conventions": "project conventions",
		"security":    "security issues",
		"coding":      "PERSONA.md",
	}
	for tmpl, marker := range cases {
		wsDir := core.Path(sub.codePath, "ws-"+tmpl)
		sub.writePromptTemplate(tmpl, wsDir)

		got, err := coreio.Local.Read(core.Path(wsDir, "src", "PROMPT.md"))
		if err != nil {
			t.Fatalf("read PROMPT.md for %q: %v", tmpl, err)
		}
		if !core.Contains(got, marker) {
			t.Fatalf("template %q PROMPT.md missing %q:\n%s", tmpl, marker, got)
		}
	}
}

func TestPrep_writePromptTemplate_Ugly_UnknownTemplateFallback(t *testing.T) {
	sub := newPlanSub(t)
	wsDir := core.Path(sub.codePath, "ws-unknown")
	sub.writePromptTemplate("does-not-exist", wsDir)

	got, err := coreio.Local.Read(core.Path(wsDir, "src", "PROMPT.md"))
	if err != nil {
		t.Fatalf("read PROMPT.md: %v", err)
	}
	if !core.Contains(got, "Read TODO.md") {
		t.Fatalf("expected default fallback prompt, got:\n%s", got)
	}
}

func TestPrep_writeAtomic_Good_CreatesParentDirs(t *testing.T) {
	root := t.TempDir()
	path := core.Path(root, "deep", "nested", "file.txt")

	if err := writeAtomic(path, "payload"); err != nil {
		t.Fatalf("writeAtomic: %v", err)
	}
	got, err := coreio.Local.Read(path)
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	if got != "payload" {
		t.Fatalf("expected payload, got %q", got)
	}
}

func TestPrep_writeAtomic_Good_Overwrite(t *testing.T) {
	root := t.TempDir()
	path := core.Path(root, "file.txt")

	if err := writeAtomic(path, "first"); err != nil {
		t.Fatalf("first write: %v", err)
	}
	if err := writeAtomic(path, "second"); err != nil {
		t.Fatalf("overwrite: %v", err)
	}
	got, err := coreio.Local.Read(path)
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	if got != "second" {
		t.Fatalf("expected overwrite to second, got %q", got)
	}
}

func TestPrep_writeAtomic_Ugly_DirCollision(t *testing.T) {
	root := t.TempDir()
	// A file exists where writeAtomic wants a parent directory.
	blocker := core.Path(root, "blocker")
	if err := writeAtomic(blocker, "x"); err != nil {
		t.Fatalf("seed blocker: %v", err)
	}
	// Now try to write under the blocker as if it were a directory.
	if err := writeAtomic(core.Path(blocker, "child.txt"), "y"); err == nil {
		t.Fatal("expected error writing under a file-as-directory")
	}
}
