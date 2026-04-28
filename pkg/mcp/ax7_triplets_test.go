// SPDX-License-Identifier: EUPL-1.2

package mcp

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net"
	"path/filepath"
	"strings"

	core "dappco.re/go"
	api "dappco.re/go/api"
	"dappco.re/go/process"
	"dappco.re/go/ws"
	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

type T = core.T

var (
	AssertContains  = core.AssertContains
	AssertEqual     = core.AssertEqual
	AssertError     = core.AssertError
	AssertErrorIs   = core.AssertErrorIs
	AssertFalse     = core.AssertFalse
	AssertLen       = core.AssertLen
	AssertNil       = core.AssertNil
	AssertNoError   = core.AssertNoError
	AssertNotEmpty  = core.AssertNotEmpty
	AssertNotEqual  = core.AssertNotEqual
	AssertNotNil    = core.AssertNotNil
	AssertNotPanics = core.AssertNotPanics
	AssertPanics    = core.AssertPanics
	AssertTrue      = core.AssertTrue
	Contains        = core.Contains
	Join            = core.Join
	RequireNoError  = core.RequireNoError
)

func ax7ConnectionPair(t *T) (*connConnection, net.Conn, func()) {
	t.Helper()
	left, right := net.Pipe()
	transport := &connTransport{conn: left}
	conn, err := transport.Connect(context.Background())
	RequireNoError(t, err)
	return conn.(*connConnection), right, func() {
		if err := left.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
			t.Fatalf("close left pipe: %v", err)
		}
		if err := right.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
			t.Fatalf("close right pipe: %v", err)
		}
	}
}

func ax7JSONRPCID(t *T, value any) jsonrpc.ID {
	t.Helper()
	id, err := jsonrpc.MakeID(value)
	RequireNoError(t, err)
	return id
}

func reflectTransformer[T any](got TransformerIn) bool {
	_, ok := got.(T)
	return ok
}

func toolNames(tools []ToolRecord) []string {
	out := make([]string, 0, len(tools))
	for _, tool := range tools {
		out = append(out, tool.Name)
	}
	return out
}

