package webview

import (
	"time"
)

// moved AX-7 triplet TestWebview_WithDebugURL_Good
func TestWebview_WithDebugURL_Good(t *T) {
	w := &Webview{}
	WithDebugURL("ws://devtools")(w)
	AssertEqual(t, "ws://devtools", w.debugURL)
}

// moved AX-7 triplet TestWebview_WithDebugURL_Bad
func TestWebview_WithDebugURL_Bad(t *T) {
	var optTarget *Webview
	AssertNil(t, optTarget)
	AssertPanics(t, func() { WithDebugURL("ws://devtools")(optTarget) })
}

// moved AX-7 triplet TestWebview_WithDebugURL_Ugly
func TestWebview_WithDebugURL_Ugly(t *T) {
	w := &Webview{debugURL: "old"}
	WithDebugURL("")(w)
	AssertEqual(t, "", w.debugURL)
}

// moved AX-7 triplet TestWebview_WithTimeout_Good
func TestWebview_WithTimeout_Good(t *T) {
	w := &Webview{}
	WithTimeout(time.Second)(w)
	AssertEqual(t, time.Second, w.timeout)
}

// moved AX-7 triplet TestWebview_WithTimeout_Bad
func TestWebview_WithTimeout_Bad(t *T) {
	var optTarget *Webview
	AssertNil(t, optTarget)
	AssertPanics(t, func() { WithTimeout(time.Second)(optTarget) })
}

// moved AX-7 triplet TestWebview_WithTimeout_Ugly
func TestWebview_WithTimeout_Ugly(t *T) {
	w := &Webview{timeout: time.Second}
	WithTimeout(0)(w)
	AssertEqual(t, time.Duration(0), w.timeout)
}

// moved AX-7 triplet TestWebview_New_Good
func TestWebview_New_Good(t *T) {
	w, err := New(WithDebugURL("ws://devtools"), WithTimeout(time.Second))
	AssertNoError(t, err)
	AssertEqual(t, "ws://devtools", w.debugURL)
	AssertEqual(t, time.Second, w.timeout)
}

// moved AX-7 triplet TestWebview_New_Bad
func TestWebview_New_Bad(t *T) {
	w, err := New(nil)
	AssertNoError(t, err)
	AssertEqual(t, 30*time.Second, w.timeout)
}

// moved AX-7 triplet TestWebview_New_Ugly
func TestWebview_New_Ugly(t *T) {
	w, err := New(WithDebugURL(""))
	AssertNoError(t, err)
	AssertEqual(t, "", w.debugURL)
	AssertEqual(t, 30*time.Second, w.timeout)
}

// moved AX-7 triplet TestWebview_Webview_Close_Good
func TestWebview_Webview_Close_Good(t *T) {
	w := &Webview{}
	AssertNoError(t, w.Close())
	AssertLen(t, w.console, 0)
}

// moved AX-7 triplet TestWebview_Webview_Close_Bad
func TestWebview_Webview_Close_Bad(t *T) {
	w := &Webview{console: []ConsoleMessage{{Text: "kept"}}}
	AssertNoError(t, w.Close())
	AssertLen(t, w.console, 1)
}

// moved AX-7 triplet TestWebview_Webview_Close_Ugly
func TestWebview_Webview_Close_Ugly(t *T) {
	var w *Webview
	AssertNoError(t, w.Close())
	AssertNil(t, w)
}

// moved AX-7 triplet TestWebview_Webview_Navigate_Good
func TestWebview_Webview_Navigate_Good(t *T) {
	w := &Webview{}
	err := w.Navigate("https://example.test")
	AssertError(t, err)
	AssertContains(t, err.Error(), "backend unavailable")
}

// moved AX-7 triplet TestWebview_Webview_Navigate_Bad
func TestWebview_Webview_Navigate_Bad(t *T) {
	w := &Webview{}
	AssertError(t, w.Navigate("https://example.test"))
	AssertLen(t, w.console, 0)
}

// moved AX-7 triplet TestWebview_Webview_Navigate_Ugly
func TestWebview_Webview_Navigate_Ugly(t *T) {
	w := &Webview{}
	AssertError(t, w.Navigate(""))
	AssertLen(t, w.console, 0)
}

// moved AX-7 triplet TestWebview_Webview_Click_Good
func TestWebview_Webview_Click_Good(t *T) {
	w := &Webview{}
	err := w.Click("#submit")
	AssertError(t, err)
	AssertContains(t, err.Error(), "backend unavailable")
}

// moved AX-7 triplet TestWebview_Webview_Click_Bad
func TestWebview_Webview_Click_Bad(t *T) {
	w := &Webview{}
	AssertError(t, w.Click("#missing"))
	AssertLen(t, w.console, 0)
}

// moved AX-7 triplet TestWebview_Webview_Click_Ugly
func TestWebview_Webview_Click_Ugly(t *T) {
	w := &Webview{}
	AssertError(t, w.Click(""))
	AssertLen(t, w.console, 0)
}

// moved AX-7 triplet TestWebview_Webview_Type_Good
func TestWebview_Webview_Type_Good(t *T) {
	w := &Webview{}
	err := w.Type("#input", "value")
	AssertError(t, err)
	AssertContains(t, err.Error(), "backend unavailable")
}

// moved AX-7 triplet TestWebview_Webview_Type_Bad
func TestWebview_Webview_Type_Bad(t *T) {
	w := &Webview{}
	AssertError(t, w.Type("#input", "value"))
	AssertLen(t, w.console, 0)
}

