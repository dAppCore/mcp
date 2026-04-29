package mcp

import (
	"bufio"
	"context"
	"github.com/goccy/go-json"
	"net"
	"reflect"
	"slices"
	"testing"
	"time"

	core "dappco.re/go"
	"dappco.re/go/ws"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type notificationReadResult struct {
	msg map[string]any
	err error
}

func connectNotificationSession(t *testing.T, svc *Service) (context.CancelFunc, *mcp.ServerSession, net.Conn) {
	t.Helper()

	serverConn, clientConn := net.Pipe()
	ctx, cancel := context.WithCancel(context.Background())

	session, err := svc.server.Connect(ctx, &connTransport{conn: serverConn}, nil)
	if err != nil {
		cancel()
		clientConn.Close()
		t.Fatalf("Connect() failed: %v", err)
	}

	return cancel, session, clientConn
}

func readNotificationMessage(t *testing.T, conn net.Conn) <-chan notificationReadResult {
	t.Helper()

	resultCh := make(chan notificationReadResult, 1)
	go func() {
		scanner := bufio.NewScanner(conn)
		scanner.Buffer(make([]byte, 64*1024), 10*1024*1024)

		if !scanner.Scan() {
			resultCh <- notificationReadResult{err: scanner.Err()}
			return
		}

		var msg map[string]any
		if err := json.Unmarshal(scanner.Bytes(), &msg); err != nil {
			resultCh <- notificationReadResult{err: err}
			return
		}

		resultCh <- notificationReadResult{msg: msg}
	}()

	return resultCh
}

func readNotificationMessageUntil(t *testing.T, conn net.Conn, match func(map[string]any) bool) <-chan notificationReadResult {
	t.Helper()

	resultCh := make(chan notificationReadResult, 1)
	scanner := bufio.NewScanner(conn)
	scanner.Buffer(make([]byte, 64*1024), 10*1024*1024)

	go func() {
		for scanner.Scan() {
			var msg map[string]any
			if err := json.Unmarshal(scanner.Bytes(), &msg); err != nil {
				resultCh <- notificationReadResult{err: err}
				return
			}
			if match(msg) {
				resultCh <- notificationReadResult{msg: msg}
				return
			}
		}

		if err := scanner.Err(); err != nil {
			resultCh <- notificationReadResult{err: err}
			return
		}
		resultCh <- notificationReadResult{err: context.DeadlineExceeded}
	}()

	return resultCh
}

func TestSendNotificationToAllClients_Good(t *testing.T) {
	svc, err := New(Options{})
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	ctx := context.Background()
	svc.SendNotificationToAllClients(ctx, "info", "test", map[string]any{
		"event": ChannelBuildComplete,
	})
}

func TestNotificationMethods_Good_NilService(t *testing.T) {
	var svc *Service

	ctx := context.Background()
	svc.SendNotificationToAllClients(ctx, "info", "test", map[string]any{"ok": true})
	svc.SendNotificationToSession(ctx, nil, "info", "test", map[string]any{"ok": true})
	svc.ChannelSend(ctx, ChannelBuildComplete, map[string]any{"ok": true})
	svc.ChannelSendToSession(ctx, nil, ChannelBuildComplete, map[string]any{"ok": true})

	for range svc.Sessions() {
		t.Fatal("expected no sessions from nil service")
	}
}

func TestNotificationMethods_Good_NilServer(t *testing.T) {
	svc := &Service{}

	ctx := context.Background()
	svc.SendNotificationToAllClients(ctx, "info", "test", map[string]any{"ok": true})
	svc.SendNotificationToSession(ctx, nil, "info", "test", map[string]any{"ok": true})
	svc.ChannelSend(ctx, ChannelBuildComplete, map[string]any{"ok": true})
	svc.ChannelSendToSession(ctx, nil, ChannelBuildComplete, map[string]any{"ok": true})

	for range svc.Sessions() {
		t.Fatal("expected no sessions from service without a server")
	}
}

func TestSessions_Good_ReturnsSnapshot(t *testing.T) {
	svc, err := New(Options{})
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	cancel, session, _ := connectNotificationSession(t, svc)
	snapshot := svc.Sessions()

	cancel()
	session.Close()

	var sessions []*mcp.ServerSession
	for session := range snapshot {
		sessions = append(sessions, session)
	}

	if len(sessions) != 1 {
		t.Fatalf("expected snapshot to retain one session, got %d", len(sessions))
	}
	if sessions[0] == nil {
		t.Fatal("expected snapshot session to be non-nil")
	}
}

func TestNotificationMethods_Good_NilContext(t *testing.T) {
	svc, err := New(Options{})
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	svc.SendNotificationToAllClients(nil, "info", "test", map[string]any{"ok": true})
	svc.SendNotificationToSession(nil, nil, "info", "test", map[string]any{"ok": true})
	svc.ChannelSend(nil, ChannelBuildComplete, map[string]any{"ok": true})
	svc.ChannelSendToSession(nil, nil, ChannelBuildComplete, map[string]any{"ok": true})
}

func TestSendNotificationToAllClients_Good_CustomNotification(t *testing.T) {
	svc, err := New(Options{})
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	serverConn, clientConn := net.Pipe()
	defer clientConn.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	session, err := svc.server.Connect(ctx, &connTransport{conn: serverConn}, nil)
	if err != nil {
		t.Fatalf("Connect() failed: %v", err)
	}
	defer session.Close()

	clientConn.SetDeadline(time.Now().Add(5 * time.Second))

	read := readNotificationMessageUntil(t, clientConn, func(msg map[string]any) bool {
		return msg["method"] == LoggingNotificationMethod
	})

	sent := make(chan struct{})
	go func() {
		svc.SendNotificationToAllClients(ctx, "info", "test", map[string]any{
			"event": ChannelBuildComplete,
		})
		close(sent)
	}()

	select {
	case <-sent:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for notification send to complete")
	}

	res := <-read
	if res.err != nil {
		t.Fatalf("failed to read notification: %v", res.err)
	}
	msg := res.msg
	if msg["method"] != LoggingNotificationMethod {
		t.Fatalf("expected method %q, got %v", LoggingNotificationMethod, msg["method"])
	}

	params, ok := msg["params"].(map[string]any)
	if !ok {
		t.Fatalf("expected params object, got %T", msg["params"])
	}
	if params["logger"] != "test" {
		t.Fatalf("expected logger test, got %v", params["logger"])
	}
	if params["level"] != "info" {
		t.Fatalf("expected level info, got %v", params["level"])
	}
	data, ok := params["data"].(map[string]any)
	if !ok {
		t.Fatalf("expected data object, got %T", params["data"])
	}
	if data["event"] != ChannelBuildComplete {
		t.Fatalf("expected event %s, got %v", ChannelBuildComplete, data["event"])
	}
}

func TestChannelSend_Good(t *testing.T) {
	svc, err := New(Options{})
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	ctx := context.Background()
	svc.ChannelSend(ctx, ChannelBuildComplete, map[string]any{
		"repo": "go-io",
	})
}

func TestChannelSendToSession_Good_GuardNilSession(t *testing.T) {
	svc, err := New(Options{})
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	ctx := context.Background()
	svc.ChannelSendToSession(ctx, nil, ChannelAgentStatus, map[string]any{
		"ok": true,
	})
}

func TestSendNotificationToSession_Good_GuardNilSession(t *testing.T) {
	svc, err := New(Options{})
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	ctx := context.Background()
	svc.SendNotificationToSession(ctx, nil, "info", "test", map[string]any{
		"ok": true,
	})
}

func TestChannelSendToSession_Good_CustomNotification(t *testing.T) {
	svc, err := New(Options{})
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	serverConn, clientConn := net.Pipe()
	defer clientConn.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	session, err := svc.server.Connect(ctx, &connTransport{conn: serverConn}, nil)
	if err != nil {
		t.Fatalf("Connect() failed: %v", err)
	}
	defer session.Close()

	clientConn.SetDeadline(time.Now().Add(5 * time.Second))

	read := readNotificationMessageUntil(t, clientConn, func(msg map[string]any) bool {
		return msg["method"] == ChannelNotificationMethod
	})

	sent := make(chan struct{})
	go func() {
		svc.ChannelSendToSession(ctx, session, ChannelBuildComplete, map[string]any{
			"repo": "go-io",
		})
		close(sent)
	}()

	select {
	case <-sent:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for notification send to complete")
	}

	res := <-read
	if res.err != nil {
		t.Fatalf("failed to read custom notification: %v", res.err)
	}
	msg := res.msg
	if msg["method"] != ChannelNotificationMethod {
		t.Fatalf("expected method %q, got %v", ChannelNotificationMethod, msg["method"])
	}

	params, ok := msg["params"].(map[string]any)
	if !ok {
		t.Fatalf("expected params object, got %T", msg["params"])
	}
	if params["channel"] != ChannelBuildComplete {
		t.Fatalf("expected channel %s, got %v", ChannelBuildComplete, params["channel"])
	}
	payload, ok := params["data"].(map[string]any)
	if !ok {
		t.Fatalf("expected data object, got %T", params["data"])
	}
	if payload["repo"] != "go-io" {
		t.Fatalf("expected repo go-io, got %v", payload["repo"])
	}
}

func TestChannelSendToClient_Good_CustomNotification(t *testing.T) {
	svc, err := New(Options{})
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	serverConn, clientConn := net.Pipe()
	defer clientConn.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	session, err := svc.server.Connect(ctx, &connTransport{conn: serverConn}, nil)
	if err != nil {
		t.Fatalf("Connect() failed: %v", err)
	}
	defer session.Close()

	clientConn.SetDeadline(time.Now().Add(5 * time.Second))

	read := readNotificationMessageUntil(t, clientConn, func(msg map[string]any) bool {
		return msg["method"] == ChannelNotificationMethod
	})

	sent := make(chan struct{})
	go func() {
		svc.ChannelSendToClient(ctx, session, ChannelBuildComplete, map[string]any{
			"repo": "go-io",
		})
		close(sent)
	}()

	select {
	case <-sent:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for notification send to complete")
	}

	res := <-read
	if res.err != nil {
		t.Fatalf("failed to read custom notification: %v", res.err)
	}
	msg := res.msg
	if msg["method"] != ChannelNotificationMethod {
		t.Fatalf("expected method %q, got %v", ChannelNotificationMethod, msg["method"])
	}
}

func TestSendNotificationToClient_Good_CustomNotification(t *testing.T) {
	svc, err := New(Options{})
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	serverConn, clientConn := net.Pipe()
	defer clientConn.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	session, err := svc.server.Connect(ctx, &connTransport{conn: serverConn}, nil)
	if err != nil {
		t.Fatalf("Connect() failed: %v", err)
	}
	defer session.Close()

	clientConn.SetDeadline(time.Now().Add(5 * time.Second))

	read := readNotificationMessageUntil(t, clientConn, func(msg map[string]any) bool {
		return msg["method"] == LoggingNotificationMethod
	})

	sent := make(chan struct{})
	go func() {
		svc.SendNotificationToClient(ctx, session, "info", "test", map[string]any{
			"event": ChannelBuildComplete,
		})
		close(sent)
	}()

	select {
	case <-sent:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for notification send to complete")
	}

	res := <-read
	if res.err != nil {
		t.Fatalf("failed to read notification: %v", res.err)
	}
	msg := res.msg
	if msg["method"] != LoggingNotificationMethod {
		t.Fatalf("expected method %q, got %v", LoggingNotificationMethod, msg["method"])
	}
}

func TestChannelCapability_Good(t *testing.T) {
	caps := channelCapability()
	raw, ok := caps[ClaudeChannelCapabilityName]
	if !ok {
		t.Fatal("expected claude/channel capability entry")
	}

	cap, ok := raw.(map[string]any)
	if !ok {
		t.Fatalf("expected claude/channel to be a map, got %T", raw)
	}

	if cap["version"] == nil || cap["description"] == nil {
		t.Fatalf("expected capability to include version and description: %#v", cap)
	}

	channels, ok := cap["channels"].([]string)
	if !ok {
		t.Fatalf("expected channels to be []string, got %T", cap["channels"])
	}
	if len(channels) == 0 {
		t.Fatal("expected at least one channel in capability definition")
	}

	want := channelCapabilityChannels()
	if got, wantLen := len(channels), len(want); got != wantLen {
		t.Fatalf("expected %d channels, got %d", wantLen, got)
	}

	for _, channel := range want {
		if !slices.Contains(channels, channel) {
			t.Fatalf("expected channel %q to be advertised in capability definition", channel)
		}
	}
}

func TestChannelCapability_Good_PublicHelpers(t *testing.T) {
	got := ChannelCapability()
	want := channelCapability()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected public capability helper to match internal definition")
	}

	spec := ClaudeChannelCapability()
	if spec.Version != "1" {
		t.Fatalf("expected typed capability version 1, got %q", spec.Version)
	}
	if spec.Description == "" {
		t.Fatal("expected typed capability description to be populated")
	}
	if !slices.Equal(spec.Channels, channelCapabilityChannels()) {
		t.Fatalf("expected typed capability channels to match: got %v want %v", spec.Channels, channelCapabilityChannels())
	}
	if !reflect.DeepEqual(spec.Map(), want[ClaudeChannelCapabilityName].(map[string]any)) {
		t.Fatal("expected typed capability map to match wire-format descriptor")
	}

	gotChannels := ChannelCapabilityChannels()
	wantChannels := channelCapabilityChannels()
	if !slices.Equal(gotChannels, wantChannels) {
		t.Fatalf("expected public channel list to match internal definition: got %v want %v", gotChannels, wantChannels)
	}
}

