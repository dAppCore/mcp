package webview

import core "dappco.re/go"

func ExampleWithDebugURL() {
	_ = WithDebugURL
	core.Println("WithDebugURL")
	// Output: WithDebugURL
}

func ExampleWithTimeout() {
	_ = WithTimeout
	core.Println("WithTimeout")
	// Output: WithTimeout
}

func ExampleNew() {
	_ = New
	core.Println("New")
	// Output: New
}

func ExampleWebview_Close() {
	var subject Webview
	_ = subject.Close
	core.Println("Webview.Close")
	// Output: Webview.Close
}

func ExampleWebview_Navigate() {
	var subject Webview
	_ = subject.Navigate
	core.Println("Webview.Navigate")
	// Output: Webview.Navigate
}

func ExampleWebview_Click() {
	var subject Webview
	_ = subject.Click
	core.Println("Webview.Click")
	// Output: Webview.Click
}

func ExampleWebview_Type() {
	var subject Webview
	_ = subject.Type
	core.Println("Webview.Type")
	// Output: Webview.Type
}

func ExampleWebview_QuerySelector() {
	var subject Webview
	_ = subject.QuerySelector
	core.Println("Webview.QuerySelector")
	// Output: Webview.QuerySelector
}

func ExampleWebview_QuerySelectorAll() {
	var subject Webview
	_ = subject.QuerySelectorAll
	core.Println("Webview.QuerySelectorAll")
	// Output: Webview.QuerySelectorAll
}

func ExampleWebview_GetConsole() {
	var subject Webview
	_ = subject.GetConsole
	core.Println("Webview.GetConsole")
	// Output: Webview.GetConsole
}

func ExampleWebview_ClearConsole() {
	var subject Webview
	_ = subject.ClearConsole
	core.Println("Webview.ClearConsole")
	// Output: Webview.ClearConsole
}

func ExampleWebview_Evaluate() {
	var subject Webview
	_ = subject.Evaluate
	core.Println("Webview.Evaluate")
	// Output: Webview.Evaluate
}

func ExampleWebview_Screenshot() {
	var subject Webview
	_ = subject.Screenshot
	core.Println("Webview.Screenshot")
	// Output: Webview.Screenshot
}
