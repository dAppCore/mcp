package mcp

import core "dappco.re/go"

type Medium = coreMedium

func ExampleMedium_Read() {
	var subject Medium
	_ = subject.Read
	core.Println("Medium.Read")
	// Output: Medium.Read
}

func ExampleMedium_Write() {
	var subject Medium
	_ = subject.Write
	core.Println("Medium.Write")
	// Output: Medium.Write
}

func ExampleMedium_WriteMode() {
	var subject Medium
	_ = subject.WriteMode
	core.Println("Medium.WriteMode")
	// Output: Medium.WriteMode
}

func ExampleMedium_EnsureDir() {
	var subject Medium
	_ = subject.EnsureDir
	core.Println("Medium.EnsureDir")
	// Output: Medium.EnsureDir
}

func ExampleMedium_IsFile() {
	var subject Medium
	_ = subject.IsFile
	core.Println("Medium.IsFile")
	// Output: Medium.IsFile
}

func ExampleMedium_Delete() {
	var subject Medium
	_ = subject.Delete
	core.Println("Medium.Delete")
	// Output: Medium.Delete
}

func ExampleMedium_DeleteAll() {
	var subject Medium
	_ = subject.DeleteAll
	core.Println("Medium.DeleteAll")
	// Output: Medium.DeleteAll
}

func ExampleMedium_Rename() {
	var subject Medium
	_ = subject.Rename
	core.Println("Medium.Rename")
	// Output: Medium.Rename
}

func ExampleMedium_List() {
	var subject Medium
	_ = subject.List
	core.Println("Medium.List")
	// Output: Medium.List
}

func ExampleMedium_Stat() {
	var subject Medium
	_ = subject.Stat
	core.Println("Medium.Stat")
	// Output: Medium.Stat
}

func ExampleMedium_Open() {
	var subject Medium
	_ = subject.Open
	core.Println("Medium.Open")
	// Output: Medium.Open
}

func ExampleMedium_Create() {
	var subject Medium
	_ = subject.Create
	core.Println("Medium.Create")
	// Output: Medium.Create
}

func ExampleMedium_Append() {
	var subject Medium
	_ = subject.Append
	core.Println("Medium.Append")
	// Output: Medium.Append
}

func ExampleMedium_ReadStream() {
	var subject Medium
	_ = subject.ReadStream
	core.Println("Medium.ReadStream")
	// Output: Medium.ReadStream
}

func ExampleMedium_WriteStream() {
	var subject Medium
	_ = subject.WriteStream
	core.Println("Medium.WriteStream")
	// Output: Medium.WriteStream
}

func ExampleMedium_Exists() {
	var subject Medium
	_ = subject.Exists
	core.Println("Medium.Exists")
	// Output: Medium.Exists
}

func ExampleMedium_IsDir() {
	var subject Medium
	_ = subject.IsDir
	core.Println("Medium.IsDir")
	// Output: Medium.IsDir
}
