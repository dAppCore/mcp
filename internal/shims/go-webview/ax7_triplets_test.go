package webview

import (
	"time"

	core "dappco.re/go"
)

type T = core.T

var (
	AssertContains = core.AssertContains
	AssertEqual    = core.AssertEqual
	AssertError    = core.AssertError
	AssertLen      = core.AssertLen
	AssertNil      = core.AssertNil
	AssertNoError  = core.AssertNoError
	AssertPanics   = core.AssertPanics
)

func TestAX7_WithDebugURL_Good(t *T) {
	w := &Webview{}
	WithDebugURL("ws://devtools")(w)
	AssertEqual(t, "ws://devtools", w.debugURL)
}

func TestAX7_WithDebugURL_Bad(t *T) {
	var optTarget *Webview
	AssertNil(t, optTarget)
	AssertPanics(t, func() { WithDebugURL("ws://devtools")(optTarget) })
}

func TestAX7_WithDebugURL_Ugly(t *T) {
	w := &Webview{debugURL: "old"}
	WithDebugURL("")(w)
	AssertEqual(t, "", w.debugURL)
}

func TestAX7_WithTimeout_Good(t *T) {
	w := &Webview{}
	WithTimeout(time.Second)(w)
	AssertEqual(t, time.Second, w.timeout)
}

func TestAX7_WithTimeout_Bad(t *T) {
	var optTarget *Webview
	AssertNil(t, optTarget)
	AssertPanics(t, func() { WithTimeout(time.Second)(optTarget) })
}

func TestAX7_WithTimeout_Ugly(t *T) {
	w := &Webview{timeout: time.Second}
	WithTimeout(0)(w)
	AssertEqual(t, time.Duration(0), w.timeout)
}

func TestAX7_New_Good(t *T) {
	w, err := New(WithDebugURL("ws://devtools"), WithTimeout(time.Second))
	AssertNoError(t, err)
	AssertEqual(t, "ws://devtools", w.debugURL)
	AssertEqual(t, time.Second, w.timeout)
}

func TestAX7_New_Bad(t *T) {
	w, err := New(nil)
	AssertNoError(t, err)
	AssertEqual(t, 30*time.Second, w.timeout)
}

func TestAX7_New_Ugly(t *T) {
	w, err := New(nil)
	AssertNoError(t, err)
	AssertEqual(t, 30*time.Second, w.timeout)
}

func TestAX7_Webview_Close_Good(t *T) {
	w := &Webview{}
	AssertNoError(t, w.Close())
	AssertLen(t, w.console, 0)
}

func TestAX7_Webview_Close_Bad(t *T) {
	w := &Webview{console: []ConsoleMessage{{Text: "kept"}}}
	AssertNoError(t, w.Close())
	AssertLen(t, w.console, 1)
}

func TestAX7_Webview_Close_Ugly(t *T) {
	var w *Webview
	AssertNoError(t, w.Close())
	AssertNil(t, w)
}

func TestAX7_Webview_Navigate_Good(t *T) {
	w := &Webview{}
	err := w.Navigate("https://example.test")
	AssertError(t, err)
	AssertContains(t, err.Error(), "backend unavailable")
}

func TestAX7_Webview_Navigate_Bad(t *T) {
	w := &Webview{}
	AssertError(t, w.Navigate("https://example.test"))
	AssertLen(t, w.console, 0)
}

func TestAX7_Webview_Navigate_Ugly(t *T) {
	w := &Webview{}
	AssertError(t, w.Navigate(""))
	AssertLen(t, w.console, 0)
}

func TestAX7_Webview_Click_Good(t *T) {
	w := &Webview{}
	err := w.Click("#submit")
	AssertError(t, err)
	AssertContains(t, err.Error(), "backend unavailable")
}

func TestAX7_Webview_Click_Bad(t *T) {
	w := &Webview{}
	AssertError(t, w.Click("#missing"))
	AssertLen(t, w.console, 0)
}

func TestAX7_Webview_Click_Ugly(t *T) {
	w := &Webview{}
	AssertError(t, w.Click(""))
	AssertLen(t, w.console, 0)
}

func TestAX7_Webview_Type_Good(t *T) {
	w := &Webview{}
	err := w.Type("#input", "value")
	AssertError(t, err)
	AssertContains(t, err.Error(), "backend unavailable")
}

