package syralit

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
)

// TestPushEchoesClientAcks covers issue #13: every push carries the highest
// frame seq handled per browser tab, so the runtime can tell a tree that
// answers an older keystroke from one that answers the latest.
func TestPushEchoesClientAcks(t *testing.T) {
	app := func() {
		TextInput("Name", Key("name"))
		Fragment("frag", func() { TextInput("Inner", Key("inner")) })
	}
	srv := httptest.NewServer((&server{cfg: Config{}, appFn: app}).handler())
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	c, _, err := websocket.Dial(ctx, "ws"+strings.TrimPrefix(srv.URL, "http")+"/_syralit/ws", nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer c.CloseNow()

	read := func() map[string]any {
		t.Helper()
		_, data, err := c.Read(ctx)
		if err != nil {
			t.Fatalf("read: %v", err)
		}
		var m map[string]any
		if err := json.Unmarshal(data, &m); err != nil {
			t.Fatalf("decode: %v", err)
		}
		return m
	}
	send := func(frame map[string]any) {
		t.Helper()
		b, _ := json.Marshal(frame)
		if err := c.Write(ctx, websocket.MessageText, b); err != nil {
			t.Fatalf("write: %v", err)
		}
	}
	ackOf := func(m map[string]any, client string) any {
		acks, _ := m["acks"].(map[string]any)
		return acks[client]
	}

	if first := read(); first["acks"] != nil {
		t.Fatalf("initial render should carry no acks, got %v", first["acks"])
	}

	send(map[string]any{"type": "widget_change", "widget_id": "name", "value": "a", "client": "tab1", "seq": 3})
	if m := read(); ackOf(m, "tab1") != float64(3) {
		t.Fatalf("ui_patch acks = %v, want tab1:3", m["acks"])
	}

	// A lower seq (reordered or replayed frame) must not move the ack back.
	send(map[string]any{"type": "widget_change", "widget_id": "name", "value": "ab", "client": "tab1", "seq": 2})
	if m := read(); ackOf(m, "tab1") != float64(3) {
		t.Fatalf("ack moved back: %v", m["acks"])
	}

	// Fragment reruns carry acks too, keyed per tab.
	send(map[string]any{"type": "widget_change", "widget_id": "inner", "value": "x", "client": "tab2", "seq": 1})
	m := read()
	if m["type"] != "fragment_patch" {
		t.Fatalf("want fragment_patch, got %v", m["type"])
	}
	if ackOf(m, "tab2") != float64(1) || ackOf(m, "tab1") != float64(3) {
		t.Fatalf("fragment_patch acks = %v, want tab1:3 tab2:1", m["acks"])
	}

	// Frames without a client id (older runtimes) leave acks untouched.
	send(map[string]any{"type": "widget_change", "widget_id": "name", "value": "abc"})
	if m := read(); ackOf(m, "tab1") != float64(3) || len(m["acks"].(map[string]any)) != 2 {
		t.Fatalf("anonymous frame changed acks: %v", m["acks"])
	}
}

func TestRecordAckCapsClients(t *testing.T) {
	s := newSession(nil)
	for i := range maxAckClients + 5 {
		s.recordAck(fmt.Sprintf("c%d", i), 1)
	}
	if len(s.acks) != maxAckClients {
		t.Fatalf("acks size = %d, want %d", len(s.acks), maxAckClients)
	}
	last := fmt.Sprintf("c%d", maxAckClients+4)
	if s.acks[last] != 1 {
		t.Fatalf("newest client %s missing from acks", last)
	}
	s.recordAck(last, 0)
	s.recordAck("", 9)
	if s.acks[last] != 1 || len(s.acks) != maxAckClients {
		t.Fatalf("zero seq or empty client should be ignored: %v", s.acks)
	}
}
