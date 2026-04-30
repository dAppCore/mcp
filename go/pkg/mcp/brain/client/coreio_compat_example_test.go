package client

import core "dappco.re/go"

type CoreFS = localCoreFS

func ExampleCoreFS_Stat() {
	var subject CoreFS
	_ = subject.Stat
	core.Println("CoreFS.Stat")
	// Output: CoreFS.Stat
}

func ExampleCoreFS_Read() {
	var subject CoreFS
	_ = subject.Read
	core.Println("CoreFS.Read")
	// Output: CoreFS.Read
}

func ExampleCoreFS_WriteMode() {
	var subject CoreFS
	_ = subject.WriteMode
	core.Println("CoreFS.WriteMode")
	// Output: CoreFS.WriteMode
}
