package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"net"
	"reflect"
	"slices"
	"testing"
	"time"

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
	raw, ok := caps["claude/channel"]
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
	if !reflect.DeepEqual(spec.Map(), want["claude/channel"].(map[string]any)) {
		t.Fatal("expected typed capability map to match wire-format descriptor")
	}

	gotChannels := ChannelCapabilityChannels()
	wantChannels := channelCapabilityChannels()
	if !slices.Equal(gotChannels, wantChannels) {
		t.Fatalf("expected public channel list to match internal definition: got %v want %v", gotChannels, wantChannels)
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
