package syralit

import (
	"bufio"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestSSETransport(t *testing.T) {
	app := func() {
		v := TextInput("Name", Key("n"))
		Text("hello:" + v)
	}
	srv := httptest.NewServer((&server{cfg: Config{}, appFn: app}).handler())
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, _ := http.NewRequestWithContext(ctx, "GET", srv.URL+"/_syralit/sse", nil)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if ct := res.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/event-stream") {
		t.Fatalf("content-type: %s", ct)
	}
	reader := bufio.NewReader(res.Body)

	// readEvent reads one SSE event, returning its (event, data) fields.
	readEvent := func() (event, data string) {
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				return event, data
			}
			line = strings.TrimRight(line, "\r\n")
			if line == "" {
				if event != "" || data != "" {
					return event, data
				}
				continue
			}
			if strings.HasPrefix(line, "event: ") {
				event = line[len("event: "):]
			} else if strings.HasPrefix(line, "data: ") {
				data = line[len("data: "):]
			}
		}
	}

	ev, sid := readEvent()
	if ev != "session" || sid == "" {
		t.Fatalf("expected session event, got event=%q data=%q", ev, sid)
	}

	if _, patch := readEvent(); !strings.Contains(patch, "hello:") {
		t.Fatalf("initial patch missing greeting: %s", patch)
	}

	// Upstream half: POST a widget change correlated by session id.
	body, _ := json.Marshal(map[string]any{
		"session_id": sid, "type": "widget_change", "widget_id": "n", "value": "Bob",
	})
	pres, err := http.Post(srv.URL+"/_syralit/msg", "application/json", strings.NewReader(string(body)))
	if err != nil {
		t.Fatal(err)
	}
	pres.Body.Close()

	// The update is pushed back down the SSE stream.
	if _, patch := readEvent(); !strings.Contains(patch, "hello:Bob") {
		t.Fatalf("change not reflected over SSE: %s", patch)
	}
}