func TestChannelCapabilitySpec_Map_Good_ClonesChannels(t *testing.T) {
	spec := ClaudeChannelCapability()
	mapped := spec.Map()

	channels, ok := mapped["channels"].([]string)
	if !ok {
		t.Fatalf("expected channels to be []string, got %T", mapped["channels"])
	}
	if len(channels) == 0 {
		t.Fatal("expected non-empty channels slice")
	}

	spec.Channels[0] = "mutated.channel"
	if channels[0] == "mutated.channel" {
		t.Fatal("expected Map() to clone the channels slice")
	}
}

func TestSendNotificationToAllClients_Good_BroadcastsToMultipleSessions(t *testing.T) {
	svc, err := New(Options{})
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cancel1, session1, clientConn1 := connectNotificationSession(t, svc)
	defer cancel1()
	defer session1.Close()
	defer clientConn1.Close()

	cancel2, session2, clientConn2 := connectNotificationSession(t, svc)
	defer cancel2()
	defer session2.Close()
	defer clientConn2.Close()

	read1 := readNotificationMessage(t, clientConn1)
	read2 := readNotificationMessage(t, clientConn2)

	sent := make(chan struct{})
	go func() {
		svc.SendNotificationToAllClients(ctx, "info", "test", map[string]any{
			"event": ChannelBuildComplete,
		})
		close(sent)
	}()

	select {
	case <-sent:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for broadcast to complete")
	}

	res1 := <-read1
	if res1.err != nil {
		t.Fatalf("failed to read notification from session 1: %v", res1.err)
	}
	res2 := <-read2
	if res2.err != nil {
		t.Fatalf("failed to read notification from session 2: %v", res2.err)
	}

	for idx, res := range []notificationReadResult{res1, res2} {
		if res.msg["method"] != LoggingNotificationMethod {
			t.Fatalf("session %d: expected method %q, got %v", idx+1, LoggingNotificationMethod, res.msg["method"])
		}

		params, ok := res.msg["params"].(map[string]any)
		if !ok {
			t.Fatalf("session %d: expected params object, got %T", idx+1, res.msg["params"])
		}
		if params["logger"] != "test" {
			t.Fatalf("session %d: expected logger test, got %v", idx+1, params["logger"])
		}
	}
}

