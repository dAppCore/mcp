package mcp

import core "dappco.re/go"

type InputError = restInputError

func ExampleInputError_Error() {
	var subject InputError
	_ = subject.Error
	core.Println("InputError.Error")
	// Output: InputError.Error
}

func ExampleInputError_Unwrap() {
	var subject InputError
	_ = subject.Unwrap
	core.Println("InputError.Unwrap")
	// Output: InputError.Unwrap
}

func ExampleInputError_Is() {
	var subject InputError
	_ = subject.Is
	core.Println("InputError.Is")
	// Output: InputError.Is
}

func ExampleAddToolRecorded() {
	_ = AddToolRecorded[ReadFileInput, ReadFileOutput]
	core.Println("AddToolRecorded")
	// Output: AddToolRecorded
}
