package agentic

import core "dappco.re/go"

func ExampleWorkspaceStatus() {
	var _ WorkspaceStatus
	core.Println("WorkspaceStatus")
	// Output: WorkspaceStatus
}

func ExampleStatusInput() {
	var _ StatusInput
	core.Println("StatusInput")
	// Output: StatusInput
}

func ExampleStatusOutput() {
	var _ StatusOutput
	core.Println("StatusOutput")
	// Output: StatusOutput
}

func ExampleWorkspaceInfo() {
	var _ WorkspaceInfo
	core.Println("WorkspaceInfo")
	// Output: WorkspaceInfo
}
