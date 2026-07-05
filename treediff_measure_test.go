package syralit

import (
	"encoding/json"
	"fmt"
	"testing"
)

// TestTreePayloadMeasurement quantifies the full-tree push cost for a large
// app — the data behind the "is UI tree diffing worth it" decision.
func TestTreePayloadMeasurement(t *testing.T) {
	rows := make([][]any, 1000)
	for i := range rows {
		rows[i] = []any{i, fmt.Sprintf("name-%d", i), float64(i) * 1.5, i%2 == 0,
			fmt.Sprintf("category-%d", i%7), fmt.Sprintf("2026-07-%02d", i%28+1)}
	}
	app := func() {
		Title("Big")
		TextInput("Filter", Key("f"))
		DataFrame([]string{"ID", "Name", "Score", "OK", "Cat", "Date"}, rows)
		LineChart(map[string][]float64{"S": make([]float64, 500)})
	}
	at := NewAppTest(app)
	tree := at.Run()
	full, err := json.Marshal(tree)
	if err != nil {
		t.Fatal(err)
	}

	// One keystroke in the filter — the entire tree is re-sent today.
	at.SetValue("f", "x")
	tree2 := at.Run()
	second, _ := json.Marshal(tree2)

	t.Logf("full tree payload: %d bytes (%.1f KB); after 1-char input: %d bytes resent",
		len(full), float64(len(full))/1024, len(second))
	if len(full) < 10_000 {
		t.Fatalf("measurement app unexpectedly small: %d", len(full))
	}
}