func TestAX7_Webview_Type_Bad(t *T) {
	w := &Webview{}
	AssertError(t, w.Type("#input", "value"))
	AssertLen(t, w.console, 0)
}

func TestAX7_Webview_Type_Ugly(t *T) {
	w := &Webview{}
	AssertError(t, w.Type("", ""))
	AssertLen(t, w.console, 0)
}

func TestAX7_Webview_QuerySelector_Good(t *T) {
	w := &Webview{}
	el, err := w.QuerySelector("#app")
	AssertError(t, err)
	AssertNil(t, el)
}

func TestAX7_Webview_QuerySelector_Bad(t *T) {
	w := &Webview{}
	el, err := w.QuerySelector("#missing")
	AssertError(t, err)
	AssertNil(t, el)
}

func TestAX7_Webview_QuerySelector_Ugly(t *T) {
	w := &Webview{}
	el, err := w.QuerySelector("")
	AssertError(t, err)
	AssertNil(t, el)
}

func TestAX7_Webview_QuerySelectorAll_Good(t *T) {
	w := &Webview{}
	els, err := w.QuerySelectorAll(".item")
	AssertError(t, err)
	AssertNil(t, els)
}

func TestAX7_Webview_QuerySelectorAll_Bad(t *T) {
	w := &Webview{}
	els, err := w.QuerySelectorAll(".missing")
	AssertError(t, err)
	AssertNil(t, els)
}

func TestAX7_Webview_QuerySelectorAll_Ugly(t *T) {
	w := &Webview{}
	els, err := w.QuerySelectorAll("")
	AssertError(t, err)
	AssertNil(t, els)
}

func TestAX7_Webview_GetConsole_Good(t *T) {
	w := &Webview{console: []ConsoleMessage{{Type: "log", Text: "hello"}}}
	got := w.GetConsole()
	AssertLen(t, got, 1)
	got[0].Text = "mutated"
	AssertEqual(t, "hello", w.console[0].Text)
}

func TestAX7_Webview_GetConsole_Bad(t *T) {
	var w *Webview
	AssertNil(t, w)
	AssertPanics(t, func() { _ = w.GetConsole() })
}

func TestAX7_Webview_GetConsole_Ugly(t *T) {
	w := &Webview{}
	AssertLen(t, w.GetConsole(), 0)
	AssertLen(t, w.console, 0)
}

func TestAX7_Webview_ClearConsole_Good(t *T) {
	w := &Webview{console: []ConsoleMessage{{Text: "hello"}}}
	w.ClearConsole()
	AssertLen(t, w.console, 0)
}

func TestAX7_Webview_ClearConsole_Bad(t *T) {
	var w *Webview
	AssertNil(t, w)
	AssertPanics(t, func() { w.ClearConsole() })
}

func TestAX7_Webview_ClearConsole_Ugly(t *T) {
	w := &Webview{}
	w.ClearConsole()
	AssertLen(t, w.console, 0)
}

func TestAX7_Webview_Evaluate_Good(t *T) {
	w := &Webview{}
	got, err := w.Evaluate("1 + 1")
	AssertError(t, err)
	AssertNil(t, got)
}

func TestAX7_Webview_Evaluate_Bad(t *T) {
	w := &Webview{}
	got, err := w.Evaluate("1 + 1")
	AssertError(t, err)
	AssertNil(t, got)
}

func TestAX7_Webview_Evaluate_Ugly(t *T) {
	w := &Webview{}
	got, err := w.Evaluate("")
	AssertError(t, err)
	AssertNil(t, got)
}

func TestAX7_Webview_Screenshot_Good(t *T) {
	w := &Webview{}
	got, err := w.Screenshot()
	AssertError(t, err)
	AssertNil(t, got)
}

func TestAX7_Webview_Screenshot_Bad(t *T) {
	w := &Webview{}
	got, err := w.Screenshot()
	AssertError(t, err)
	AssertNil(t, got)
}

func TestAX7_Webview_Screenshot_Ugly(t *T) {
	var w *Webview
	got, err := w.Screenshot()
	AssertError(t, err)
	AssertNil(t, got)
}
