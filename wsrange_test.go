package syralit

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
)

// TestRangeSelectionOverWS reproduces the browser wire path for a chart range
// selection: JSON over a real WebSocket, not AppTest's in-process map.
func TestRangeSelectionOverWS(t *testing.T) {
	app := func() {
		sel := LineChart(map[string][]float64{"S": {1, 5, 3, 8, 2, 7}},
			XLabels([]string{"a", "b", "c", "d", "e", "f"}),
			RangeSelectable(), Key("range_chart"))
		if sel != nil && sel.Range {
			Textf("picked:%s..%s", sel.X, sel.EndX)
		}
	}
	srv := httptest.NewServer((&server{cfg: Config{}, appFn: app}).handler())
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/_syralit/ws"
	c, _, err := websocket.Dial(ctx, wsURL, nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer c.CloseNow()

	_ = readPatch(t, ctx, c) // initial render

	// Exactly what the browser sends on drag end.
	payload := `{"type":"widget_change","widget_id":"range_chart","value":{"range":true,"index":1,"x":"b","end_index":4,"end_x":"e","series":"","value":0},"is_button":false}`
	if err := c.Write(ctx, websocket.MessageText, []byte(payload)); err != nil {
		t.Fatal(err)
	}
	nodes := readPatch(t, ctx, c)
	b, _ := json.Marshal(nodes)
	if !strings.Contains(string(b), "picked:b..e") {
		t.Fatalf("rerun missing picked text:\n%s", b)
	}
}