// moved AX-7 triplet TestNotify_ChannelCapability_Good
func TestNotify_ChannelCapability_Good(t *T) {
	got := ChannelCapability()
	AssertNotNil(t, got[ClaudeChannelCapabilityName])
	AssertLen(t, got, 1)
}

// moved AX-7 triplet TestNotify_ChannelCapability_Bad
func TestNotify_ChannelCapability_Bad(t *T) {
	got := ChannelCapability()
	AssertNil(t, got["missing/channel"])
	AssertNotNil(t, got[ClaudeChannelCapabilityName])
}

// moved AX-7 triplet TestNotify_ChannelCapability_Ugly
func TestNotify_ChannelCapability_Ugly(t *T) {
	got := ChannelCapability()
	got[ClaudeChannelCapabilityName] = "mutated"
	AssertNotEqual(t, got[ClaudeChannelCapabilityName], ChannelCapability()[ClaudeChannelCapabilityName])
}

// moved AX-7 triplet TestNotify_ChannelCapabilityChannels_Good
func TestNotify_ChannelCapabilityChannels_Good(t *T) {
	got := ChannelCapabilityChannels()
	AssertContains(t, got, ChannelAgentStatus)
	AssertContains(t, got, ChannelProcessOutput)
}

// moved AX-7 triplet TestNotify_ChannelCapabilityChannels_Bad
func TestNotify_ChannelCapabilityChannels_Bad(t *T) {
	got := ChannelCapabilityChannels()
	AssertFalse(t, Contains(Join(",", got...), "missing.channel"))
	AssertTrue(t, len(got) > 0)
}

