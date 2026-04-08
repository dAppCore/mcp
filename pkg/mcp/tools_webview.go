// SPDX-License-Identifier: EUPL-1.2

package mcp

import (
	"bytes"
	"context"
	"encoding/base64"
	"image"
	"image/jpeg"
	_ "image/png"
	"sync"
	"time"

	core "dappco.re/go/core"
	"dappco.re/go/core/log"
	"dappco.re/go/core/webview"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// webviewMu protects webviewInstance from concurrent access.
var webviewMu sync.Mutex

// webviewInstance holds the current webview connection.
// This is managed by the MCP service.
var webviewInstance *webview.Webview

// Sentinel errors for webview tools.
var (
	errNotConnected     = log.E("webview", "not connected; use webview_connect first", nil)
	errSelectorRequired = log.E("webview", "selector is required", nil)
)

// closeWebviewConnection closes and clears the shared browser connection.
func closeWebviewConnection() error {
	webviewMu.Lock()
	defer webviewMu.Unlock()

	if webviewInstance == nil {
		return nil
	}

	err := webviewInstance.Close()
	webviewInstance = nil
	return err
}

// WebviewConnectInput contains parameters for connecting to Chrome DevTools.
//
//	input := WebviewConnectInput{DebugURL: "http://localhost:9222", Timeout: 10}
type WebviewConnectInput struct {
	DebugURL string `json:"debug_url"`         // e.g. "http://localhost:9222"
	Timeout  int    `json:"timeout,omitempty"` // seconds (default: 30)
}

// WebviewConnectOutput contains the result of connecting to Chrome.
//
//	// out.Success == true, out.Message == "Connected to Chrome DevTools at http://localhost:9222"
type WebviewConnectOutput struct {
	Success bool   `json:"success"`           // true when connection established
	Message string `json:"message,omitempty"` // connection status
}

// WebviewNavigateInput contains parameters for navigating to a URL.
//
//	input := WebviewNavigateInput{URL: "https://lthn.ai/dashboard"}
type WebviewNavigateInput struct {
	URL string `json:"url"` // e.g. "https://lthn.ai/dashboard"
}

// WebviewNavigateOutput contains the result of navigation.
//
//	// out.Success == true, out.URL == "https://lthn.ai/dashboard"
type WebviewNavigateOutput struct {
	Success bool   `json:"success"` // true when navigation completed
	URL     string `json:"url"`     // the URL navigated to
}

// WebviewClickInput contains parameters for clicking an element.
//
//	input := WebviewClickInput{Selector: "button.submit"}
type WebviewClickInput struct {
	Selector string `json:"selector"` // e.g. "button.submit"
}

// WebviewClickOutput contains the result of a click action.
//
//	// out.Success == true
type WebviewClickOutput struct {
	Success bool `json:"success"` // true when the click was performed
}

// WebviewTypeInput contains parameters for typing text into a form element.
//
//	input := WebviewTypeInput{Selector: "input#email", Text: "user@example.com"}
type WebviewTypeInput struct {
	Selector string `json:"selector"` // e.g. "input#email"
	Text     string `json:"text"`     // e.g. "user@example.com"
}

// WebviewTypeOutput contains the result of a type action.
//
//	// out.Success == true
type WebviewTypeOutput struct {
	Success bool `json:"success"` // true when text was typed
}

// WebviewQueryInput contains parameters for querying DOM elements.
//
//	input := WebviewQueryInput{Selector: "div.card", All: true}
type WebviewQueryInput struct {
	Selector string `json:"selector"`      // e.g. "div.card"
	All      bool   `json:"all,omitempty"` // true to return all matches (default: first only)
}

// WebviewQueryOutput contains the result of a DOM query.
//
//	// out.Found == true, out.Count == 3, len(out.Elements) == 3
type WebviewQueryOutput struct {
	Found    bool                 `json:"found"`              // true when at least one element matched
	Count    int                  `json:"count"`              // number of matches
	Elements []WebviewElementInfo `json:"elements,omitempty"` // matched elements
}

// WebviewElementInfo represents information about a DOM element.
//
//	// el.TagName == "div", el.Attributes["class"] == "card active"
type WebviewElementInfo struct {
	NodeID      int                  `json:"nodeId"`                // CDP node identifier
	TagName     string               `json:"tagName"`               // e.g. "div", "button"
	Attributes  map[string]string    `json:"attributes,omitempty"`  // e.g. {"class": "card", "id": "main"}
	BoundingBox *webview.BoundingBox `json:"boundingBox,omitempty"` // viewport coordinates
}

// WebviewConsoleInput contains parameters for getting console output.
//
//	input := WebviewConsoleInput{Clear: true}
type WebviewConsoleInput struct {
	Clear bool `json:"clear,omitempty"` // true to clear the buffer after reading
}

// WebviewConsoleOutput contains console messages.
//
//	// out.Count == 5, out.Messages[0].Type == "log"
type WebviewConsoleOutput struct {
	Messages []WebviewConsoleMessage `json:"messages"` // captured console entries
	Count    int                     `json:"count"`    // number of messages
}

// WebviewConsoleMessage represents a single browser console entry.
//
//	// msg.Type == "log", msg.Text == "App loaded"
type WebviewConsoleMessage struct {
	Type      string `json:"type"`           // e.g. "log", "warn", "error"
	Text      string `json:"text"`           // e.g. "App loaded"
	Timestamp string `json:"timestamp"`      // RFC3339 formatted
	URL       string `json:"url,omitempty"`  // source file URL
	Line      int    `json:"line,omitempty"` // source line number
}

// WebviewEvalInput contains parameters for evaluating JavaScript.
//
//	input := WebviewEvalInput{Script: "document.title"}
type WebviewEvalInput struct {
	Script string `json:"script"` // e.g. "document.title"
}

// WebviewEvalOutput contains the result of JavaScript evaluation.
//
//	// out.Success == true, out.Result == "Dashboard - Host UK"
type WebviewEvalOutput struct {
	Success bool   `json:"success"`          // true when script executed without error
	Result  any    `json:"result,omitempty"` // return value of the script
	Error   string `json:"error,omitempty"`  // JS error message if execution failed
}

// WebviewScreenshotInput contains parameters for taking a screenshot.
//
//	input := WebviewScreenshotInput{Format: "png"}
type WebviewScreenshotInput struct {
	Format string `json:"format,omitempty"` // "png" or "jpeg" (default: "png")
}

// WebviewScreenshotOutput contains the screenshot data.
//
//	// out.Success == true, out.Format == "png", len(out.Data) > 0
type WebviewScreenshotOutput struct {
	Success bool   `json:"success"` // true when screenshot was captured
	Data    string `json:"data"`    // base64-encoded image bytes
	Format  string `json:"format"`  // "png" or "jpeg"
}

// WebviewWaitInput contains parameters for waiting for an element to appear.
//
//	input := WebviewWaitInput{Selector: "div.loaded", Timeout: 10}
type WebviewWaitInput struct {
	Selector string `json:"selector,omitempty"` // e.g. "div.loaded"
	Timeout  int    `json:"timeout,omitempty"`  // seconds to wait before timing out
}

// WebviewWaitOutput contains the result of waiting for an element.
//
//	// out.Success == true, out.Message == "Element found: div.loaded"
type WebviewWaitOutput struct {
	Success bool   `json:"success"`           // true when element appeared
	Message string `json:"message,omitempty"` // e.g. "Element found: div.loaded"
}

// WebviewDisconnectInput takes no parameters.
//
//	input := WebviewDisconnectInput{}
type WebviewDisconnectInput struct{}

// WebviewDisconnectOutput contains the result of disconnecting.
//
//	// out.Success == true, out.Message == "Disconnected from Chrome DevTools"
type WebviewDisconnectOutput struct {
	Success bool   `json:"success"`           // true when disconnection completed
	Message string `json:"message,omitempty"` // e.g. "Disconnected from Chrome DevTools"
}

// registerWebviewTools adds webview tools to the MCP server.
func (s *Service) registerWebviewTools(server *mcp.Server) {
	addToolRecorded(s, server, "webview", &mcp.Tool{
		Name:        "webview_connect",
		Description: "Connect to Chrome DevTools Protocol. Start Chrome with --remote-debugging-port=9222 first.",
	}, s.webviewConnect)

	addToolRecorded(s, server, "webview", &mcp.Tool{
		Name:        "webview_disconnect",
		Description: "Disconnect from Chrome DevTools.",
	}, s.webviewDisconnect)

	addToolRecorded(s, server, "webview", &mcp.Tool{
		Name:        "webview_navigate",
		Description: "Navigate the browser to a URL.",
	}, s.webviewNavigate)

	addToolRecorded(s, server, "webview", &mcp.Tool{
		Name:        "webview_click",
		Description: "Click on an element by CSS selector.",
	}, s.webviewClick)

	addToolRecorded(s, server, "webview", &mcp.Tool{
		Name:        "webview_type",
		Description: "Type text into an element by CSS selector.",
	}, s.webviewType)

	addToolRecorded(s, server, "webview", &mcp.Tool{
		Name:        "webview_query",
		Description: "Query DOM elements by CSS selector.",
	}, s.webviewQuery)

	addToolRecorded(s, server, "webview", &mcp.Tool{
		Name:        "webview_console",
		Description: "Get browser console output.",
	}, s.webviewConsole)

	addToolRecorded(s, server, "webview", &mcp.Tool{
		Name:        "webview_eval",
		Description: "Evaluate JavaScript in the browser context.",
	}, s.webviewEval)

	addToolRecorded(s, server, "webview", &mcp.Tool{
		Name:        "webview_screenshot",
		Description: "Capture a screenshot of the browser window.",
	}, s.webviewScreenshot)

	addToolRecorded(s, server, "webview", &mcp.Tool{
		Name:        "webview_wait",
		Description: "Wait for an element to appear by CSS selector.",
	}, s.webviewWait)
}

// webviewConnect handles the webview_connect tool call.
func (s *Service) webviewConnect(ctx context.Context, req *mcp.CallToolRequest, input WebviewConnectInput) (*mcp.CallToolResult, WebviewConnectOutput, error) {
	webviewMu.Lock()
	defer webviewMu.Unlock()

	s.logger.Security("MCP tool execution", "tool", "webview_connect", "debug_url", input.DebugURL, "user", log.Username())

	if input.DebugURL == "" {
		return nil, WebviewConnectOutput{}, log.E("webviewConnect", "debug_url is required", nil)
	}

	// Close existing connection if any
	if webviewInstance != nil {
		_ = webviewInstance.Close()
		webviewInstance = nil
	}

	// Set up options
	opts := []webview.Option{
		webview.WithDebugURL(input.DebugURL),
	}

	if input.Timeout > 0 {
		opts = append(opts, webview.WithTimeout(time.Duration(input.Timeout)*time.Second))
	}

	// Create new webview instance
	wv, err := webview.New(opts...)
	if err != nil {
		log.Error("mcp: webview connect failed", "debug_url", input.DebugURL, "err", err)
		return nil, WebviewConnectOutput{}, log.E("webviewConnect", "failed to connect", err)
	}

	webviewInstance = wv

	return nil, WebviewConnectOutput{
		Success: true,
		Message: core.Sprintf("Connected to Chrome DevTools at %s", input.DebugURL),
	}, nil
}

// webviewDisconnect handles the webview_disconnect tool call.
func (s *Service) webviewDisconnect(ctx context.Context, req *mcp.CallToolRequest, input WebviewDisconnectInput) (*mcp.CallToolResult, WebviewDisconnectOutput, error) {
	webviewMu.Lock()
	defer webviewMu.Unlock()

	s.logger.Info("MCP tool execution", "tool", "webview_disconnect", "user", log.Username())

	if webviewInstance == nil {
		return nil, WebviewDisconnectOutput{
			Success: true,
			Message: "No active connection",
		}, nil
	}

	if err := webviewInstance.Close(); err != nil {
		log.Error("mcp: webview disconnect failed", "err", err)
		return nil, WebviewDisconnectOutput{}, log.E("webviewDisconnect", "failed to disconnect", err)
	}

	webviewInstance = nil

	return nil, WebviewDisconnectOutput{
		Success: true,
		Message: "Disconnected from Chrome DevTools",
	}, nil
}

// webviewNavigate handles the webview_navigate tool call.
func (s *Service) webviewNavigate(ctx context.Context, req *mcp.CallToolRequest, input WebviewNavigateInput) (*mcp.CallToolResult, WebviewNavigateOutput, error) {
	webviewMu.Lock()
	defer webviewMu.Unlock()

	s.logger.Info("MCP tool execution", "tool", "webview_navigate", "url", input.URL, "user", log.Username())

	if webviewInstance == nil {
		return nil, WebviewNavigateOutput{}, errNotConnected
	}

	if input.URL == "" {
		return nil, WebviewNavigateOutput{}, log.E("webviewNavigate", "url is required", nil)
	}

	if err := webviewInstance.Navigate(input.URL); err != nil {
		log.Error("mcp: webview navigate failed", "url", input.URL, "err", err)
		return nil, WebviewNavigateOutput{}, log.E("webviewNavigate", "failed to navigate", err)
	}

	return nil, WebviewNavigateOutput{
		Success: true,
		URL:     input.URL,
	}, nil
}

// webviewClick handles the webview_click tool call.
func (s *Service) webviewClick(ctx context.Context, req *mcp.CallToolRequest, input WebviewClickInput) (*mcp.CallToolResult, WebviewClickOutput, error) {
	webviewMu.Lock()
	defer webviewMu.Unlock()

	s.logger.Info("MCP tool execution", "tool", "webview_click", "selector", input.Selector, "user", log.Username())

	if webviewInstance == nil {
		return nil, WebviewClickOutput{}, errNotConnected
	}

	if input.Selector == "" {
		return nil, WebviewClickOutput{}, errSelectorRequired
	}

	if err := webviewInstance.Click(input.Selector); err != nil {
		log.Error("mcp: webview click failed", "selector", input.Selector, "err", err)
		return nil, WebviewClickOutput{}, log.E("webviewClick", "failed to click", err)
	}

	return nil, WebviewClickOutput{Success: true}, nil
}

// webviewType handles the webview_type tool call.
func (s *Service) webviewType(ctx context.Context, req *mcp.CallToolRequest, input WebviewTypeInput) (*mcp.CallToolResult, WebviewTypeOutput, error) {
	webviewMu.Lock()
	defer webviewMu.Unlock()

	s.logger.Info("MCP tool execution", "tool", "webview_type", "selector", input.Selector, "user", log.Username())

	if webviewInstance == nil {
		return nil, WebviewTypeOutput{}, errNotConnected
	}

	if input.Selector == "" {
		return nil, WebviewTypeOutput{}, errSelectorRequired
	}

	if err := webviewInstance.Type(input.Selector, input.Text); err != nil {
		log.Error("mcp: webview type failed", "selector", input.Selector, "err", err)
		return nil, WebviewTypeOutput{}, log.E("webviewType", "failed to type", err)
	}

	return nil, WebviewTypeOutput{Success: true}, nil
}

// webviewQuery handles the webview_query tool call.
func (s *Service) webviewQuery(ctx context.Context, req *mcp.CallToolRequest, input WebviewQueryInput) (*mcp.CallToolResult, WebviewQueryOutput, error) {
	webviewMu.Lock()
	defer webviewMu.Unlock()

	s.logger.Info("MCP tool execution", "tool", "webview_query", "selector", input.Selector, "all", input.All, "user", log.Username())

	if webviewInstance == nil {
		return nil, WebviewQueryOutput{}, errNotConnected
	}

	if input.Selector == "" {
		return nil, WebviewQueryOutput{}, errSelectorRequired
	}

	if input.All {
		elements, err := webviewInstance.QuerySelectorAll(input.Selector)
		if err != nil {
			log.Error("mcp: webview query all failed", "selector", input.Selector, "err", err)
			return nil, WebviewQueryOutput{}, log.E("webviewQuery", "failed to query", err)
		}

		output := WebviewQueryOutput{
			Found:    len(elements) > 0,
			Count:    len(elements),
			Elements: make([]WebviewElementInfo, len(elements)),
		}

		for i, elem := range elements {
			output.Elements[i] = WebviewElementInfo{
				NodeID:      elem.NodeID,
				TagName:     elem.TagName,
				Attributes:  elem.Attributes,
				BoundingBox: elem.BoundingBox,
			}
		}

		return nil, output, nil
	}

	elem, err := webviewInstance.QuerySelector(input.Selector)
	if err != nil {
		// Element not found is not necessarily an error
		return nil, WebviewQueryOutput{
			Found: false,
			Count: 0,
		}, nil
	}

	return nil, WebviewQueryOutput{
		Found: true,
		Count: 1,
		Elements: []WebviewElementInfo{{
			NodeID:      elem.NodeID,
			TagName:     elem.TagName,
			Attributes:  elem.Attributes,
			BoundingBox: elem.BoundingBox,
		}},
	}, nil
}

// webviewConsole handles the webview_console tool call.
func (s *Service) webviewConsole(ctx context.Context, req *mcp.CallToolRequest, input WebviewConsoleInput) (*mcp.CallToolResult, WebviewConsoleOutput, error) {
	webviewMu.Lock()
	defer webviewMu.Unlock()

	s.logger.Info("MCP tool execution", "tool", "webview_console", "clear", input.Clear, "user", log.Username())

	if webviewInstance == nil {
		return nil, WebviewConsoleOutput{}, errNotConnected
	}

	messages := webviewInstance.GetConsole()

	output := WebviewConsoleOutput{
		Messages: make([]WebviewConsoleMessage, len(messages)),
		Count:    len(messages),
	}

	for i, msg := range messages {
		output.Messages[i] = WebviewConsoleMessage{
			Type:      msg.Type,
			Text:      msg.Text,
			Timestamp: msg.Timestamp.Format(time.RFC3339),
			URL:       msg.URL,
			Line:      msg.Line,
		}
	}

	if input.Clear {
		webviewInstance.ClearConsole()
	}

	return nil, output, nil
}

// webviewEval handles the webview_eval tool call.
func (s *Service) webviewEval(ctx context.Context, req *mcp.CallToolRequest, input WebviewEvalInput) (*mcp.CallToolResult, WebviewEvalOutput, error) {
	webviewMu.Lock()
	defer webviewMu.Unlock()

	s.logger.Security("MCP tool execution", "tool", "webview_eval", "user", log.Username())

	if webviewInstance == nil {
		return nil, WebviewEvalOutput{}, errNotConnected
	}

	if input.Script == "" {
		return nil, WebviewEvalOutput{}, log.E("webviewEval", "script is required", nil)
	}

	result, err := webviewInstance.Evaluate(input.Script)
	if err != nil {
		log.Error("mcp: webview eval failed", "err", err)
		return nil, WebviewEvalOutput{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	return nil, WebviewEvalOutput{
		Success: true,
		Result:  result,
	}, nil
}

// webviewScreenshot handles the webview_screenshot tool call.
func (s *Service) webviewScreenshot(ctx context.Context, req *mcp.CallToolRequest, input WebviewScreenshotInput) (*mcp.CallToolResult, WebviewScreenshotOutput, error) {
	webviewMu.Lock()
	defer webviewMu.Unlock()

	s.logger.Info("MCP tool execution", "tool", "webview_screenshot", "format", input.Format, "user", log.Username())

	if webviewInstance == nil {
		return nil, WebviewScreenshotOutput{}, errNotConnected
	}

	format := input.Format
	if format == "" {
		format = "png"
	}
	format = core.Lower(format)

	data, err := webviewInstance.Screenshot()
	if err != nil {
		log.Error("mcp: webview screenshot failed", "err", err)
		return nil, WebviewScreenshotOutput{}, log.E("webviewScreenshot", "failed to capture screenshot", err)
	}

	encoded, outputFormat, err := normalizeScreenshotData(data, format)
	if err != nil {
		return nil, WebviewScreenshotOutput{}, log.E("webviewScreenshot", "failed to encode screenshot", err)
	}

	return nil, WebviewScreenshotOutput{
		Success: true,
		Data:    base64.StdEncoding.EncodeToString(encoded),
		Format:  outputFormat,
	}, nil
}

// normalizeScreenshotData converts screenshot bytes into the requested format.
// PNG is preserved as-is. JPEG requests are re-encoded so the output matches
// the declared format in WebviewScreenshotOutput.
func normalizeScreenshotData(data []byte, format string) ([]byte, string, error) {
	switch format {
	case "", "png":
		return data, "png", nil
	case "jpeg", "jpg":
		img, _, err := image.Decode(bytes.NewReader(data))
		if err != nil {
			return nil, "", err
		}
		var buf bytes.Buffer
		if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 90}); err != nil {
			return nil, "", err
		}
		return buf.Bytes(), "jpeg", nil
	default:
		return nil, "", log.E("webviewScreenshot", "unsupported screenshot format: "+format, nil)
	}
}

// webviewWait handles the webview_wait tool call.
func (s *Service) webviewWait(ctx context.Context, req *mcp.CallToolRequest, input WebviewWaitInput) (*mcp.CallToolResult, WebviewWaitOutput, error) {
	webviewMu.Lock()
	defer webviewMu.Unlock()

	s.logger.Info("MCP tool execution", "tool", "webview_wait", "selector", input.Selector, "timeout", input.Timeout, "user", log.Username())

	if webviewInstance == nil {
		return nil, WebviewWaitOutput{}, errNotConnected
	}

	if input.Selector == "" {
		return nil, WebviewWaitOutput{}, errSelectorRequired
	}

	timeout := time.Duration(input.Timeout) * time.Second
	if timeout <= 0 {
		timeout = 30 * time.Second
	}

	if err := waitForSelector(ctx, timeout, input.Selector, func(selector string) error {
		_, err := webviewInstance.QuerySelector(selector)
		return err
	}); err != nil {
		log.Error("mcp: webview wait failed", "selector", input.Selector, "err", err)
		return nil, WebviewWaitOutput{}, log.E("webviewWait", "failed to wait for selector", err)
	}

	return nil, WebviewWaitOutput{
		Success: true,
		Message: core.Sprintf("Element found: %s", input.Selector),
	}, nil
}

// waitForSelector polls until the selector exists or the timeout elapses.
// Query helpers in go-webview report "element not found" as an error, so we
// keep retrying until we see the element or hit the deadline.
func waitForSelector(ctx context.Context, timeout time.Duration, selector string, query func(string) error) error {
	if timeout <= 0 {
		timeout = 30 * time.Second
	}

	waitCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	for {
		err := query(selector)
		if err == nil {
			return nil
		}
		if !core.Contains(err.Error(), "element not found") {
			return err
		}

		select {
		case <-waitCtx.Done():
			return log.E("webviewWait", "timed out waiting for selector", waitCtx.Err())
		case <-ticker.C:
		}
	}
}
