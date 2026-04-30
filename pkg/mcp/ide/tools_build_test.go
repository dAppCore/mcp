package ide

import "testing"

func TestToolsBuild_PublicTypes(t *testing.T) {
	build := BuildInfo{ID: "build-1", Status: "passed", Branch: "main"}
	out := BuildStatusOutput{Build: build}
	logs := BuildLogsOutput{BuildID: "build-1", Lines: []string{"ok"}}
	if out.Build.ID != "build-1" || logs.Lines[0] != "ok" {
		t.Fatalf("unexpected build outputs: %+v %+v", out, logs)
	}
}