// moved AX-7 triplet TestNotify_ChannelCapabilityChannels_Ugly
func TestNotify_ChannelCapabilityChannels_Ugly(t *T) {
	got := ChannelCapabilityChannels()
	got[0] = "mutated"
	AssertNotEqual(t, "mutated", ChannelCapabilityChannels()[0])
}

// moved AX-7 triplet TestNotify_ChannelCapabilitySpec_Map_Good
func TestNotify_ChannelCapabilitySpec_Map_Good(t *T) {
	spec := ChannelCapabilitySpec{Version: "1", Description: "d", Channels: []string{"a"}}
	got := spec.Map()
	AssertEqual(t, "1", got["version"])
	AssertEqual(t, "d", got["description"])
}

// moved AX-7 triplet TestNotify_ChannelCapabilitySpec_Map_Bad
func TestNotify_ChannelCapabilitySpec_Map_Bad(t *T) {
	got := (ChannelCapabilitySpec{}).Map()
	AssertEqual(t, "", got["version"])
	AssertEqual(t, []string(nil), got["channels"])
}

// moved AX-7 triplet TestNotify_ChannelCapabilitySpec_Map_Ugly
func TestNotify_ChannelCapabilitySpec_Map_Ugly(t *T) {
	spec := ChannelCapabilitySpec{Channels: []string{"a"}}
	channels := spec.Map()["channels"].([]string)
	channels[0] = "mutated"
	AssertEqual(t, "a", spec.Channels[0])
}

