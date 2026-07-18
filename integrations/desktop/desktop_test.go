package sydesktop

import (
	"testing"

	sy "github.com/HazelnutParadise/syralit"
)

// The window itself can't open under `go test`; what we can verify headlessly
// is that options land on the opts struct the way Run will read them.
func TestOptionsApply(t *testing.T) {
	o := &opts{width: 1024, height: 768}
	for _, apply := range []Option{
		Config(sy.Config{Title: "cfg"}),
		WindowTitle("win"),
		WindowSize(640, 480),
		MinSize(320, 240),
		Frameless(),
		Icon([]byte{1, 2}),
	} {
		apply(o)
	}
	if o.cfg.Title != "cfg" || o.title != "win" {
		t.Errorf("titles not applied: %+v", o)
	}
	if o.width != 640 || o.height != 480 || o.minWidth != 320 || o.minHeight != 240 {
		t.Errorf("sizes not applied: %+v", o)
	}
	if !o.frameless || len(o.icon) != 2 {
		t.Errorf("frameless/icon not applied: %+v", o)
	}
}
