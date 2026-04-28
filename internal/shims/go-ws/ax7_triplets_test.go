package ws

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"

	core "dappco.re/go"
	"github.com/gorilla/websocket"
)

type T = core.T

var (
	AssertEqual    = core.AssertEqual
	AssertLen      = core.AssertLen
	AssertNil      = core.AssertNil
	AssertNoError  = core.AssertNoError
	AssertNotEqual = core.AssertNotEqual
	AssertNotNil   = core.AssertNotNil
	AssertPanics   = core.AssertPanics
)

func TestAX7_NewHub_Good(t *T) {
	h := NewHub()
	AssertNotNil(t, h)
	AssertEqual(t, 0, h.Stats().Clients)
}

func TestAX7_NewHub_Bad(t *T) {
	h := NewHub()
	AssertNotNil(t, h)
	AssertEqual(t, Stats{}, h.Stats())
}

func TestAX7_NewHub_Ugly(t *T) {
	h := NewHub()
	AssertLen(t, h.channels, 0)
	AssertLen(t, h.clients, 0)
}

func TestAX7_Hub_Run_Good(t *T) {
	h := NewHub()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		h.Run(ctx)
		close(done)
	}()
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Run did not return after context cancellation")
	}
}

func TestAX7_Hub_Run_Bad(t *T) {
	h := NewHub()
	AssertNotNil(t, h)
	AssertPanics(t, func() { h.Run(nil) })
}

func TestAX7_Hub_Run_Ugly(t *T) {
	h := NewHub()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	h.Run(ctx)
}

func TestAX7_Hub_Handler_Good(t *T) {
	h := NewHub()
	srv := httptest.NewServer(h.Handler())
	defer srv.Close()

	conn, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(srv.URL, "http"), nil)
	AssertNoError(t, err)
	AssertEqual(t, 1, h.Stats().Clients)
	AssertNoError(t, conn.Close())
}

func TestAX7_Hub_Handler_Bad(t *T) {
	h := NewHub()
	req := httptest.NewRequest(http.MethodGet, "/ws", nil)
	rr := httptest.NewRecorder()
	h.Handler()(rr, req)
	AssertNotEqual(t, http.StatusSwitchingProtocols, rr.Code)
}

func TestAX7_Hub_Handler_Ugly(t *T) {
	h := NewHub()
	srv := httptest.NewServer(h.Handler())
	defer srv.Close()
	conn, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(srv.URL, "http"), nil)
	AssertNoError(t, err)
	AssertNoError(t, conn.Close())
	for i := 0; i < 20 && h.Stats().Clients != 0; i++ {
		time.Sleep(10 * time.Millisecond)
	}
	AssertEqual(t, 0, h.Stats().Clients)
}

func TestAX7_Hub_SendToChannel_Good(t *T) {
	h := NewHub()
	err := h.SendToChannel("events", Message{Type: TypeEvent, Data: "ok"})
	AssertNoError(t, err)
	AssertLen(t, h.channels["events"], 1)
}

func TestAX7_Hub_SendToChannel_Bad(t *T) {
	var h *Hub
	AssertNil(t, h)
	AssertPanics(t, func() { _ = h.SendToChannel("events", Message{}) })
}

func TestAX7_Hub_SendToChannel_Ugly(t *T) {
	h := &Hub{}
	err := h.SendToChannel("", Message{})
	AssertNoError(t, err)
	AssertLen(t, h.channels[""], 1)
}

func TestAX7_Hub_SendProcessOutput_Good(t *T) {
	h := NewHub()
	err := h.SendProcessOutput("proc-1", "line")
	AssertNoError(t, err)
	AssertLen(t, h.channels["process.output"], 1)
}

func TestAX7_Hub_SendProcessOutput_Bad(t *T) {
	var h *Hub
	AssertNil(t, h)
	AssertPanics(t, func() { _ = h.SendProcessOutput("proc-1", "line") })
}

func TestAX7_Hub_SendProcessOutput_Ugly(t *T) {
	h := NewHub()
	err := h.SendProcessOutput("", "")
	AssertNoError(t, err)
	AssertLen(t, h.channels["process.output"], 1)
}

func TestAX7_Hub_SendProcessStatus_Good(t *T) {
	h := NewHub()
	err := h.SendProcessStatus("proc-1", "exited", 0)
	AssertNoError(t, err)
	AssertLen(t, h.channels["process.status"], 1)
}

func TestAX7_Hub_SendProcessStatus_Bad(t *T) {
	var h *Hub
	AssertNil(t, h)
	AssertPanics(t, func() { _ = h.SendProcessStatus("proc-1", "exited", 0) })
}

func TestAX7_Hub_SendProcessStatus_Ugly(t *T) {
	h := NewHub()
	err := h.SendProcessStatus("", "", -1)
	AssertNoError(t, err)
	AssertLen(t, h.channels["process.status"], 1)
}

func TestAX7_Hub_Stats_Good(t *T) {
	h := NewHub()
	err := h.SendToChannel("events", Message{})
	AssertNoError(t, err)
	stats := h.Stats()
	AssertEqual(t, 0, stats.Clients)
	AssertEqual(t, 1, stats.Channels)
}

func TestAX7_Hub_Stats_Bad(t *T) {
	var h *Hub
	AssertNil(t, h)
	AssertPanics(t, func() { _ = h.Stats() })
}

func TestAX7_Hub_Stats_Ugly(t *T) {
	h := &Hub{}
	stats := h.Stats()
	AssertEqual(t, 0, stats.Clients)
	AssertEqual(t, 0, stats.Channels)
}