// moved AX-7 triplet TestNotify_ClaudeChannelCapability_Good
func TestNotify_ClaudeChannelCapability_Good(t *T) {
	got := ClaudeChannelCapability()
	AssertEqual(t, "1", got.Version)
	AssertContains(t, got.Channels, ChannelBrainRecallDone)
}

// moved AX-7 triplet TestNotify_ClaudeChannelCapability_Bad
func TestNotify_ClaudeChannelCapability_Bad(t *T) {
	got := ClaudeChannelCapability()
	AssertNotEmpty(t, got.Description)
	AssertFalse(t, Contains(Join(",", got.Channels...), "missing.channel"))
}

// moved AX-7 triplet TestNotify_ClaudeChannelCapability_Ugly
func TestNotify_ClaudeChannelCapability_Ugly(t *T) {
	got := ClaudeChannelCapability()
	got.Channels[0] = "mutated"
	AssertNotEqual(t, "mutated", ClaudeChannelCapability().Channels[0])
}

// moved AX-7 triplet TestNotify_Error_Error_Good
func TestNotify_Error_Error_Good(t *T) {
	err := &notificationError{message: "boom"}
	AssertEqual(t, "boom", err.Error())
	AssertNotNil(t, err)
}

// moved AX-7 triplet TestNotify_Error_Error_Bad
func TestNotify_Error_Error_Bad(t *T) {
	err := &notificationError{}
	AssertEqual(t, "", err.Error())
	AssertNotNil(t, err)
}

