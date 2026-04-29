package process

import core "dappco.re/go"

func ExampleBytes_Len() {
	var subject Bytes
	_ = subject.Len
	core.Println("Bytes.Len")
	// Output: Bytes.Len
}

func ExampleBuffer_Write() {
	var subject Buffer
	_ = subject.Write
	core.Println("Buffer.Write")
	// Output: Buffer.Write
}

func ExampleBuffer_String() {
	var subject Buffer
	_ = subject.String
	core.Println("Buffer.String")
	// Output: Buffer.String
}

func ExampleProcess_Info() {
	var subject Process
	_ = subject.Info
	core.Println("Process.Info")
	// Output: Process.Info
}

func ExampleProcess_Done() {
	var subject Process
	_ = subject.Done
	core.Println("Process.Done")
	// Output: Process.Done
}

func ExampleProcess_Output() {
	var subject Process
	_ = subject.Output
	core.Println("Process.Output")
	// Output: Process.Output
}

func ExampleProcess_Shutdown() {
	var subject Process
	_ = subject.Shutdown
	core.Println("Process.Shutdown")
	// Output: Process.Shutdown
}

func ExampleProcess_SendInput() {
	var subject Process
	_ = subject.SendInput
	core.Println("Process.SendInput")
	// Output: Process.SendInput
}

func ExampleNewService() {
	_ = NewService
	core.Println("NewService")
	// Output: NewService
}

func ExampleService_OnStartup() {
	var subject Service
	_ = subject.OnStartup
	core.Println("Service.OnStartup")
	// Output: Service.OnStartup
}

func ExampleService_OnShutdown() {
	var subject Service
	_ = subject.OnShutdown
	core.Println("Service.OnShutdown")
	// Output: Service.OnShutdown
}

func ExampleService_StartWithOptions() {
	var subject Service
	_ = subject.StartWithOptions
	core.Println("Service.StartWithOptions")
	// Output: Service.StartWithOptions
}

func ExampleService_Get() {
	var subject Service
	_ = subject.Get
	core.Println("Service.Get")
	// Output: Service.Get
}

func ExampleService_List() {
	var subject Service
	_ = subject.List
	core.Println("Service.List")
	// Output: Service.List
}

func ExampleService_Running() {
	var subject Service
	_ = subject.Running
	core.Println("Service.Running")
	// Output: Service.Running
}

func ExampleService_Kill() {
	var subject Service
	_ = subject.Kill
	core.Println("Service.Kill")
	// Output: Service.Kill
}

func ExampleService_Output() {
	var subject Service
	_ = subject.Output
	core.Println("Service.Output")
	// Output: Service.Output
}
