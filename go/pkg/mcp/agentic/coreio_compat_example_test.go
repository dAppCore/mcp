package agentic

import core "dappco.re/go"

type CoreFS = localCoreFS

func ExampleCoreFS_Read() {
	var subject CoreFS
	_ = subject.Read
	core.Println("CoreFS.Read")
	// Output: CoreFS.Read
}

func ExampleCoreFS_EnsureDir() {
	var subject CoreFS
	_ = subject.EnsureDir
	core.Println("CoreFS.EnsureDir")
	// Output: CoreFS.EnsureDir
}

func ExampleCoreFS_List() {
	var subject CoreFS
	_ = subject.List
	core.Println("CoreFS.List")
	// Output: CoreFS.List
}

func ExampleCoreFS_IsFile() {
	var subject CoreFS
	_ = subject.IsFile
	core.Println("CoreFS.IsFile")
	// Output: CoreFS.IsFile
}

func ExampleCoreFS_Delete() {
	var subject CoreFS
	_ = subject.Delete
	core.Println("CoreFS.Delete")
	// Output: CoreFS.Delete
}
