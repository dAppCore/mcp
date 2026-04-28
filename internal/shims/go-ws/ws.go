package ws

import (
	"context"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

const TypeEvent = "event"

type Message struct {
	Type string `json:"type"`
	Data any    `json:"data,omitempty"`
}

type Stats struct {
	Clients  int
	Channels int
}

type Hub struct {
	mu       sync.RWMutex
	clients  map[*websocket.Conn]struct{}
	channels map[string][]Message
}

func NewHub() *Hub {
	return &Hub{
		clients:  make(map[*websocket.Conn]struct{}),
		channels: make(map[string][]Message),
	}
}

func (h *Hub) Run(ctx context.Context) {
	<-ctx.Done()
}

func (h *Hub) Handler() http.HandlerFunc {
	upgrader := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		h.mu.Lock()
		h.clients[conn] = struct{}{}
		h.mu.Unlock()
		defer func() {
			h.mu.Lock()
			delete(h.clients, conn)
			h.mu.Unlock()
			if err := conn.Close(); err != nil {
				return
			}
		}()
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}
}

func (h *Hub) SendToChannel(channel string, msg Message) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.channels == nil {
		h.channels = make(map[string][]Message)
	}
	h.channels[channel] = append(h.channels[channel], msg)
	for conn := range h.clients {
		if err := conn.WriteJSON(msg); err != nil {
			return err
		}
	}
	return nil
}

func (h *Hub) Stats() Stats {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return Stats{Clients: len(h.clients), Channels: len(h.channels)}
}

func (h *Hub) SendProcessOutput(processID, line string) error {
	return h.SendToChannel("process.output", Message{
		Type: TypeEvent,
		Data: map[string]any{"id": processID, "line": line},
	})
}

func (h *Hub) SendProcessStatus(processID, status string, exitCode int) error {
	return h.SendToChannel("process.status", Message{
		Type: TypeEvent,
		Data: map[string]any{"id": processID, "status": status, "exitCode": exitCode},
	})
}
