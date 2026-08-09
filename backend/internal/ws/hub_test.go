package ws

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/aeroxe/docu-flow/backend/internal/pkg/logger"
)

func TestMain(m *testing.M) {
	_ = logger.Init("warn", false)
	os.Exit(m.Run())
}

// fakeConn implements the minimal Conn surface the hub uses (writePump).
type fakeConn struct {
	messages chan []byte
	closed   chan struct{}
}

func newFakeConn() *fakeConn {
	return &fakeConn{messages: make(chan []byte, 16), closed: make(chan struct{}, 1)}
}

func (f *fakeConn) WriteMessage(messageType int, data []byte) error {
	f.messages <- data
	return nil
}

func (f *fakeConn) Close() error {
	select {
	case f.closed <- struct{}{}:
	default:
	}
	return nil
}

func newClient(h *Hub, userID string, c *fakeConn) *Client {
	return &Client{hub: h, conn: c, userID: userID, send: make(chan []byte, 32)}
}

func drain(h *Hub) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for c := range h.clients {
		close(c.send)
	}
}

func TestHubRegisterAndCount(t *testing.T) {
	h := NewHub()
	c1, c2 := newFakeConn(), newFakeConn()
	cl1 := newClient(h, "u1", c1)
	cl2 := newClient(h, "u2", c2)
	h.Register(cl1)
	h.Register(cl2)
	if got := h.ClientCount(); got != 2 {
		t.Fatalf("ClientCount = %d, want 2", got)
	}
	h.Unregister(cl1)
	if got := h.ClientCount(); got != 1 {
		t.Fatalf("ClientCount after unregister = %d, want 1", got)
	}
	// Unregister is idempotent.
	h.Unregister(cl1)
	h.Unregister(cl2)
	if got := h.ClientCount(); got != 0 {
		t.Fatalf("ClientCount after draining = %d, want 0", got)
	}
}

func TestHubPublishToSpecificUsers(t *testing.T) {
	h := NewHub()
	c1, c2, c3 := newFakeConn(), newFakeConn(), newFakeConn()
	h.Register(newClient(h, "u1", c1))
	h.Register(newClient(h, "u2", c2))
	h.Register(newClient(h, "u1", c3)) // second connection for u1
	defer drain(h)

	h.PublishToUsers([]string{"u1"}, Event{Type: "document_ready", DocumentID: "d1"})

	assertReceived(t, c1, "document_ready")
	assertReceived(t, c3, "document_ready")
	if len(c2.messages) != 0 {
		t.Fatalf("u2 must not receive u1's event, got %d messages", len(c2.messages))
	}
}

func TestHubBroadcast(t *testing.T) {
	h := NewHub()
	c1, c2 := newFakeConn(), newFakeConn()
	h.Register(newClient(h, "u1", c1))
	h.Register(newClient(h, "u2", c2))
	defer drain(h)

	h.Broadcast(Event{Type: "system.notice"})
	assertReceived(t, c1, "system.notice")
	assertReceived(t, c2, "system.notice")
}

func TestHubSlowConsumerDropsWithoutBlocking(t *testing.T) {
	h := NewHub()
	fc := newFakeConn()
	client := &Client{hub: h, conn: fc, userID: "slow", send: make(chan []byte, 1)}
	h.Register(client)
	defer drain(h)

	// Fill the buffered channel so the next delivery would block.
	client.send <- []byte("fill")
	done := make(chan struct{})
	go func() {
		h.PublishToUsers([]string{"slow"}, Event{Type: "overflow"})
		close(done)
	}()
	select {
	case <-done:
		// Non-blocking drop as designed.
	case <-time.After(2 * time.Second):
		t.Fatal("PublishToUsers blocked on a slow consumer")
	}
}

func TestHubClientCountConcurrent(t *testing.T) {
	h := NewHub()
	for i := 0; i < 50; i++ {
		h.Register(newClient(h, "u", newFakeConn()))
	}
	defer drain(h)
	if got := h.ClientCount(); got != 50 {
		t.Fatalf("ClientCount = %d, want 50", got)
	}
}

func assertReceived(t *testing.T, fc *fakeConn, wantType string) {
	t.Helper()
	select {
	case raw := <-fc.messages:
		var evt Event
		if err := json.Unmarshal(raw, &evt); err != nil {
			t.Fatalf("bad event payload: %v", err)
		}
		if evt.Type != wantType {
			t.Fatalf("event type = %q, want %q", evt.Type, wantType)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out waiting for event %q", wantType)
	}
}
