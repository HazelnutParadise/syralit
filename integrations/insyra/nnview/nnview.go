// Package syinn shows Insyra neural-network training (insyra/nn) in a Syralit
// app: a loss-curve chart for a finished fit, and a live view that trains in
// the background while the page shows a progress bar and the curve growing.
//
// It lives in its own package because insyra/nn pulls in the GPU-acceleration
// libraries; apps that only use the main adapter (package syinsyra) do not
// link them.
package syinn

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strconv"
	"sync"
	"time"

	sy "github.com/HazelnutParadise/syralit"

	"github.com/HazelnutParadise/insyra/nn"
)

// liveRefresh is how often the live view re-reads training progress.
const liveRefresh = 500 * time.Millisecond

// LossChart plots the training loss (and the validation loss, when the fit had
// ValX/ValY) per epoch of a finished or partial fit. Epochs are the x axis.
// A loss that is NaN or infinite ends the curve there with a warning, because
// such a value cannot be drawn.
//
//	res, err := model.Fit(x, y, cfg)
//	syinn.LossChart(res)
func LossChart(res *nn.FitResult, opts ...sy.Option) {
	if res == nil || len(res.TrainLosses) == 0 {
		sy.Info("No epochs completed yet.")
		return
	}
	train, trainBad := finitePrefix(res.TrainLosses)
	series := map[string][]float64{"train": train}
	n := len(train)
	if len(res.ValLosses) > 0 {
		val, valBad := finitePrefix(res.ValLosses)
		series["val"] = val
		if len(val) > n {
			n = len(val)
		}
		if valBad >= 0 {
			sy.Warning(fmt.Sprintf("Validation loss became non-finite at epoch %d; the curve stops there.", valBad+1))
		}
	}
	if trainBad >= 0 {
		sy.Warning(fmt.Sprintf("Training loss became non-finite at epoch %d; the curve stops there.", trainBad+1))
	}
	labels := make([]string, n)
	for i := range labels {
		labels[i] = strconv.Itoa(i + 1)
	}
	sy.LineChart(series, append([]sy.Option{sy.XLabels(labels), sy.ChartTitle("Loss per epoch")}, opts...)...)
}

// finitePrefix returns the losses before the first NaN/Inf and the index of
// that value (-1 when every loss is finite).
func finitePrefix(v []float64) ([]float64, int) {
	for i, f := range v {
		if math.IsNaN(f) || math.IsInf(f, 0) {
			return v[:i], i
		}
	}
	return v, -1
}

// FitHandle is a live view of a background training run started by FitLive.
type FitHandle struct {
	st *fitState
}

type fitState struct {
	mu     sync.Mutex
	epochs []nn.FitEpoch
	total  int
	done   bool
	res    *nn.FitResult
	err    error
	ctx    context.Context
	cancel context.CancelFunc
}

func newFitState(total int) *fitState {
	ctx, cancel := context.WithCancel(context.Background())
	return &fitState{total: total, ctx: ctx, cancel: cancel}
}

type fitSnapshot struct {
	epochs []nn.FitEpoch
	total  int
	done   bool
	res    *nn.FitResult
	err    error
}

func (s *fitState) snapshot() fitSnapshot {
	s.mu.Lock()
	defer s.mu.Unlock()
	return fitSnapshot{epochs: append([]nn.FitEpoch(nil), s.epochs...), total: s.total, done: s.done, res: s.res, err: s.err}
}

// Running reports whether training is still in progress.
func (h *FitHandle) Running() bool { return !h.st.snapshot().done }

// Done reports whether training has finished, failed or been cancelled.
func (h *FitHandle) Done() bool { return h.st.snapshot().done }

// Result returns the fit result; nil until Done. After Cancel it holds the
// epochs that completed.
func (h *FitHandle) Result() *nn.FitResult { return h.st.snapshot().res }

// Err returns the training error, or nil. A cancelled run reports
// context.Canceled.
func (h *FitHandle) Err() error { return h.st.snapshot().err }

// Cancel stops training after the current batch. It is safe to call at any time.
func (h *FitHandle) Cancel() { h.st.cancel() }

