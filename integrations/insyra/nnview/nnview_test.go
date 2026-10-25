package syinn

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"strings"
	"testing"
	"time"

	sy "github.com/HazelnutParadise/syralit"

	"github.com/HazelnutParadise/insyra/nn"
)

func toyProblem(t *testing.T) (*nn.Sequential, *nn.Tensor, *nn.Tensor) {
	t.Helper()
	model, err := nn.NewSequential(nn.NewTape(7), nn.Dense(2, 4), nn.ReLU(), nn.Dense(4, 1))
	if err != nil {
		t.Fatal(err)
	}
	const rows = 16
	xd, yd := make([]float32, 0, rows*2), make([]float32, 0, rows)
	for i := 0; i < rows; i++ {
		a, b := float32(i)/rows, float32(i%4)/4
		xd = append(xd, a, b)
		yd = append(yd, a+b)
	}
	x, err := nn.NewTensor([]int{rows, 2}, xd)
	if err != nil {
		t.Fatal(err)
	}
	y, err := nn.NewTensor([]int{rows, 1}, yd)
	if err != nil {
		t.Fatal(err)
	}
	return model, x, y
}

func cfg(epochs int) nn.FitConfig {
	return nn.FitConfig{Epochs: epochs, BatchSize: 8, Seed: 1, Loss: nn.MSE{}, Optimizer: nn.Adam{Rate: 0.01}, Quiet: true}
}

func chartJSON(t *testing.T, tree *sy.Node) string {
	t.Helper()
	nodes := tree.Find("line_chart")
	if len(nodes) != 1 {
		t.Fatalf("want 1 line_chart, got %d", len(nodes))
	}
	b, err := json.Marshal(nodes[0].Props)
	if err != nil {
		t.Fatalf("chart props must be JSON-safe: %v", err)
	}
	return string(b)
}

func TestLossChartTrainAndVal(t *testing.T) {
	res := &nn.FitResult{TrainLosses: []float64{3, 2, 1}, ValLosses: []float64{3.5, 2.5, 1.5}}
	js := chartJSON(t, sy.RenderOnce(func() { LossChart(res) }))
	for _, want := range []string{`"train"`, `"val"`, `"3"`} {
		if !strings.Contains(js, want) {
			t.Errorf("chart props missing %s: %s", want, js)
		}
	}
}

func TestLossChartTrainOnly(t *testing.T) {
	res := &nn.FitResult{TrainLosses: []float64{3, 2}}
	if js := chartJSON(t, sy.RenderOnce(func() { LossChart(res) })); strings.Contains(js, `"val"`) {
		t.Errorf("no validation series expected: %s", js)
	}
}

func TestLossChartEmpty(t *testing.T) {
	for _, res := range []*nn.FitResult{nil, {}} {
		tree := sy.RenderOnce(func() { LossChart(res) })
		if len(tree.Find("line_chart")) != 0 || len(tree.Find("status")) != 1 {
			t.Errorf("empty result should show only a notice")
		}
	}
}

func TestLossChartStopsAtNaN(t *testing.T) {
	res := &nn.FitResult{TrainLosses: []float64{3, 2, math.NaN(), 1}, ValLosses: []float64{3, math.Inf(1)}}
	tree := sy.RenderOnce(func() { LossChart(res) })
	chartJSON(t, tree) // must marshal
	if n := len(tree.Find("status")); n != 2 {
		t.Errorf("want a warning for train and for val, got %d", n)
	}
}

func waitDone(t *testing.T, h *FitHandle) {
	t.Helper()
	deadline := time.Now().Add(60 * time.Second)
	for !h.Done() {
		if time.Now().After(deadline) {
			t.Fatal("training did not finish")
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func TestFitLiveTrainsInBackground(t *testing.T) {
	model, x, y := toyProblem(t)
	var h *FitHandle
	var seen int
	c := cfg(3)
	c.Progress = func(nn.FitEpoch) { seen++ }
	tree := sy.RenderOnce(func() { h = FitLive("toy", model, x, y, c) })
	if len(tree.Find("fragment")) != 1 || len(tree.Find("progress")) != 1 {
		t.Fatalf("live view should render a fragment with a progress bar")
	}
	waitDone(t, h)
	if err := h.Err(); err != nil {
		t.Fatal(err)
	}
	res := h.Result()
	if res == nil || len(res.TrainLosses) != 3 {
		t.Fatalf("want 3 epochs, got %+v", res)
	}
	if seen != 3 {
		t.Errorf("caller's Progress should still see every epoch, saw %d", seen)
	}
	chartJSON(t, sy.RenderOnce(func() { renderFit(h.st, "toy") }))
	if h.Running() {
		t.Error("Running should be false after Done")
	}
}

func TestFitLiveCancel(t *testing.T) {
	model, x, y := toyProblem(t)
	var h *FitHandle
	sy.RenderOnce(func() { h = FitLive("cancel", model, x, y, cfg(100000)) })
	h.Cancel()
	waitDone(t, h)
	if !errors.Is(h.Err(), context.Canceled) {
		t.Fatalf("want context.Canceled, got %v", h.Err())
	}
	tree := sy.RenderOnce(func() { renderFit(h.st, "cancel") })
	if len(tree.Find("status")) == 0 {
		t.Error("a stopped run should say so")
	}
}

func TestFitLiveProgressPanicEndsTheRun(t *testing.T) {
	model, x, y := toyProblem(t)
	c := cfg(3)
	c.Progress = func(nn.FitEpoch) { panic("boom") }
	var h *FitHandle
	sy.RenderOnce(func() { h = FitLive("panic", model, x, y, c) })
	waitDone(t, h)
	if h.Err() == nil || !strings.Contains(h.Err().Error(), "boom") {
		t.Fatalf("panic should surface as the run's error, got %v", h.Err())
	}
}