// moved AX-7 triplet TestNotify_Error_Error_Ugly
func TestNotify_Error_Error_Ugly(t *T) {
	err := &notificationError{message: repeatString("x", 32)}
	AssertEqual(t, repeatString("x", 32), err.Error())
	AssertEqual(t, 32, len(err.Error()))
}

// moved AX-7 triplet TestNotify_NotifySession_Good
func TestNotify_NotifySession_Good(t *T) {
	err := NotifySession(context.Background(), nil, "method", map[string]any{"ok": true})
	AssertNoError(t, err)
	AssertNoError(t, NotifySession(context.Background(), nil, "method", nil))
}

// moved AX-7 triplet TestNotify_NotifySession_Bad
func TestNotify_NotifySession_Bad(t *T) {
	err := NotifySession(context.Background(), nil, "", nil)
	AssertNoError(t, err)
	AssertNil(t, ProgressTokenFromRequest(nil))
}

// moved AX-7 triplet TestNotify_NotifySession_Ugly
func TestNotify_NotifySession_Ugly(t *T) {
	err := NotifySession(nil, nil, "method", map[string]any{})
	AssertNoError(t, err)
	AssertNoError(t, NotifySession(nil, nil, "", map[string]any{}))
}

// moved AX-7 triplet TestNotify_Service_ChannelSend_Good
func TestNotify_Service_ChannelSend_Good(t *T) {
	svc := newServiceForTest(t, Options{WSHub: ws.NewHub()})
	AssertNotPanics(t, func() { svc.ChannelSend(context.Background(), "ax7", map[string]any{"ok": true}) })
	AssertNotNil(t, svc.WSHub())
}

// moved AX-7 triplet TestNotify_Service_ChannelSend_Bad
func TestNotify_Service_ChannelSend_Bad(t *T) {
	var svc *Service
	AssertNotPanics(t, func() { svc.ChannelSend(context.Background(), "ax7", nil) })
	AssertNil(t, svc)
}

// moved AX-7 triplet TestNotify_Service_ChannelSend_Ugly
func TestNotify_Service_ChannelSend_Ugly(t *T) {
	svc := newServiceForTest(t, Options{})
	AssertNotPanics(t, func() { svc.ChannelSend(nil, "", nil) })
	AssertNil(t, svc.WSHub())
}

// moved AX-7 triplet TestNotify_Service_ChannelSendToClient_Good
func TestNotify_Service_ChannelSendToClient_Good(t *T) {
	svc := newServiceForTest(t, Options{})
	AssertNotPanics(t, func() { svc.ChannelSendToClient(context.Background(), nil, "ax7", map[string]any{"ok": true}) })
	AssertNotNil(t, svc.Server())
}

// moved AX-7 triplet TestNotify_Service_ChannelSendToClient_Bad
func TestNotify_Service_ChannelSendToClient_Bad(t *T) {
	var svc *Service
	AssertNotPanics(t, func() { svc.ChannelSendToClient(context.Background(), nil, "ax7", nil) })
	AssertNil(t, svc)
}