func TestAX7_AddToolRecorded_Good(t *T) {
	svc := ax7NewServiceForTest(t, Options{})
	AddToolRecorded(svc, svc.Server(), "ax7", &sdkmcp.Tool{Name: "ax7_echo", InputSchema: map[string]any{"type": "object"}}, func(context.Context, *sdkmcp.CallToolRequest, struct{}) (*sdkmcp.CallToolResult, map[string]string, error) {
		return nil, map[string]string{"ok": "true"}, nil
	})
	AssertContains(t, toolNames(svc.Tools()), "ax7_echo")
}
func TestAX7_AddToolRecorded_Bad(t *T) {
	svc := ax7NewServiceForTest(t, Options{})
	AssertPanics(t, func() {
		AddToolRecorded(svc, nil, "ax7", &sdkmcp.Tool{Name: "ax7_bad", InputSchema: map[string]any{"type": "object"}}, func(context.Context, *sdkmcp.CallToolRequest, struct{}) (*sdkmcp.CallToolResult, map[string]string, error) {
			return nil, map[string]string{}, nil
		})
	})
	AssertFalse(t, Contains(Join(",", toolNames(svc.Tools())...), "ax7_bad"))
}
func TestAX7_AddToolRecorded_Ugly(t *T) {
	svc := ax7NewServiceForTest(t, Options{})
	AddToolRecorded(svc, svc.Server(), "", &sdkmcp.Tool{Name: "ax7_empty_group", InputSchema: map[string]any{"type": "object"}}, func(context.Context, *sdkmcp.CallToolRequest, struct{}) (*sdkmcp.CallToolResult, map[string]string, error) {
		return nil, map[string]string{"ok": ""}, nil
	})
	AssertContains(t, toolNames(svc.Tools()), "ax7_empty_group")
}
func TestAX7_AnthropicTransformer_Detect_Good(t *T) {
	body := []byte(`{"model":"claude","max_tokens":8,"messages":[{"role":"user","content":"hi"}]}`)
	got := (AnthropicTransformer{}).Detect(body, "", "")
	AssertTrue(t, got)
}
func TestAX7_AnthropicTransformer_Detect_Bad(t *T) {
	body := []byte(`{"model":"claude","messages":[{"role":"system","content":"policy"}]}`)
	got := (AnthropicTransformer{}).Detect(body, "", "")
	AssertFalse(t, got)
}
func TestAX7_AnthropicTransformer_Detect_Ugly(t *T) {
	got := (AnthropicTransformer{}).Detect(nil, "application/anthropic+json", "")
	AssertTrue(t, got)
	AssertTrue(t, (AnthropicTransformer{}).Detect(nil, "", "/v1/messages"))
}
func TestAX7_AnthropicTransformer_Normalise_Good(t *T) {
	body := []byte(`{"model":"claude","max_tokens":8,"messages":[{"role":"user","content":"hi"}]}`)
	req, err := (AnthropicTransformer{}).Normalise(body)
	AssertNoError(t, err)
	AssertEqual(t, "sampling/createMessage", req.Method)
}
func TestAX7_AnthropicTransformer_Normalise_Bad(t *T) {
	req, err := (AnthropicTransformer{}).Normalise([]byte(`{"messages":[]}`))
	AssertError(t, err)
	AssertEqual(t, MCPRequest{}, req)
}
func TestAX7_AnthropicTransformer_Normalise_Ugly(t *T) {
	body := []byte(`{"model":"claude","messages":[{"role":"assistant","content":[{"type":"tool_use","id":"1","name":"echo","input":{"x":1}}]}]}`)
	req, err := (AnthropicTransformer{}).Normalise(body)
	AssertNoError(t, err)
	AssertEqual(t, "tools/call", req.Method)
}
func TestAX7_AnthropicTransformer_Transform_Good(t *T) {
	out, err := (AnthropicTransformer{}).Transform(MCPResult{ID: "1", Content: []MCPContent{{Type: "text", Text: "ok"}}})
	AssertNoError(t, err)
	AssertContains(t, string(out), `"type":"message"`)
}
func TestAX7_AnthropicTransformer_Transform_Bad(t *T) {
	_, err := (AnthropicTransformer{}).Transform(MCPResult{ToolCalls: []MCPToolCall{{Name: "bad", Arguments: map[string]any{"bad": make(chan int)}}}})
	AssertError(t, err)
	AssertContains(t, err.Error(), "unsupported type")
}
func TestAX7_AnthropicTransformer_Transform_Ugly(t *T) {
	out, err := (AnthropicTransformer{}).Transform(MCPResult{})
	AssertNoError(t, err)
	AssertContains(t, string(out), `"text":""`)
}
func TestAX7_BridgeToAPI_Good(t *T) {
	bridge := api.NewToolBridge()
	svc := ax7NewServiceForTest(t, Options{})
	BridgeToAPI(svc, bridge)
	AssertTrue(t, len(bridge.Tools()) > 0)
}
func TestAX7_BridgeToAPI_Bad(t *T) {
	bridge := api.NewToolBridge()
	BridgeToAPI(nil, bridge)
	AssertLen(t, bridge.Tools(), 0)
}
func TestAX7_BridgeToAPI_Ugly(t *T) {
	svc := ax7NewServiceForTest(t, Options{})
	BridgeToAPI(svc, nil)
	AssertTrue(t, len(svc.Tools()) > 0)
}
func TestAX7_ChannelCapability_Good(t *T) {
	got := ChannelCapability()
	AssertNotNil(t, got[ClaudeChannelCapabilityName])
	AssertLen(t, got, 1)
}
func TestAX7_ChannelCapability_Bad(t *T) {
	got := ChannelCapability()
	AssertNil(t, got["missing/channel"])
	AssertNotNil(t, got[ClaudeChannelCapabilityName])
}
func TestAX7_ChannelCapability_Ugly(t *T) {
	got := ChannelCapability()
	got[ClaudeChannelCapabilityName] = "mutated"
	AssertNotEqual(t, got[ClaudeChannelCapabilityName], ChannelCapability()[ClaudeChannelCapabilityName])
}
func TestAX7_ChannelCapabilityChannels_Good(t *T) {
	got := ChannelCapabilityChannels()
	AssertContains(t, got, ChannelAgentStatus)
	AssertContains(t, got, ChannelProcessOutput)
}
func TestAX7_ChannelCapabilityChannels_Bad(t *T) {
	got := ChannelCapabilityChannels()
	AssertFalse(t, Contains(Join(",", got...), "missing.channel"))
	AssertTrue(t, len(got) > 0)
}
func TestAX7_ChannelCapabilityChannels_Ugly(t *T) {
	got := ChannelCapabilityChannels()
	got[0] = "mutated"
	AssertNotEqual(t, "mutated", ChannelCapabilityChannels()[0])
}
func TestAX7_ChannelCapabilitySpec_Map_Good(t *T) {
	spec := ChannelCapabilitySpec{Version: "1", Description: "d", Channels: []string{"a"}}
	got := spec.Map()
	AssertEqual(t, "1", got["version"])
	AssertEqual(t, "d", got["description"])
}
func TestAX7_ChannelCapabilitySpec_Map_Bad(t *T) {
	got := (ChannelCapabilitySpec{}).Map()
	AssertEqual(t, "", got["version"])
	AssertEqual(t, []string(nil), got["channels"])
}
func TestAX7_ChannelCapabilitySpec_Map_Ugly(t *T) {
	spec := ChannelCapabilitySpec{Channels: []string{"a"}}
	channels := spec.Map()["channels"].([]string)
	channels[0] = "mutated"
	AssertEqual(t, "a", spec.Channels[0])
}
func TestAX7_ClaudeChannelCapability_Good(t *T) {
	got := ClaudeChannelCapability()
	AssertEqual(t, "1", got.Version)
	AssertContains(t, got.Channels, ChannelBrainRecallDone)
}
func TestAX7_ClaudeChannelCapability_Bad(t *T) {
	got := ClaudeChannelCapability()
	AssertNotEmpty(t, got.Description)
	AssertFalse(t, Contains(Join(",", got.Channels...), "missing.channel"))
}
func TestAX7_ClaudeChannelCapability_Ugly(t *T) {
	got := ClaudeChannelCapability()
	got.Channels[0] = "mutated"
	AssertNotEqual(t, "mutated", ClaudeChannelCapability().Channels[0])
}
func TestAX7_Connection_Close_Good(t *T) {
	c, _, cleanup := ax7ConnectionPair(t)
	defer cleanup()
	AssertNoError(t, c.Close())
}
func TestAX7_Connection_Close_Bad(t *T) {
	var c *connConnection
	AssertPanics(t, func() { _ = c.Close() })
	AssertNil(t, c)
}
func TestAX7_Connection_Close_Ugly(t *T) {
	c, _, cleanup := ax7ConnectionPair(t)
	defer cleanup()
	AssertNoError(t, c.Close())
	AssertNoError(t, c.Close())
}
func TestAX7_Connection_Read_Good(t *T) {
	c, right, cleanup := ax7ConnectionPair(t)
	defer cleanup()
	go func() { _, _ = right.Write([]byte(`{"jsonrpc":"2.0","id":1,"method":"x"}` + "\n")) }()
	msg, err := c.Read(context.Background())
	AssertNoError(t, err)
	AssertNotNil(t, msg)
}
func TestAX7_Connection_Read_Bad(t *T) {
	c, right, cleanup := ax7ConnectionPair(t)
	defer cleanup()
	go func() { _, _ = right.Write([]byte(`bad` + "\n")) }()
	msg, err := c.Read(context.Background())
	AssertError(t, err)
	AssertNil(t, msg)
}
func TestAX7_Connection_Read_Ugly(t *T) {
	c, right, cleanup := ax7ConnectionPair(t)
	defer cleanup()
	AssertNoError(t, right.Close())
	_, err := c.Read(context.Background())
	AssertErrorIs(t, err, io.EOF)
}
func TestAX7_Connection_SessionID_Good(t *T) {
	c, _, cleanup := ax7ConnectionPair(t)
	defer cleanup()
	AssertContains(t, c.SessionID(), "tcp-")
}
func TestAX7_Connection_SessionID_Bad(t *T) {
	var c *connConnection
	AssertPanics(t, func() { _ = c.SessionID() })
	AssertNil(t, c)
}
func TestAX7_Connection_SessionID_Ugly(t *T) {
	c, _, cleanup := ax7ConnectionPair(t)
	defer cleanup()
	AssertNotEmpty(t, c.SessionID())
}
func TestAX7_Connection_Write_Good(t *T) {
	c, right, cleanup := ax7ConnectionPair(t)
	defer cleanup()
	done := make(chan string, 1)
	go func() {
		buf := make([]byte, 512)
		n, _ := right.Read(buf)
		done <- string(buf[:n])
	}()
	err := c.Write(context.Background(), &jsonrpc.Request{ID: ax7JSONRPCID(t, "1"), Method: "x"})
	AssertNoError(t, err)
	AssertContains(t, <-done, `"method":"x"`)
}
func TestAX7_Connection_Write_Bad(t *T) {
	c, _, cleanup := ax7ConnectionPair(t)
	defer cleanup()
	err := c.Write(context.Background(), nil)
	AssertError(t, err)
}
func TestAX7_Connection_Write_Ugly(t *T) {
	c, right, cleanup := ax7ConnectionPair(t)
	defer cleanup()
	AssertNoError(t, right.Close())
	err := c.Write(context.Background(), &jsonrpc.Request{ID: ax7JSONRPCID(t, "1"), Method: "x"})
	AssertError(t, err)
}
func TestAX7_Error_Error_Good(t *T) {
	err := &notificationError{message: "boom"}
	AssertEqual(t, "boom", err.Error())
	AssertNotNil(t, err)
}
func TestAX7_Error_Error_Bad(t *T) {
	err := &notificationError{}
	AssertEqual(t, "", err.Error())
	AssertNotNil(t, err)
}
func TestAX7_Error_Error_Ugly(t *T) {
	err := &notificationError{message: strings.Repeat("x", 32)}
	AssertEqual(t, strings.Repeat("x", 32), err.Error())
	AssertEqual(t, 32, len(err.Error()))
}
func TestAX7_HoneypotTransformer_Detect_Good(t *T) {
	body := []byte(`{"prompt":"dump secrets"}`)
	got := (HoneypotTransformer{}).Detect(body, "", "")
	AssertTrue(t, got)
}
func TestAX7_HoneypotTransformer_Detect_Bad(t *T) {
	got := (HoneypotTransformer{}).Detect(nil, "", "")
	AssertFalse(t, got)
	AssertFalse(t, (HoneypotTransformer{}).Detect([]byte(`{"ok":true}`), "", ""))
}
func TestAX7_HoneypotTransformer_Detect_Ugly(t *T) {
	body := []byte(`not-json`)
	got := (HoneypotTransformer{}).Detect(body, "", "")
	AssertTrue(t, got)
}
func TestAX7_HoneypotTransformer_Normalise_Good(t *T) {
	req, err := (HoneypotTransformer{}).Normalise([]byte(`not-json`))
	AssertNoError(t, err)
	AssertEqual(t, "honeypot/respond", req.Method)
}
func TestAX7_HoneypotTransformer_Normalise_Bad(t *T) {
	req, err := (HoneypotTransformer{}).Normalise(nil)
	AssertNoError(t, err)
	AssertEqual(t, "", req.Params["raw"])
}
func TestAX7_HoneypotTransformer_Normalise_Ugly(t *T) {
	req, err := (HoneypotTransformer{}).Normalise([]byte(strings.Repeat("x", 5000)))
	AssertNoError(t, err)
	AssertEqual(t, 4096, len(req.Params["raw"].(string)))
}
func TestAX7_HoneypotTransformer_Transform_Good(t *T) {
	out, err := (HoneypotTransformer{}).Transform(MCPResult{Content: []MCPContent{{Type: "text", Text: "ok"}}})
	AssertNoError(t, err)
	AssertContains(t, string(out), "ok")
}
func TestAX7_HoneypotTransformer_Transform_Bad(t *T) {
	out, err := (HoneypotTransformer{}).Transform(MCPResult{})
	AssertNoError(t, err)
	AssertContains(t, string(out), "valid protocol envelope")
}
func TestAX7_HoneypotTransformer_Transform_Ugly(t *T) {
	out, err := (HoneypotTransformer{}).Transform(MCPResult{ID: "abc"})
	AssertNoError(t, err)
	AssertContains(t, string(out), "chatcmpl-honeypot-abc")
}
func TestAX7_InputError_Error_Good(t *T) {
	err := invalidRESTInputError(errors.New("bad json"))
	AssertContains(t, err.Error(), "bad json")
	AssertTrue(t, errors.Is(err, errInvalidRESTInput))
}
func TestAX7_InputError_Error_Bad(t *T) {
	var err *restInputError
	AssertEqual(t, "invalid REST input", err.Error())
	AssertNil(t, err)
}
func TestAX7_InputError_Error_Ugly(t *T) {
	err := invalidRESTInputError(nil)
	AssertEqual(t, "invalid REST input", err.Error())
	AssertNil(t, errors.Unwrap(err))
}
func TestAX7_InputError_Is_Good(t *T) {
	err := invalidRESTInputError(errors.New("bad json"))
	AssertTrue(t, errors.Is(err, errInvalidRESTInput))
	AssertError(t, err)
}
func TestAX7_InputError_Is_Bad(t *T) {
	err := errors.New("plain")
	AssertFalse(t, errors.Is(err, errInvalidRESTInput))
	AssertError(t, err)
}
func TestAX7_InputError_Is_Ugly(t *T) {
	err := &restInputError{}
	AssertTrue(t, errors.Is(err, &restInputError{}))
	AssertEqual(t, "invalid REST input", err.Error())
}
func TestAX7_InputError_Unwrap_Good(t *T) {
	cause := errors.New("bad json")
	err := invalidRESTInputError(cause)
	AssertErrorIs(t, errors.Unwrap(err), cause)
}
func TestAX7_InputError_Unwrap_Bad(t *T) {
	var err *restInputError
	AssertNil(t, err.Unwrap())
	AssertEqual(t, "invalid REST input", err.Error())
}
func TestAX7_InputError_Unwrap_Ugly(t *T) {
	err := invalidRESTInputError(nil)
	AssertNil(t, errors.Unwrap(err))
	AssertTrue(t, errors.Is(err, errInvalidRESTInput))
}
func TestAX7_MCPNativeTransformer_Detect_Good(t *T) {
	body := []byte(`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`)
	got := (MCPNativeTransformer{}).Detect(body, "", "")
	AssertTrue(t, got)
}
func TestAX7_MCPNativeTransformer_Detect_Bad(t *T) {
	got := (MCPNativeTransformer{}).Detect([]byte(`{"method":"tools/list"}`), "", "")
	AssertFalse(t, got)
	AssertFalse(t, (MCPNativeTransformer{}).Detect([]byte(`bad`), "", ""))
}
func TestAX7_MCPNativeTransformer_Detect_Ugly(t *T) {
	got := (MCPNativeTransformer{}).Detect(nil, "application/mcp+json", "")
	AssertTrue(t, got)
	AssertTrue(t, (MCPNativeTransformer{}).Detect(nil, "", "/mcp"))
}
func TestAX7_MCPNativeTransformer_Normalise_Good(t *T) {
	req, err := (MCPNativeTransformer{}).Normalise([]byte(`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`))
	AssertNoError(t, err)
	AssertEqual(t, "tools/list", req.Method)
}
func TestAX7_MCPNativeTransformer_Normalise_Bad(t *T) {
	req, err := (MCPNativeTransformer{}).Normalise([]byte(`bad`))
	AssertError(t, err)
	AssertEqual(t, MCPRequest{}, req)
}
func TestAX7_MCPNativeTransformer_Normalise_Ugly(t *T) {
	req, err := (MCPNativeTransformer{}).Normalise([]byte(`{"method":"ping"}`))
	AssertNoError(t, err)
	AssertEqual(t, "2.0", req.JSONRPC)
}
func TestAX7_MCPNativeTransformer_Transform_Good(t *T) {
	out, err := (MCPNativeTransformer{}).Transform(MCPResult{ID: 1, Result: map[string]any{"ok": true}})
	AssertNoError(t, err)
	AssertContains(t, string(out), `"jsonrpc":"2.0"`)
}
func TestAX7_MCPNativeTransformer_Transform_Bad(t *T) {
	_, err := (MCPNativeTransformer{}).Transform(MCPResult{Result: make(chan int)})
	AssertError(t, err)
	AssertContains(t, err.Error(), "unsupported type")
}
func TestAX7_MCPNativeTransformer_Transform_Ugly(t *T) {
	out, err := (MCPNativeTransformer{}).Transform(MCPResult{JSONRPC: "2.0", Error: map[string]any{"code": -1}})
	AssertNoError(t, err)
	AssertContains(t, string(out), `"error"`)
}
func TestAX7_Medium_Append_Good(t *T) {
	m := newCoreMedium(t.TempDir())
	AssertNoError(t, m.Write("a.txt", "hello"))
	w, err := m.Append("a.txt")
	AssertNoError(t, err)
	_, err = w.Write([]byte(" world"))
	AssertNoError(t, err)
	AssertNoError(t, w.Close())
	got, err := m.Read("a.txt")
	AssertNoError(t, err)
	AssertEqual(t, "hello world", got)
}
func TestAX7_Medium_Append_Bad(t *T) {
	m := newCoreMedium(t.TempDir())
	AssertNoError(t, m.EnsureDir("dir"))
	_, err := m.Append("dir")
	AssertError(t, err)
}
func TestAX7_Medium_Append_Ugly(t *T) {
	m := newCoreMedium(t.TempDir())
	w, err := m.Append("new.txt")
	AssertNoError(t, err)
	AssertNoError(t, w.Close())
}
func TestAX7_Medium_Create_Good(t *T) {
	m := newCoreMedium(t.TempDir())
	w, err := m.Create("a.txt")
	AssertNoError(t, err)
	_, err = w.Write([]byte("hello"))
	AssertNoError(t, err)
	AssertNoError(t, w.Close())
}
func TestAX7_Medium_Create_Bad(t *T) {
	m := newCoreMedium(t.TempDir())
	AssertNoError(t, m.EnsureDir("dir"))
	_, err := m.Create("dir")
	AssertError(t, err)
}
func TestAX7_Medium_Create_Ugly(t *T) {
	m := newCoreMedium(t.TempDir())
	w, err := m.Create("nested/a.txt")
	AssertNoError(t, err)
	AssertNoError(t, w.Close())
}
func TestAX7_Medium_Delete_Good(t *T) {
	m := newCoreMedium(t.TempDir())
	AssertNoError(t, m.Write("a.txt", "hello"))
	AssertNoError(t, m.Delete("a.txt"))
	AssertFalse(t, m.Exists("a.txt"))
}
func TestAX7_Medium_Delete_Bad(t *T) {
	m := newCoreMedium(t.TempDir())
	err := m.Delete("missing")
	AssertError(t, err)
}
func TestAX7_Medium_Delete_Ugly(t *T) {
	m := newCoreMedium(t.TempDir())
	AssertNoError(t, m.DeleteAll("missing"))
	AssertFalse(t, m.Exists("missing"))
}
func TestAX7_Medium_DeleteAll_Good(t *T) {
	m := newCoreMedium(t.TempDir())
	AssertNoError(t, m.Write("dir/a.txt", "hello"))
	AssertNoError(t, m.DeleteAll("dir"))
	AssertFalse(t, m.Exists("dir/a.txt"))
}
func TestAX7_Medium_DeleteAll_Bad(t *T) {
	m := newCoreMedium("/")
	err := m.DeleteAll("")
	AssertError(t, err)
}
func TestAX7_Medium_DeleteAll_Ugly(t *T) {
	m := newCoreMedium(t.TempDir())
	AssertNoError(t, m.DeleteAll("missing"))
	AssertFalse(t, m.Exists("missing"))
}
func TestAX7_Medium_EnsureDir_Good(t *T) {
	m := newCoreMedium(t.TempDir())
	AssertNoError(t, m.EnsureDir("dir/sub"))
	AssertTrue(t, m.IsDir("dir/sub"))
}
func TestAX7_Medium_EnsureDir_Bad(t *T) {
	m := newCoreMedium(t.TempDir())
	AssertNoError(t, m.Write("file", "x"))
	AssertError(t, m.EnsureDir("file"))
}
func TestAX7_Medium_EnsureDir_Ugly(t *T) {
	m := newCoreMedium(t.TempDir())
	AssertNoError(t, m.EnsureDir("nested/deep"))
	AssertNoError(t, m.EnsureDir("nested/deep"))
}
func TestAX7_Medium_Exists_Good(t *T) {
	m := newCoreMedium(t.TempDir())
	AssertNoError(t, m.Write("a.txt", "hello"))
	AssertTrue(t, m.Exists("a.txt"))
}
func TestAX7_Medium_Exists_Bad(t *T) {
	m := newCoreMedium(t.TempDir())
	AssertFalse(t, m.Exists("missing"))
	AssertFalse(t, m.Exists("../missing"))
}
func TestAX7_Medium_Exists_Ugly(t *T) {
	m := newCoreMedium(t.TempDir())
	AssertNoError(t, m.EnsureDir("dir"))
	AssertTrue(t, m.Exists("dir"))
}
func TestAX7_Medium_IsDir_Good(t *T) {
	m := newCoreMedium(t.TempDir())
	AssertNoError(t, m.EnsureDir("dir"))
	AssertTrue(t, m.IsDir("dir"))
}
func TestAX7_Medium_IsDir_Bad(t *T) {
	m := newCoreMedium(t.TempDir())
	AssertFalse(t, m.IsDir("missing"))
	AssertFalse(t, m.IsDir(""))
}
func TestAX7_Medium_IsDir_Ugly(t *T) {
	m := newCoreMedium(t.TempDir())
	AssertNoError(t, m.Write("a.txt", "hello"))
	AssertFalse(t, m.IsDir("a.txt"))
}
func TestAX7_Medium_IsFile_Good(t *T) {
	m := newCoreMedium(t.TempDir())
	AssertNoError(t, m.Write("a.txt", "hello"))
	AssertTrue(t, m.IsFile("a.txt"))
}
func TestAX7_Medium_IsFile_Bad(t *T) {
	m := newCoreMedium(t.TempDir())
	AssertFalse(t, m.IsFile("missing"))
	AssertFalse(t, m.IsFile(""))
}
func TestAX7_Medium_IsFile_Ugly(t *T) {
	m := newCoreMedium(t.TempDir())
	AssertNoError(t, m.EnsureDir("dir"))
	AssertFalse(t, m.IsFile("dir"))
}
func TestAX7_Medium_List_Good(t *T) {
	m := newCoreMedium(t.TempDir())
	AssertNoError(t, m.Write("dir/a.txt", "hello"))
	entries, err := m.List("dir")
	AssertNoError(t, err)
	AssertLen(t, entries, 1)
}
func TestAX7_Medium_List_Bad(t *T) {
	m := newCoreMedium(t.TempDir())
	entries, err := m.List("missing")
	AssertError(t, err)
	AssertNil(t, entries)
}
func TestAX7_Medium_List_Ugly(t *T) {
	m := newCoreMedium(t.TempDir())
	AssertNoError(t, m.EnsureDir("empty"))
	entries, err := m.List("empty")
	AssertNoError(t, err)
	AssertLen(t, entries, 0)
}
func TestAX7_Medium_Open_Good(t *T) {
	m := newCoreMedium(t.TempDir())
	AssertNoError(t, m.Write("a.txt", "hello"))
	f, err := m.Open("a.txt")
	AssertNoError(t, err)
	AssertNoError(t, f.Close())
}
func TestAX7_Medium_Open_Bad(t *T) {
	m := newCoreMedium(t.TempDir())
	f, err := m.Open("missing.txt")
	AssertError(t, err)
	AssertNil(t, f)
}
func TestAX7_Medium_Open_Ugly(t *T) {
	m := newCoreMedium(t.TempDir())
	AssertNoError(t, m.Write("empty.txt", ""))
	f, err := m.Open("empty.txt")
	AssertNoError(t, err)
	AssertNoError(t, f.Close())
}
func TestAX7_Medium_Read_Good(t *T) {
	m := newCoreMedium(t.TempDir())
	AssertNoError(t, m.Write("a.txt", "hello"))
	got, err := m.Read("a.txt")
	AssertNoError(t, err)
	AssertEqual(t, "hello", got)
}
func TestAX7_Medium_Read_Bad(t *T) {
	m := newCoreMedium(t.TempDir())
	got, err := m.Read("missing.txt")
	AssertError(t, err)
	AssertEqual(t, "", got)
}
func TestAX7_Medium_Read_Ugly(t *T) {
	m := newCoreMedium(t.TempDir())
	AssertNoError(t, m.Write("nested/empty.txt", ""))
	got, err := m.Read("nested/empty.txt")
	AssertNoError(t, err)
	AssertEqual(t, "", got)
}
func TestAX7_Medium_ReadStream_Good(t *T) {
	m := newCoreMedium(t.TempDir())
	AssertNoError(t, m.Write("a.txt", "hello"))
	r, err := m.ReadStream("a.txt")
	AssertNoError(t, err)
	defer r.Close()
	data, err := io.ReadAll(r)
	AssertNoError(t, err)
	AssertEqual(t, "hello", string(data))
}
func TestAX7_Medium_ReadStream_Bad(t *T) {
	m := newCoreMedium(t.TempDir())
	r, err := m.ReadStream("missing.txt")
	AssertError(t, err)
	AssertNil(t, r)
}
func TestAX7_Medium_ReadStream_Ugly(t *T) {
	m := newCoreMedium(t.TempDir())
	AssertNoError(t, m.Write("empty.txt", ""))
	r, err := m.ReadStream("empty.txt")
	AssertNoError(t, err)
	defer r.Close()
	data, err := io.ReadAll(r)
	AssertNoError(t, err)
	AssertEqual(t, "", string(data))
}
func TestAX7_Medium_Rename_Good(t *T) {
	m := newCoreMedium(t.TempDir())
	AssertNoError(t, m.Write("old.txt", "hello"))
	AssertNoError(t, m.Rename("old.txt", "new.txt"))
	AssertTrue(t, m.Exists("new.txt"))
}
func TestAX7_Medium_Rename_Bad(t *T) {
	m := newCoreMedium(t.TempDir())
	err := m.Rename("missing", "new")
	AssertError(t, err)
}
func TestAX7_Medium_Rename_Ugly(t *T) {
	m := newCoreMedium(t.TempDir())
	AssertNoError(t, m.Write("old.txt", ""))
	AssertNoError(t, m.EnsureDir("nested"))
	AssertNoError(t, m.Rename("old.txt", "nested/new.txt"))
	AssertTrue(t, m.Exists("nested/new.txt"))
}
func TestAX7_Medium_Stat_Good(t *T) {
	m := newCoreMedium(t.TempDir())
	AssertNoError(t, m.Write("a.txt", "hello"))
	info, err := m.Stat("a.txt")
	AssertNoError(t, err)
	AssertEqual(t, "a.txt", info.Name())
}
func TestAX7_Medium_Stat_Bad(t *T) {
	m := newCoreMedium(t.TempDir())
	info, err := m.Stat("missing.txt")
	AssertError(t, err)
	AssertNil(t, info)
}
func TestAX7_Medium_Stat_Ugly(t *T) {
	m := newCoreMedium(t.TempDir())
	AssertNoError(t, m.Write("empty.txt", ""))
	info, err := m.Stat("empty.txt")
	AssertNoError(t, err)
	AssertEqual(t, int64(0), info.Size())
}
func TestAX7_Medium_Write_Good(t *T) {
	m := newCoreMedium(t.TempDir())
	AssertNoError(t, m.Write("a.txt", "hello"))
	got, err := m.Read("a.txt")
	AssertNoError(t, err)
	AssertEqual(t, "hello", got)
}
func TestAX7_Medium_Write_Bad(t *T) {
	m := newCoreMedium(t.TempDir())
	AssertNoError(t, m.EnsureDir("dir"))
	err := m.Write("dir", "hello")
	AssertError(t, err)
}
func TestAX7_Medium_Write_Ugly(t *T) {
	m := newCoreMedium(t.TempDir())
	AssertNoError(t, m.Write("nested/empty.txt", ""))
	got, err := m.Read("nested/empty.txt")
	AssertNoError(t, err)
	AssertEqual(t, "", got)
}
func TestAX7_Medium_WriteMode_Good(t *T) {
	m := newCoreMedium(t.TempDir())
	AssertNoError(t, m.WriteMode("a.txt", "hello", 0o600))
	info, err := m.Stat("a.txt")
	AssertNoError(t, err)
	AssertEqual(t, "a.txt", info.Name())
}
func TestAX7_Medium_WriteMode_Bad(t *T) {
	m := newCoreMedium(t.TempDir())
	AssertNoError(t, m.EnsureDir("dir"))
	err := m.WriteMode("dir", "hello", 0o600)
	AssertError(t, err)
}
func TestAX7_Medium_WriteMode_Ugly(t *T) {
	m := newCoreMedium(t.TempDir())
	AssertNoError(t, m.WriteMode("nested/empty.txt", "", 0o644))
	got, err := m.Read("nested/empty.txt")
	AssertNoError(t, err)
	AssertEqual(t, "", got)
}
func TestAX7_Medium_WriteStream_Good(t *T) {
	m := newCoreMedium(t.TempDir())
	w, err := m.WriteStream("a.txt")
	AssertNoError(t, err)
	_, err = w.Write([]byte("hello"))
	AssertNoError(t, err)
	AssertNoError(t, w.Close())
}
func TestAX7_Medium_WriteStream_Bad(t *T) {
	m := newCoreMedium(t.TempDir())
	AssertNoError(t, m.EnsureDir("dir"))
	w, err := m.WriteStream("dir")
	AssertError(t, err)
	AssertNil(t, w)
}
func TestAX7_Medium_WriteStream_Ugly(t *T) {
	m := newCoreMedium(t.TempDir())
	w, err := m.WriteStream("nested/a.txt")
	AssertNoError(t, err)
	AssertNoError(t, w.Close())
}
func TestAX7_NegotiateTransformer_Good(t *T) {
	body := []byte(`{"model":"gpt","messages":[{"role":"user","content":"hi"}]}`)
	got := NegotiateTransformer(body, "application/openai+json", "")
	AssertTrue(t, reflectTransformer[OpenAITransformer](got))
}
func TestAX7_NegotiateTransformer_Bad(t *T) {
	got := NegotiateTransformer([]byte(`{}`), "", "")
	AssertTrue(t, reflectTransformer[MCPNativeTransformer](got))
	AssertFalse(t, reflectTransformer[OpenAITransformer](got))
}
func TestAX7_NegotiateTransformer_Ugly(t *T) {
	got := NegotiateTransformer([]byte(`not-json`), "", "/mcp")
	AssertTrue(t, reflectTransformer[HoneypotTransformer](got))
	AssertFalse(t, reflectTransformer[AnthropicTransformer](got))
}
func TestAX7_New_Good(t *T) {
	svc, err := New(Options{WorkspaceRoot: t.TempDir()})
	AssertNoError(t, err)
	AssertNotNil(t, svc.Server())
	AssertTrue(t, len(svc.Tools()) > 0)
}
func TestAX7_New_Bad(t *T) {
	svc, err := New(Options{WorkspaceRoot: t.TempDir(), Subsystems: []Subsystem{nil}})
	AssertNoError(t, err)
	AssertLen(t, svc.Subsystems(), 0)
}
func TestAX7_New_Ugly(t *T) {
	svc, err := New(Options{Unrestricted: true})
	AssertNoError(t, err)
	AssertNotNil(t, svc.medium)
}
func TestAX7_NewProcessEventCallback_Good(t *T) {
	hub := ws.NewHub()
	cb := NewProcessEventCallback(hub)
	AssertNotNil(t, cb)
	AssertEqual(t, hub, cb.hub)
}
func TestAX7_NewProcessEventCallback_Bad(t *T) {
	cb := NewProcessEventCallback(nil)
	AssertNotNil(t, cb)
	AssertNil(t, cb.hub)
}
func TestAX7_NewProcessEventCallback_Ugly(t *T) {
	cb := NewProcessEventCallback(ws.NewHub())
	cb.OnProcessOutput("", "")
	AssertNotNil(t, cb.hub)
}
func TestAX7_NewProgressNotifier_Good(t *T) {
	req := &sdkmcp.CallToolRequest{}
	notifier := NewProgressNotifier(context.Background(), req)
	AssertNoError(t, notifier.Send(1, 2, "ok"))
}
func TestAX7_NewProgressNotifier_Bad(t *T) {
	notifier := NewProgressNotifier(context.Background(), nil)
	AssertNoError(t, notifier.Send(1, 2, "ok"))
	AssertNil(t, notifier.req)
}
func TestAX7_NewProgressNotifier_Ugly(t *T) {
	notifier := NewProgressNotifier(nil, &sdkmcp.CallToolRequest{})
	AssertNoError(t, notifier.Send(-1, 0, ""))
	AssertNotNil(t, notifier.req)
}
func TestAX7_NewTCPTransport_Good(t *T) {
	tr, err := NewTCPTransport("127.0.0.1:0")
	AssertNoError(t, err)
	defer tr.listener.Close()
	AssertNotNil(t, tr.listener)
}
func TestAX7_NewTCPTransport_Bad(t *T) {
	tr, err := NewTCPTransport("127.0.0.1:bad")
	AssertError(t, err)
	AssertNil(t, tr)
}
func TestAX7_NewTCPTransport_Ugly(t *T) {
	tr, err := NewTCPTransport(":0")
	AssertNoError(t, err)
	defer tr.listener.Close()
	AssertNotEmpty(t, tr.listener.Addr().String())
}
func TestAX7_NotifySession_Good(t *T) {
	err := NotifySession(context.Background(), nil, "method", map[string]any{"ok": true})
	AssertNoError(t, err)
	AssertNoError(t, NotifySession(context.Background(), nil, "method", nil))
}
func TestAX7_NotifySession_Bad(t *T) {
	err := NotifySession(context.Background(), nil, "", nil)
	AssertNoError(t, err)
	AssertNil(t, ProgressTokenFromRequest(nil))
}
func TestAX7_NotifySession_Ugly(t *T) {
	err := NotifySession(nil, nil, "method", map[string]any{})
	AssertNoError(t, err)
	AssertNoError(t, NotifySession(nil, nil, "", map[string]any{}))
}
func TestAX7_OpenAITransformer_Detect_Good(t *T) {
	body := []byte(`{"model":"gpt","messages":[{"role":"user","content":"hi"}]}`)
	got := (OpenAITransformer{}).Detect(body, "", "")
	AssertTrue(t, got)
}
func TestAX7_OpenAITransformer_Detect_Bad(t *T) {
	got := (OpenAITransformer{}).Detect([]byte(`{"model":"gpt"}`), "", "")
	AssertFalse(t, got)
	AssertFalse(t, (OpenAITransformer{}).Detect([]byte(`bad`), "", ""))
}
func TestAX7_OpenAITransformer_Detect_Ugly(t *T) {
	got := (OpenAITransformer{}).Detect(nil, "application/openai+json", "")
	AssertTrue(t, got)
	AssertTrue(t, (OpenAITransformer{}).Detect(nil, "", "/v1/chat/completions"))
}
func TestAX7_OpenAITransformer_Normalise_Good(t *T) {
	body := []byte(`{"model":"gpt","messages":[{"role":"user","content":"hi"}]}`)
	req, err := (OpenAITransformer{}).Normalise(body)
	AssertNoError(t, err)
	AssertEqual(t, "sampling/createMessage", req.Method)
}
func TestAX7_OpenAITransformer_Normalise_Bad(t *T) {
	req, err := (OpenAITransformer{}).Normalise([]byte(`{"messages":[]}`))
	AssertError(t, err)
	AssertEqual(t, MCPRequest{}, req)
}
func TestAX7_OpenAITransformer_Normalise_Ugly(t *T) {
	body := []byte(`{"model":"gpt","messages":[{"role":"assistant","tool_calls":[{"id":"1","type":"function","function":{"name":"echo","arguments":"{\"x\":1}"}}]}]}`)
	req, err := (OpenAITransformer{}).Normalise(body)
	AssertNoError(t, err)
	AssertEqual(t, "tools/call", req.Method)
}
func TestAX7_OpenAITransformer_Transform_Good(t *T) {
	out, err := (OpenAITransformer{}).Transform(MCPResult{ID: "1", Content: []MCPContent{{Type: "text", Text: "ok"}}})
	AssertNoError(t, err)
	AssertContains(t, string(out), "chat.completion")
}
func TestAX7_OpenAITransformer_Transform_Bad(t *T) {
	out, err := (OpenAITransformer{}).Transform(MCPResult{ToolCalls: []MCPToolCall{{Name: "bad", Arguments: map[string]any{"bad": make(chan int)}}}})
	AssertNoError(t, err)
	AssertContains(t, string(out), `"arguments":"{}"`)
}
func TestAX7_OpenAITransformer_Transform_Ugly(t *T) {
	out, err := (OpenAITransformer{}).Transform(MCPResult{})
	AssertNoError(t, err)
	AssertContains(t, string(out), `"content":""`)
}
func TestAX7_ProcessEventCallback_OnProcessOutput_Good(t *T) {
	hub := ws.NewHub()
	cb := NewProcessEventCallback(hub)
	cb.OnProcessOutput("proc", "line")
	AssertEqual(t, 1, hub.Stats().Channels)
}
func TestAX7_ProcessEventCallback_OnProcessOutput_Bad(t *T) {
	cb := NewProcessEventCallback(nil)
	AssertNotPanics(t, func() { cb.OnProcessOutput("proc", "line") })
	AssertNil(t, cb.hub)
}
func TestAX7_ProcessEventCallback_OnProcessOutput_Ugly(t *T) {
	hub := ws.NewHub()
	cb := NewProcessEventCallback(hub)
	cb.OnProcessOutput("", "")
	AssertEqual(t, 1, hub.Stats().Channels)
}
func TestAX7_ProcessEventCallback_OnProcessStatus_Good(t *T) {
	hub := ws.NewHub()
	cb := NewProcessEventCallback(hub)
	cb.OnProcessStatus("proc", "exited", 0)
	AssertEqual(t, 1, hub.Stats().Channels)
}
func TestAX7_ProcessEventCallback_OnProcessStatus_Bad(t *T) {
	cb := NewProcessEventCallback(nil)
	AssertNotPanics(t, func() { cb.OnProcessStatus("proc", "exited", 0) })
	AssertNil(t, cb.hub)
}
func TestAX7_ProcessEventCallback_OnProcessStatus_Ugly(t *T) {
	hub := ws.NewHub()
	cb := NewProcessEventCallback(hub)
	cb.OnProcessStatus("", "", -1)
	AssertEqual(t, 1, hub.Stats().Channels)
}
func TestAX7_ProgressNotifier_Send_Good(t *T) {
	notifier := NewProgressNotifier(context.Background(), &sdkmcp.CallToolRequest{})
	err := notifier.Send(1, 2, "ok")
	AssertNoError(t, err)
}
func TestAX7_ProgressNotifier_Send_Bad(t *T) {
	notifier := ProgressNotifier{}
	err := notifier.Send(1, 2, "ok")
	AssertNoError(t, err)
}
func TestAX7_ProgressNotifier_Send_Ugly(t *T) {
	notifier := NewProgressNotifier(nil, nil)
	err := notifier.Send(-1, 0, "")
	AssertNoError(t, err)
}
func TestAX7_ProgressTokenFromRequest_Good(t *T) {
	req := &sdkmcp.CallToolRequest{Params: &sdkmcp.CallToolParamsRaw{}}
	req.Params.SetProgressToken("tok")
	AssertEqual(t, "tok", ProgressTokenFromRequest(req))
}
func TestAX7_ProgressTokenFromRequest_Bad(t *T) {
	var req *sdkmcp.CallToolRequest
	AssertNil(t, ProgressTokenFromRequest(req))
	AssertNil(t, ProgressTokenFromRequest(&sdkmcp.CallToolRequest{}))
}
func TestAX7_ProgressTokenFromRequest_Ugly(t *T) {
	req := &sdkmcp.CallToolRequest{Params: &sdkmcp.CallToolParamsRaw{}}
	req.Params.SetProgressToken(42)
	AssertEqual(t, 42, ProgressTokenFromRequest(req))
}
func TestAX7_Register_Good(t *T) {
	c := core.New()
	r := Register(c)
	AssertTrue(t, r.OK)
	AssertNotNil(t, r.Value)
}
func TestAX7_Register_Bad(t *T) {
	AssertPanics(t, func() { _ = Register(nil) })
	c := core.New()
	AssertNotNil(t, c)
}
func TestAX7_Register_Ugly(t *T) {
	c := core.New()
	r := Register(c)
	AssertTrue(t, r.OK)
	AssertNotNil(t, r.Value)
}
func TestAX7_SendProgressNotification_Good(t *T) {
	req := &sdkmcp.CallToolRequest{Params: &sdkmcp.CallToolParamsRaw{}}
	err := SendProgressNotification(context.Background(), req, 1, 2, "ok")
	AssertNoError(t, err)
}
func TestAX7_SendProgressNotification_Bad(t *T) {
	err := SendProgressNotification(context.Background(), nil, 1, 2, "ok")
	AssertNoError(t, err)
	AssertNil(t, ProgressTokenFromRequest(nil))
}
func TestAX7_SendProgressNotification_Ugly(t *T) {
	req := &sdkmcp.CallToolRequest{Params: &sdkmcp.CallToolParamsRaw{}}
	req.Params.SetProgressToken("tok")
	err := SendProgressNotification(context.Background(), req, -1, 0, "")
	AssertNoError(t, err)
}
func TestAX7_Service_ChannelSend_Good(t *T) {
	svc := ax7NewServiceForTest(t, Options{WSHub: ws.NewHub()})
	AssertNotPanics(t, func() { svc.ChannelSend(context.Background(), "ax7", map[string]any{"ok": true}) })
	AssertNotNil(t, svc.WSHub())
}
func TestAX7_Service_ChannelSend_Bad(t *T) {
	var svc *Service
	AssertNotPanics(t, func() { svc.ChannelSend(context.Background(), "ax7", nil) })
	AssertNil(t, svc)
}
func TestAX7_Service_ChannelSend_Ugly(t *T) {
	svc := ax7NewServiceForTest(t, Options{})
	AssertNotPanics(t, func() { svc.ChannelSend(nil, "", nil) })
	AssertNil(t, svc.WSHub())
}
func TestAX7_Service_ChannelSendToClient_Good(t *T) {
	svc := ax7NewServiceForTest(t, Options{})
	AssertNotPanics(t, func() { svc.ChannelSendToClient(context.Background(), nil, "ax7", map[string]any{"ok": true}) })
	AssertNotNil(t, svc.Server())
}
func TestAX7_Service_ChannelSendToClient_Bad(t *T) {
	var svc *Service
	AssertNotPanics(t, func() { svc.ChannelSendToClient(context.Background(), nil, "ax7", nil) })
	AssertNil(t, svc)
}
func TestAX7_Service_ChannelSendToClient_Ugly(t *T) {
	svc := ax7NewServiceForTest(t, Options{})
	AssertNotPanics(t, func() { svc.ChannelSendToClient(nil, nil, "", nil) })
	count := 0
	for range svc.Sessions() {
		count++
	}
	AssertEqual(t, 0, count)
}
func TestAX7_Service_ChannelSendToSession_Good(t *T) {
	svc := ax7NewServiceForTest(t, Options{})
	AssertNotPanics(t, func() { svc.ChannelSendToSession(context.Background(), nil, "ax7", map[string]any{"ok": true}) })
	AssertNotNil(t, svc.Server())
}
func TestAX7_Service_ChannelSendToSession_Bad(t *T) {
	var svc *Service
	AssertNotPanics(t, func() { svc.ChannelSendToSession(context.Background(), nil, "ax7", nil) })
	AssertNil(t, svc)
}
func TestAX7_Service_ChannelSendToSession_Ugly(t *T) {
	svc := ax7NewServiceForTest(t, Options{})
	AssertNotPanics(t, func() { svc.ChannelSendToSession(nil, nil, "", nil) })
	count := 0
	for range svc.Sessions() {
		count++
	}
	AssertEqual(t, 0, count)
}
func TestAX7_Service_HandleIPCEvents_Good(t *T) {
	svc := ax7NewServiceForTest(t, Options{WSHub: ws.NewHub()})
	r := svc.HandleIPCEvents(nil, ChannelPush{Channel: "ax7", Data: map[string]any{"ok": true}})
	AssertTrue(t, r.OK)
}
func TestAX7_Service_HandleIPCEvents_Bad(t *T) {
	var svc *Service
	r := svc.HandleIPCEvents(nil, ChannelPush{Channel: "", Data: nil})
	AssertFalse(t, r.OK)
	AssertContains(t, r.Error(), "channel is required")
}
func TestAX7_Service_HandleIPCEvents_Ugly(t *T) {
	svc := ax7NewServiceForTest(t, Options{})
	r := svc.HandleIPCEvents(nil, ChannelPush{Channel: "ax7", Data: nil})
	AssertTrue(t, r.OK)
}
func TestAX7_Service_OnShutdown_Good(t *T) {
	svc := ax7NewServiceForTest(t, Options{})
	r := svc.OnShutdown(context.Background())
	AssertTrue(t, r.OK)
}
func TestAX7_Service_OnShutdown_Bad(t *T) {
	var svc *Service
	AssertPanics(t, func() { _ = svc.OnShutdown(context.Background()) })
	AssertNil(t, svc)
}
func TestAX7_Service_OnShutdown_Ugly(t *T) {
	svc := ax7NewServiceForTest(t, Options{})
	r := svc.OnShutdown(nil)
	AssertTrue(t, r.OK)
}
func TestAX7_Service_OnStartup_Good(t *T) {
	svc := ax7NewServiceForTest(t, Options{})
	r := svc.OnStartup(context.Background())
	AssertTrue(t, r.OK)
}
func TestAX7_Service_OnStartup_Bad(t *T) {
	var svc *Service
	r := svc.OnStartup(context.Background())
	AssertTrue(t, r.OK)
}
func TestAX7_Service_OnStartup_Ugly(t *T) {
	svc := ax7NewServiceForTest(t, Options{})
	r := svc.OnStartup(nil)
	AssertTrue(t, r.OK)
}
func TestAX7_Service_ProcessService_Good(t *T) {
	ps := &process.Service{}
	svc := ax7NewServiceForTest(t, Options{ProcessService: ps})
	AssertEqual(t, ps, svc.ProcessService())
}
func TestAX7_Service_ProcessService_Bad(t *T) {
	svc := ax7NewServiceForTest(t, Options{})
	AssertNil(t, svc.ProcessService())
	AssertNotNil(t, svc.Server())
}
func TestAX7_Service_ProcessService_Ugly(t *T) {
	var svc *Service
	AssertPanics(t, func() { _ = svc.ProcessService() })
	AssertNil(t, svc)
}
func TestAX7_Service_Run_Good(t *T) {
	svc := ax7NewServiceForTest(t, Options{})
	t.Setenv("MCP_ADDR", "127.0.0.1:0")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := svc.Run(ctx)
	AssertNoError(t, err)
}
func TestAX7_Service_Run_Bad(t *T) {
	var svc *Service
	AssertPanics(t, func() { _ = svc.Run(context.Background()) })
	AssertNil(t, svc)
}
func TestAX7_Service_Run_Ugly(t *T) {
	svc := ax7NewServiceForTest(t, Options{})
	t.Setenv("MCP_HTTP_ADDR", "127.0.0.1:bad")
	err := svc.Run(context.Background())
	AssertError(t, err)
}
func TestAX7_Service_SendNotificationToAllClients_Good(t *T) {
	svc := ax7NewServiceForTest(t, Options{})
	AssertNotPanics(t, func() {
		svc.SendNotificationToAllClients(context.Background(), "info", "ax7", map[string]any{"ok": true})
	})
	AssertNotNil(t, svc.Server())
}
func TestAX7_Service_SendNotificationToAllClients_Bad(t *T) {
	var svc *Service
	AssertNotPanics(t, func() { svc.SendNotificationToAllClients(context.Background(), "info", "ax7", nil) })
	AssertNil(t, svc)
}
func TestAX7_Service_SendNotificationToAllClients_Ugly(t *T) {
	svc := ax7NewServiceForTest(t, Options{})
	AssertNotPanics(t, func() { svc.SendNotificationToAllClients(nil, "", "", nil) })
	count := 0
	for range svc.Sessions() {
		count++
	}
	AssertEqual(t, 0, count)
}
func TestAX7_Service_SendNotificationToClient_Good(t *T) {
	svc := ax7NewServiceForTest(t, Options{})
	AssertNotPanics(t, func() { svc.SendNotificationToClient(context.Background(), nil, "info", "ax7", nil) })
	AssertNotNil(t, svc.Server())
}
func TestAX7_Service_SendNotificationToClient_Bad(t *T) {
	var svc *Service
	AssertNotPanics(t, func() { svc.SendNotificationToClient(context.Background(), nil, "info", "ax7", nil) })
	AssertNil(t, svc)
}
func TestAX7_Service_SendNotificationToClient_Ugly(t *T) {
	svc := ax7NewServiceForTest(t, Options{})
	AssertNotPanics(t, func() { svc.SendNotificationToClient(nil, nil, "", "", nil) })
	count := 0
	for range svc.Sessions() {
		count++
	}
	AssertEqual(t, 0, count)
}
func TestAX7_Service_SendNotificationToSession_Good(t *T) {
	svc := ax7NewServiceForTest(t, Options{})
	AssertNotPanics(t, func() { svc.SendNotificationToSession(context.Background(), nil, "info", "ax7", nil) })
	AssertNotNil(t, svc.Server())
}
func TestAX7_Service_SendNotificationToSession_Bad(t *T) {
	var svc *Service
	AssertNotPanics(t, func() { svc.SendNotificationToSession(context.Background(), nil, "info", "ax7", nil) })
	AssertNil(t, svc)
}
func TestAX7_Service_SendNotificationToSession_Ugly(t *T) {
	svc := ax7NewServiceForTest(t, Options{})
	AssertNotPanics(t, func() { svc.SendNotificationToSession(nil, nil, "", "", nil) })
	count := 0
	for range svc.Sessions() {
		count++
	}
	AssertEqual(t, 0, count)
}
func TestAX7_Service_ServeHTTP_Good(t *T) {
	svc := ax7NewServiceForTest(t, Options{})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := svc.ServeHTTP(ctx, "127.0.0.1:0")
	AssertNoError(t, err)
}
func TestAX7_Service_ServeHTTP_Bad(t *T) {
	svc := ax7NewServiceForTest(t, Options{})
	err := svc.ServeHTTP(context.Background(), "127.0.0.1:bad")
	AssertError(t, err)
}
func TestAX7_Service_ServeHTTP_Ugly(t *T) {
	svc := ax7NewServiceForTest(t, Options{})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := svc.ServeHTTP(ctx, "")
	AssertNoError(t, err)
}
func TestAX7_Service_ServeStdio_Good(t *T) {
	svc := ax7NewServiceForTest(t, Options{})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := svc.ServeStdio(ctx)
	AssertError(t, err)
}
func TestAX7_Service_ServeStdio_Bad(t *T) {
	var svc *Service
	AssertPanics(t, func() { _ = svc.ServeStdio(context.Background()) })
	AssertNil(t, svc)
}
func TestAX7_Service_ServeStdio_Ugly(t *T) {
	svc := ax7NewServiceForTest(t, Options{})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := svc.ServeStdio(ctx)
	AssertError(t, err)
}
func TestAX7_Service_ServeTCP_Good(t *T) {
	svc := ax7NewServiceForTest(t, Options{})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := svc.ServeTCP(ctx, "127.0.0.1:0")
	AssertNoError(t, err)
}
func TestAX7_Service_ServeTCP_Bad(t *T) {
	svc := ax7NewServiceForTest(t, Options{})
	err := svc.ServeTCP(context.Background(), "127.0.0.1:bad")
	AssertError(t, err)
}
func TestAX7_Service_ServeTCP_Ugly(t *T) {
	svc := ax7NewServiceForTest(t, Options{})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := svc.ServeTCP(ctx, "")
	AssertNoError(t, err)
}
func TestAX7_Service_ServeUnix_Good(t *T) {
	svc := ax7NewServiceForTest(t, Options{})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := svc.ServeUnix(ctx, filepath.Join(t.TempDir(), "sock"))
	AssertNoError(t, err)
}
func TestAX7_Service_ServeUnix_Bad(t *T) {
	svc := ax7NewServiceForTest(t, Options{})
	err := svc.ServeUnix(context.Background(), filepath.Join(t.TempDir(), "missing", "sock"))
	AssertError(t, err)
}
func TestAX7_Service_ServeUnix_Ugly(t *T) {
	svc := ax7NewServiceForTest(t, Options{})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := svc.ServeUnix(ctx, "")
	AssertError(t, err)
}
func TestAX7_Service_Server_Good(t *T) {
	svc := ax7NewServiceForTest(t, Options{})
	AssertNotNil(t, svc.Server())
	AssertTrue(t, len(svc.Tools()) > 0)
}
func TestAX7_Service_Server_Bad(t *T) {
	var svc *Service
	AssertPanics(t, func() { _ = svc.Server() })
	AssertNil(t, svc)
}
func TestAX7_Service_Server_Ugly(t *T) {
	svc := ax7NewServiceForTest(t, Options{})
	AssertEqual(t, svc.Server(), svc.Server())
	AssertNotNil(t, svc.Server())
}
func TestAX7_Service_Sessions_Good(t *T) {
	svc := ax7NewServiceForTest(t, Options{})
	count := 0
	for range svc.Sessions() {
		count++
	}
	AssertEqual(t, 0, count)
}
func TestAX7_Service_Sessions_Bad(t *T) {
	var svc *Service
	count := 0
	for range svc.Sessions() {
		count++
	}
	AssertEqual(t, 0, count)
}
func TestAX7_Service_Sessions_Ugly(t *T) {
	svc := ax7NewServiceForTest(t, Options{})
	count := 0
	for range svc.Sessions() {
		count++
	}
	AssertEqual(t, 0, count)
}
func TestAX7_Service_Shutdown_Good(t *T) {
	svc := ax7NewServiceForTest(t, Options{})
	err := svc.Shutdown(context.Background())
	AssertNoError(t, err)
}
func TestAX7_Service_Shutdown_Bad(t *T) {
	var svc *Service
	AssertPanics(t, func() { _ = svc.Shutdown(context.Background()) })
	AssertNil(t, svc)
}
func TestAX7_Service_Shutdown_Ugly(t *T) {
	svc := ax7NewServiceForTest(t, Options{})
	err := svc.Shutdown(nil)
	AssertNoError(t, err)
}
func TestAX7_Service_Subsystems_Good(t *T) {
	svc := ax7NewServiceForTest(t, Options{})
	AssertLen(t, svc.Subsystems(), 0)
	AssertNotNil(t, svc.Server())
}
func TestAX7_Service_Subsystems_Bad(t *T) {
	var svc *Service
	AssertPanics(t, func() { _ = svc.Subsystems() })
	AssertNil(t, svc)
}
func TestAX7_Service_Subsystems_Ugly(t *T) {
	svc := ax7NewServiceForTest(t, Options{Subsystems: []Subsystem{nil}})
	AssertLen(t, svc.Subsystems(), 0)
	AssertTrue(t, len(svc.Tools()) > 0)
}
func TestAX7_Service_SubsystemsSeq_Good(t *T) {
	svc := ax7NewServiceForTest(t, Options{})
	count := 0
	for range svc.SubsystemsSeq() {
		count++
	}
	AssertEqual(t, 0, count)
}
func TestAX7_Service_SubsystemsSeq_Bad(t *T) {
	var svc *Service
	AssertPanics(t, func() {
		for range svc.SubsystemsSeq() {
		}
	})
}
func TestAX7_Service_SubsystemsSeq_Ugly(t *T) {
	svc := ax7NewServiceForTest(t, Options{Subsystems: []Subsystem{nil}})
	count := 0
	for range svc.SubsystemsSeq() {
		count++
	}
	AssertEqual(t, 0, count)
}
func TestAX7_Service_Tools_Good(t *T) {
	svc := ax7NewServiceForTest(t, Options{})
	AssertTrue(t, len(svc.Tools()) > 0)
	AssertNotNil(t, svc.Server())
}
func TestAX7_Service_Tools_Bad(t *T) {
	var svc *Service
	AssertPanics(t, func() { _ = svc.Tools() })
	AssertNil(t, svc)
}
func TestAX7_Service_Tools_Ugly(t *T) {
	svc := ax7NewServiceForTest(t, Options{})
	got := svc.Tools()
	got[0].Name = "mutated"
	AssertNotEqual(t, "mutated", svc.Tools()[0].Name)
}
func TestAX7_Service_ToolsSeq_Good(t *T) {
	svc := ax7NewServiceForTest(t, Options{})
	count := 0
	for range svc.ToolsSeq() {
		count++
	}
	AssertTrue(t, count > 0)
}
func TestAX7_Service_ToolsSeq_Bad(t *T) {
	var svc *Service
	AssertPanics(t, func() {
		for range svc.ToolsSeq() {
		}
	})
	AssertNil(t, svc)
}
func TestAX7_Service_ToolsSeq_Ugly(t *T) {
	svc := ax7NewServiceForTest(t, Options{})
	count := 0
	for range svc.ToolsSeq() {
		count++
	}
	AssertEqual(t, len(svc.Tools()), count)
}
func TestAX7_Service_WSHub_Good(t *T) {
	hub := ws.NewHub()
	svc := ax7NewServiceForTest(t, Options{WSHub: hub})
	AssertEqual(t, hub, svc.WSHub())
}
func TestAX7_Service_WSHub_Bad(t *T) {
	svc := ax7NewServiceForTest(t, Options{})
	AssertNil(t, svc.WSHub())
	AssertNotNil(t, svc.Server())
}
func TestAX7_Service_WSHub_Ugly(t *T) {
	hub := ws.NewHub()
	svc := ax7NewServiceForTest(t, Options{WSHub: hub})
	AssertEqual(t, hub, svc.WSHub())
}
func TestAX7_Transport_Connect_Good(t *T) {
	left, right := net.Pipe()
	defer left.Close()
	defer right.Close()
	conn, err := (&connTransport{conn: left}).Connect(context.Background())
	AssertNoError(t, err)
	AssertNotNil(t, conn)
}
func TestAX7_Transport_Connect_Bad(t *T) {
	conn, err := (&connTransport{}).Connect(context.Background())
	AssertNoError(t, err)
	AssertNotNil(t, conn)
}
func TestAX7_Transport_Connect_Ugly(t *T) {
	left, right := net.Pipe()
	right.Close()
	conn, err := (&connTransport{conn: left}).Connect(nil)
	AssertNoError(t, err)
	AssertNotNil(t, conn)
}
func TestAX7_Writer_Close_Good(t *T) {
	var buf bytes.Buffer
	w := &lockedWriter{w: &buf}
	AssertNoError(t, w.Close())
}
func TestAX7_Writer_Close_Bad(t *T) {
	var w *lockedWriter
	AssertNoError(t, w.Close())
	AssertNil(t, w)
}
func TestAX7_Writer_Close_Ugly(t *T) {
	w := &lockedWriter{}
	AssertNoError(t, w.Close())
	AssertNil(t, w.w)
}
func TestAX7_Writer_Write_Good(t *T) {
	var buf bytes.Buffer
	w := &lockedWriter{w: &buf}
	n, err := w.Write([]byte("ok"))
	AssertNoError(t, err)
	AssertEqual(t, 2, n)
}
func TestAX7_Writer_Write_Bad(t *T) {
	w := &lockedWriter{}
	n, err := w.Write([]byte("x"))
	AssertError(t, err)
	AssertEqual(t, 0, n)
	AssertNil(t, w.w)
}
func TestAX7_Writer_Write_Ugly(t *T) {
	var buf bytes.Buffer
	w := &lockedWriter{w: &buf}
	n, err := w.Write(nil)
	AssertNoError(t, err)
	AssertEqual(t, 0, n)
}
