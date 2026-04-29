package agentic

import "testing"

func TestMirror_PublicTypes(t *testing.T) {
	input := MirrorInput{Repo: "go-mcp", MaxFiles: 50}
	sync := MirrorSync{Repo: input.Repo, CommitsAhead: 2, FilesChanged: 3, Pushed: true}
	out := MirrorOutput{Success: true, Synced: []MirrorSync{sync}, Count: 1}
	if out.Synced[0].Repo != "go-mcp" || out.Count != len(out.Synced) {
		t.Fatalf("unexpected mirror output: %+v", out)
	}
}
