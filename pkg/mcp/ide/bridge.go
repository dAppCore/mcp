// SPDX-License-Identifier: EUPL-1.2

package ide

import (
	"context"
	"net/http"
	"sync"
	"time"

	core "dappco.re/go"
	"dappco.re/go/ws"
	"github.com/gorilla/websocket"
)

// BridgeMessage is the wire format between the IDE bridge and Laravel.
//
//	msg := BridgeMessage{
//	    Type: "chat_send",
//	    SessionID: "sess-42",
//	    Data: "hello",
//	}
type BridgeMessage struct {
	Type      string    `json:"type"`
	Channel   string    `json:"channel,omitempty"`
	SessionID string    `json:"sessionId,omitempty"`
	Data      any       `json:"data,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// Bridge maintains a WebSocket connection to the Laravel core-agentic
// backend and forwards responses to a local ws.Hub.
//
//	bridge := NewBridge(hub, cfg)
type Bridge struct {
	cfg  Config
	hub  *ws.Hub
	conn *websocket.Conn

	mu        sync.Mutex
	connected bool
	cancel    context.CancelFunc
	observers []func(BridgeMessage)
}

// NewBridge creates a bridge that will connect to the Laravel backend and
// forward incoming messages to the provided ws.Hub channels.
//
//	bridge := NewBridge(hub, cfg)
func NewBridge(hub *ws.Hub, cfg Config) *Bridge {
	return &Bridge{cfg: cfg, hub: hub}
}

// SetObserver registers a callback for inbound bridge messages.
//
//	bridge.SetObserver(func(msg BridgeMessage) {
//	    fmt.Println(msg.Type)
//	})
func (b *Bridge) SetObserver(fn func(BridgeMessage)) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if fn == nil {
		b.observers = nil
		return
	}
	b.observers = []func(BridgeMessage){fn}
}

// AddObserver registers an additional bridge observer.
// Observers are invoked in registration order after each inbound message.
//
//	bridge.AddObserver(func(msg BridgeMessage) { core.Println(msg.Type) })
func (b *Bridge) AddObserver(fn func(BridgeMessage)) {
	if fn == nil {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	b.observers = append(b.observers, fn)
}

// Start begins the connection loop in a background goroutine.
// Call Shutdown to stop it.
//
//	bridge.Start(ctx)
func (b *Bridge) Start(ctx context.Context) {
	ctx, b.cancel = context.WithCancel(ctx)
	go b.connectLoop(ctx)
}

// Shutdown cleanly closes the bridge.
//
//	bridge.Shutdown()
func (b *Bridge) Shutdown() {
	if b.cancel != nil {
		b.cancel()
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.conn != nil {
		b.conn.Close()
		b.conn = nil
	}
	b.connected = false
}

// Connected reports whether the bridge has an active connection.
//
//	if bridge.Connected() {
//	    fmt.Println("online")
//	}
func (b *Bridge) Connected() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.connected
}

// Send sends a message to the Laravel backend.
//
//	err := bridge.Send(BridgeMessage{Type: "dashboard_overview"})
func (b *Bridge) Send(
	msg BridgeMessage,
) (
	_ error, // result
) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.conn == nil {
		return core.E("bridge.Send", "not connected", nil)
	}
	msg.Timestamp = time.Now()
	data := []byte(core.JSONMarshalString(msg))
	return b.conn.WriteMessage(websocket.TextMessage, data)
}

// connectLoop reconnects to Laravel with exponential backoff.
func (b *Bridge) connectLoop(ctx context.Context) {
	delay := b.cfg.ReconnectInterval
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		if err := b.dial(ctx); err != nil {
			core.Warn("ide bridge: connect failed", "err", err)
			select {
			case <-ctx.Done():
				return
			case <-time.After(delay):
			}
			delay = min(delay*2, b.cfg.MaxReconnectInterval)
			continue
		}

		// Reset backoff on successful connection
		delay = b.cfg.ReconnectInterval
		b.readLoop(ctx)
	}
}

func (b *Bridge) dial(
	ctx context.Context,
) (
	_ error, // result
) {
	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}

	var header http.Header
	if b.cfg.Token != "" {
		header = http.Header{}
		header.Set("Authorization", "Bearer "+b.cfg.Token)
	}

	conn, _, err := dialer.DialContext(ctx, b.cfg.LaravelWSURL, header)
	if err != nil {
		return err
	}

	b.mu.Lock()
	b.conn = conn
	b.connected = true
	b.mu.Unlock()

	core.Info("ide bridge: connected", "url", b.cfg.LaravelWSURL)
	return nil
}

func (b *Bridge) readLoop(ctx context.Context) {
	defer func() {
		b.mu.Lock()
		if b.conn != nil {
			b.conn.Close()
		}
		b.connected = false
		b.mu.Unlock()
	}()

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		_, data, err := b.conn.ReadMessage()
		if err != nil {
			core.Warn("ide bridge: read error", "err", err)
			return
		}

		var msg BridgeMessage
		if r := core.JSONUnmarshal(data, &msg); !r.OK {
			core.Warn("ide bridge: unmarshal error")
			continue
		}

		b.dispatch(msg)
		for _, observer := range b.snapshotObservers() {
			observer(msg)
		}
	}
}

func (b *Bridge) snapshotObservers() []func(BridgeMessage) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if len(b.observers) == 0 {
		return nil
	}
	observers := make([]func(BridgeMessage), len(b.observers))
	copy(observers, b.observers)
	return observers
}

// dispatch routes an incoming message to the appropriate ws.Hub channel.
func (b *Bridge) dispatch(msg BridgeMessage) {
	if b.hub == nil {
		return
	}

	wsMsg := ws.Message{
		Type: ws.TypeEvent,
		Data: msg.Data,
	}

	channel := msg.Channel
	if channel == "" {
		channel = "ide:" + msg.Type
	}

	if r := b.hub.SendToChannel(channel, wsMsg); !r.OK {
		core.Warn("ide bridge: dispatch failed", "channel", channel, "err", r.Error())
	}
}