// moved AX-7 triplet TestNotify_Service_ChannelSendToClient_Ugly
func TestNotify_Service_ChannelSendToClient_Ugly(t *T) {
	svc := newServiceForTest(t, Options{})
	AssertNotPanics(t, func() { svc.ChannelSendToClient(nil, nil, "", nil) })
	count := 0
	for range svc.Sessions() {
		count++
	}
	AssertEqual(t, 0, count)
}

// moved AX-7 triplet TestNotify_Service_ChannelSendToSession_Good
func TestNotify_Service_ChannelSendToSession_Good(t *T) {
	svc := newServiceForTest(t, Options{})
	AssertNotPanics(t, func() { svc.ChannelSendToSession(context.Background(), nil, "ax7", map[string]any{"ok": true}) })
	AssertNotNil(t, svc.Server())
}

// moved AX-7 triplet TestNotify_Service_ChannelSendToSession_Bad
func TestNotify_Service_ChannelSendToSession_Bad(t *T) {
	var svc *Service
	AssertNotPanics(t, func() { svc.ChannelSendToSession(context.Background(), nil, "ax7", nil) })
	AssertNil(t, svc)
}

// moved AX-7 triplet TestNotify_Service_ChannelSendToSession_Ugly
func TestNotify_Service_ChannelSendToSession_Ugly(t *T) {
	svc := newServiceForTest(t, Options{})
	AssertNotPanics(t, func() { svc.ChannelSendToSession(nil, nil, "", nil) })
	count := 0
	for range svc.Sessions() {
		count++
	}
	AssertEqual(t, 0, count)
}

// moved AX-7 triplet TestNotify_Service_SendNotificationToAllClients_Good
func TestNotify_Service_SendNotificationToAllClients_Good(t *T) {
	svc := newServiceForTest(t, Options{})
	AssertNotPanics(t, func() {
		svc.SendNotificationToAllClients(context.Background(), "info", "ax7", map[string]any{"ok": true})
	})
	AssertNotNil(t, svc.Server())
}

// moved AX-7 triplet TestNotify_Service_SendNotificationToAllClients_Bad
func TestNotify_Service_SendNotificationToAllClients_Bad(t *T) {
	var svc *Service
	AssertNotPanics(t, func() { svc.SendNotificationToAllClients(context.Background(), "info", "ax7", nil) })
	AssertNil(t, svc)
}

// moved AX-7 triplet TestNotify_Service_SendNotificationToAllClients_Ugly
func TestNotify_Service_SendNotificationToAllClients_Ugly(t *T) {
	svc := newServiceForTest(t, Options{})
	AssertNotPanics(t, func() { svc.SendNotificationToAllClients(nil, "", "", nil) })
	count := 0
	for range svc.Sessions() {
		count++
	}
	AssertEqual(t, 0, count)
}

// moved AX-7 triplet TestNotify_Service_SendNotificationToClient_Good
func TestNotify_Service_SendNotificationToClient_Good(t *T) {
	svc := newServiceForTest(t, Options{})
	AssertNotPanics(t, func() { svc.SendNotificationToClient(context.Background(), nil, "info", "ax7", nil) })
	AssertNotNil(t, svc.Server())
}

// moved AX-7 triplet TestNotify_Service_SendNotificationToClient_Bad
func TestNotify_Service_SendNotificationToClient_Bad(t *T) {
	var svc *Service
	AssertNotPanics(t, func() { svc.SendNotificationToClient(context.Background(), nil, "info", "ax7", nil) })
	AssertNil(t, svc)
}

// moved AX-7 triplet TestNotify_Service_SendNotificationToClient_Ugly
func TestNotify_Service_SendNotificationToClient_Ugly(t *T) {
	svc := newServiceForTest(t, Options{})
	AssertNotPanics(t, func() { svc.SendNotificationToClient(nil, nil, "", "", nil) })
	count := 0
	for range svc.Sessions() {
		count++
	}
	AssertEqual(t, 0, count)
}

