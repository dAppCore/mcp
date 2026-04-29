package api

import core "dappco.re/go"

func ExampleOK() {
	_ = OK
	core.Println("OK")
	// Output: OK
}

func ExampleFail() {
	_ = Fail
	core.Println("Fail")
	// Output: Fail
}

func ExampleNewToolBridge() {
	_ = NewToolBridge
	core.Println("NewToolBridge")
	// Output: NewToolBridge
}

func ExampleToolBridge_Name() {
	var subject ToolBridge
	_ = subject.Name
	core.Println("ToolBridge.Name")
	// Output: ToolBridge.Name
}

func ExampleToolBridge_BasePath() {
	var subject ToolBridge
	_ = subject.BasePath
	core.Println("ToolBridge.BasePath")
	// Output: ToolBridge.BasePath
}

func ExampleToolBridge_Add() {
	var subject ToolBridge
	_ = subject.Add
	core.Println("ToolBridge.Add")
	// Output: ToolBridge.Add
}

func ExampleToolBridge_Tools() {
	var subject ToolBridge
	_ = subject.Tools
	core.Println("ToolBridge.Tools")
	// Output: ToolBridge.Tools
}

func ExampleToolBridge_RegisterRoutes() {
	var subject ToolBridge
	_ = subject.RegisterRoutes
	core.Println("ToolBridge.RegisterRoutes")
	// Output: ToolBridge.RegisterRoutes
}

func ExampleToolBridge_Describe() {
	var subject ToolBridge
	_ = subject.Describe
	core.Println("ToolBridge.Describe")
	// Output: ToolBridge.Describe
}

func ExampleWithSwagger() {
	_ = WithSwagger
	core.Println("WithSwagger")
	// Output: WithSwagger
}

func ExampleNew() {
	_ = New
	core.Println("New")
	// Output: New
}

func ExampleEngine_Register() {
	var subject Engine
	_ = subject.Register
	core.Println("Engine.Register")
	// Output: Engine.Register
}

func ExampleEngine_Handler() {
	var subject Engine
	_ = subject.Handler
	core.Println("Engine.Handler")
	// Output: Engine.Handler
}