// moved AX-7 triplet TestWebview_Webview_Type_Ugly
func TestWebview_Webview_Type_Ugly(t *T) {
	w := &Webview{}
	AssertError(t, w.Type("", ""))
	AssertLen(t, w.console, 0)
}

// moved AX-7 triplet TestWebview_Webview_QuerySelector_Good
func TestWebview_Webview_QuerySelector_Good(t *T) {
	w := &Webview{}
	el, err := w.QuerySelector("#app")
	AssertError(t, err)
	AssertNil(t, el)
}

// moved AX-7 triplet TestWebview_Webview_QuerySelector_Bad
func TestWebview_Webview_QuerySelector_Bad(t *T) {
	w := &Webview{}
	el, err := w.QuerySelector("#missing")
	AssertError(t, err)
	AssertNil(t, el)
}

// moved AX-7 triplet TestWebview_Webview_QuerySelector_Ugly
func TestWebview_Webview_QuerySelector_Ugly(t *T) {
	w := &Webview{}
	el, err := w.QuerySelector("")
	AssertError(t, err)
	AssertNil(t, el)
}

// moved AX-7 triplet TestWebview_Webview_QuerySelectorAll_Good
func TestWebview_Webview_QuerySelectorAll_Good(t *T) {
	w := &Webview{}
	els, err := w.QuerySelectorAll(".item")
	AssertError(t, err)
	AssertNil(t, els)
}

// moved AX-7 triplet TestWebview_Webview_QuerySelectorAll_Bad
func TestWebview_Webview_QuerySelectorAll_Bad(t *T) {
	w := &Webview{}
	els, err := w.QuerySelectorAll(".missing")
	AssertError(t, err)
	AssertNil(t, els)
}

// moved AX-7 triplet TestWebview_Webview_QuerySelectorAll_Ugly
func TestWebview_Webview_QuerySelectorAll_Ugly(t *T) {
	w := &Webview{}
	els, err := w.QuerySelectorAll("")
	AssertError(t, err)
	AssertNil(t, els)
}

// moved AX-7 triplet TestWebview_Webview_GetConsole_Good
func TestWebview_Webview_GetConsole_Good(t *T) {
	w := &Webview{console: []ConsoleMessage{{Type: `log`, Text: "hello"}}}
	got := w.GetConsole()
	AssertLen(t, got, 1)
	got[0].Text = "mutated"
	AssertEqual(t, "hello", w.console[0].Text)
}

// moved AX-7 triplet TestWebview_Webview_GetConsole_Bad
func TestWebview_Webview_GetConsole_Bad(t *T) {
	var w *Webview
	AssertNil(t, w)
	AssertPanics(t, func() { _ = w.GetConsole() })
}

// moved AX-7 triplet TestWebview_Webview_GetConsole_Ugly
func TestWebview_Webview_GetConsole_Ugly(t *T) {
	w := &Webview{}
	AssertLen(t, w.GetConsole(), 0)
	AssertLen(t, w.console, 0)
}

// moved AX-7 triplet TestWebview_Webview_ClearConsole_Good
func TestWebview_Webview_ClearConsole_Good(t *T) {
	w := &Webview{console: []ConsoleMessage{{Text: "hello"}}}
	w.ClearConsole()
	AssertLen(t, w.console, 0)
}

// moved AX-7 triplet TestWebview_Webview_ClearConsole_Bad
func TestWebview_Webview_ClearConsole_Bad(t *T) {
	var w *Webview
	AssertNil(t, w)
	AssertPanics(t, func() { w.ClearConsole() })
}

// moved AX-7 triplet TestWebview_Webview_ClearConsole_Ugly
func TestWebview_Webview_ClearConsole_Ugly(t *T) {
	w := &Webview{}
	w.ClearConsole()
	AssertLen(t, w.console, 0)
}

// moved AX-7 triplet TestWebview_Webview_Evaluate_Good
func TestWebview_Webview_Evaluate_Good(t *T) {
	w := &Webview{}
	got, err := w.Evaluate("1 + 1")
	AssertError(t, err)
	AssertNil(t, got)
}

// moved AX-7 triplet TestWebview_Webview_Evaluate_Bad
func TestWebview_Webview_Evaluate_Bad(t *T) {
	var w *Webview
	AssertPanics(t, func() {
		_, _ = w.Evaluate("1 + 1")
	})
	AssertNil(t, w)
}

// moved AX-7 triplet TestWebview_Webview_Evaluate_Ugly
func TestWebview_Webview_Evaluate_Ugly(t *T) {
	w := &Webview{}
	got, err := w.Evaluate("")
	AssertError(t, err)
	AssertNil(t, got)
}

// moved AX-7 triplet TestWebview_Webview_Screenshot_Good
func TestWebview_Webview_Screenshot_Good(t *T) {
	w := &Webview{}
	got, err := w.Screenshot()
	AssertError(t, err)
	AssertNil(t, got)
}

// moved AX-7 triplet TestWebview_Webview_Screenshot_Bad
func TestWebview_Webview_Screenshot_Bad(t *T) {
	w := &Webview{debugURL: "ws://missing"}
	got, err := w.Screenshot()
	AssertError(t, err)
	AssertNil(t, got)
	AssertEqual(t, "ws://missing", w.debugURL)
}

// moved AX-7 triplet TestWebview_Webview_Screenshot_Ugly
func TestWebview_Webview_Screenshot_Ugly(t *T) {
	var w *Webview
	got, err := w.Screenshot()
	AssertError(t, err)
	AssertNil(t, got)
}