// moved AX-7 triplet TestNotify_Service_SendNotificationToSession_Good
func TestNotify_Service_SendNotificationToSession_Good(t *T) {
	svc := newServiceForTest(t, Options{})
	AssertNotPanics(t, func() { svc.SendNotificationToSession(context.Background(), nil, "info", "ax7", nil) })
	AssertNotNil(t, svc.Server())
}

// moved AX-7 triplet TestNotify_Service_SendNotificationToSession_Bad
func TestNotify_Service_SendNotificationToSession_Bad(t *T) {
	var svc *Service
	AssertNotPanics(t, func() { svc.SendNotificationToSession(context.Background(), nil, "info", "ax7", nil) })
	AssertNil(t, svc)
}

// moved AX-7 triplet TestNotify_Service_SendNotificationToSession_Ugly
func TestNotify_Service_SendNotificationToSession_Ugly(t *T) {
	svc := newServiceForTest(t, Options{})
	AssertNotPanics(t, func() { svc.SendNotificationToSession(nil, nil, "", "", nil) })
	count := 0
	for range svc.Sessions() {
		count++
	}
	AssertEqual(t, 0, count)
}

// moved AX-7 triplet TestNotify_Service_Sessions_Good
func TestNotify_Service_Sessions_Good(t *T) {
	svc := newServiceForTest(t, Options{})
	count := 0
	for range svc.Sessions() {
		count++
	}
	AssertEqual(t, 0, count)
}

// moved AX-7 triplet TestNotify_Service_Sessions_Bad
func TestNotify_Service_Sessions_Bad(t *T) {
	var svc *Service
	count := 0
	for range svc.Sessions() {
		count++
	}
	AssertEqual(t, 0, count)
}

// moved AX-7 triplet TestNotify_Service_Sessions_Ugly
func TestNotify_Service_Sessions_Ugly(t *T) {
	svc := newServiceForTest(t, Options{})
	svc.ChannelSend(context.Background(), ChannelAgentStatus, map[string]any{"ok": true})
	count := 0
	for range svc.Sessions() {
		count++
	}
	AssertEqual(t, 0, count)
}

// moved AX-7 triplet TestNotify_Writer_Close_Good
func TestNotify_Writer_Close_Good(t *T) {
	buf := core.NewBuffer()
	w := &lockedWriter{w: buf}
	AssertNoError(t, w.Close())
}

// moved AX-7 triplet TestNotify_Writer_Close_Bad
func TestNotify_Writer_Close_Bad(t *T) {
	var w *lockedWriter
	AssertNoError(t, w.Close())
	AssertNil(t, w)
}

// moved AX-7 triplet TestNotify_Writer_Close_Ugly
func TestNotify_Writer_Close_Ugly(t *T) {
	w := &lockedWriter{}
	AssertNoError(t, w.Close())
	AssertNil(t, w.w)
}

// moved AX-7 triplet TestNotify_Writer_Write_Good
func TestNotify_Writer_Write_Good(t *T) {
	buf := core.NewBuffer()
	w := &lockedWriter{w: buf}
	n, err := w.Write([]byte("ok"))
	AssertNoError(t, err)
	AssertEqual(t, 2, n)
}

// moved AX-7 triplet TestNotify_Writer_Write_Bad
func TestNotify_Writer_Write_Bad(t *T) {
	w := &lockedWriter{}
	n, err := w.Write([]byte("x"))
	AssertError(t, err)
	AssertEqual(t, 0, n)
	AssertNil(t, w.w)
}

// moved AX-7 triplet TestNotify_Writer_Write_Ugly
func TestNotify_Writer_Write_Ugly(t *T) {
	buf := core.NewBuffer()
	w := &lockedWriter{w: buf}
	n, err := w.Write(nil)
	AssertNoError(t, err)
	AssertEqual(t, 0, n)
}