// FitLive trains model in the background and shows live progress: a progress
// bar, the latest losses, a Stop button, and the loss curve growing each
// epoch. It returns at once with a handle; the page stays responsive while the
// model trains. The same key must be used on every rerun, and training starts
// only once per key and session.
//
// cfg.Progress, if set, still receives every epoch. Do not use model (predict,
// save) until Done reports true. Training continues if the user closes the
// page and is cancelled only by Cancel or the Stop button.
//
//	h := syinn.FitLive("mnist", model, x, y, nn.FitConfig{Epochs: 20, BatchSize: 64, Loss: nn.CrossEntropy{}, Optimizer: nn.Adam{Rate: 1e-3}})
//	if h.Done() && h.Err() == nil {
//	    sy.Success("trained")
//	}
func FitLive(key string, model *nn.Sequential, x, y *nn.Tensor, cfg nn.FitConfig) *FitHandle {
	store := sy.Session()
	storeKey := "syinn:fit:" + key
	var st *fitState
	if v, ok := store.Get(storeKey); ok {
		st = v.(*fitState)
	} else {
		st = newFitState(cfg.Epochs)
		store.Set(storeKey, st)
	}
	// Task starts the training once per key and asks for a full rerun when it ends.
	sy.Task("syinn:fit:"+key, func() struct{} {
		runFit(st, model, x, y, cfg)
		return struct{}{}
	})
	snap := st.snapshot()
	opts := []sy.Option{}
	if !snap.done {
		opts = append(opts, sy.RunEvery(liveRefresh))
	}
	sy.Fragment("syinn-fit-"+key, func() { renderFit(st, key) }, opts...)
	return &FitHandle{st: st}
}

func runFit(st *fitState, model *nn.Sequential, x, y *nn.Tensor, cfg nn.FitConfig) {
	userProgress := cfg.Progress
	cfg.Progress = func(e nn.FitEpoch) {
		st.mu.Lock()
		st.epochs = append(st.epochs, e)
		st.mu.Unlock()
		if userProgress != nil {
			userProgress(e)
		}
	}
	var res *nn.FitResult
	var err error
	// A panic (for example in the caller's Progress callback) must still end
	// the run, or Done would stay false forever.
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("training panic: %v", r)
		}
		st.cancel()
		st.mu.Lock()
		st.res, st.err, st.done = res, err, true
		st.mu.Unlock()
	}()
	res, err = model.FitContext(st.ctx, x, y, cfg)
}

// renderFit draws the live view from the current snapshot.
func renderFit(st *fitState, key string) {
	snap := st.snapshot()
	last := nn.FitEpoch{}
	if n := len(snap.epochs); n > 0 {
		last = snap.epochs[n-1]
	}
	total := snap.total
	if total == 0 {
		total = last.Epochs
	}
	frac := 0.0
	if total > 0 {
		frac = float64(last.Epoch) / float64(total)
	}
	text := fmt.Sprintf("Epoch %d/%d", last.Epoch, total)
	if last.Epoch > 0 {
		text += fmt.Sprintf(" · loss %.4g", last.TrainLoss)
		if last.HasValLoss {
			text += fmt.Sprintf(" · val %.4g", last.ValLoss)
		}
	}
	switch {
	case snap.done && snap.err != nil && !errors.Is(snap.err, context.Canceled):
		sy.Exception(snap.err)
	case snap.done && errors.Is(snap.err, context.Canceled):
		sy.Warning(fmt.Sprintf("Training stopped after %d epochs.", last.Epoch))
	case snap.done:
		sy.Progress(1, text+" · done")
	default:
		sy.Progress(frac, text)
		if sy.Button("Stop training", sy.Key("syinn-stop-"+key)) {
			st.cancel()
		}
	}
	res := snap.res
	if res == nil {
		res = &nn.FitResult{}
		for _, e := range snap.epochs {
			res.TrainLosses = append(res.TrainLosses, e.TrainLoss)
			if e.HasValLoss {
				res.ValLosses = append(res.ValLosses, e.ValLoss)
			}
		}
	}
	LossChart(res)
}
