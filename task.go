package syralit

import (
	"fmt"
	"sync"
)

// taskState holds a background job's status, guarded by its own mutex so the
// worker goroutine and the render loop can touch it safely.
type taskState struct {
	mu      sync.Mutex
	running bool
	done    bool
	result  any
	err     error
}

func (t *taskState) snapshot() (running, done bool, result any, err error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.running, t.done, t.result, t.err
}

// TaskHandle is the result of Task: a live view of a background job's state.
type TaskHandle[T any] struct {
	st *taskState
}

// Running reports whether the job is still in progress.
func (h TaskHandle[T]) Running() bool { r, _, _, _ := h.st.snapshot(); return r }

// Done reports whether the job has finished (successfully or with an error).
func (h TaskHandle[T]) Done() bool { _, d, _, _ := h.st.snapshot(); return d }

// Err returns the job's error (nil unless it panicked or you stored one).
func (h TaskHandle[T]) Err() error { _, _, _, e := h.st.snapshot(); return e }

// Result returns the job's result; the zero value of T until Done.
func (h TaskHandle[T]) Result() T {
	_, _, res, _ := h.st.snapshot()
	v, _ := res.(T)
	return v
}

// Task runs fn in the background (a goroutine) and returns a handle to its
// status. The job starts on first call for a given key in a session and runs
// exactly once; later reruns observe its progress via the returned handle. When
// the job finishes, the server pushes a rerun so the UI updates on its own —
// the page stays responsive while the work runs, which a Streamlit rerun (which
// blocks) cannot do.
//
//	job := sy.Task("report", func() Report { return buildReport() })
//	if job.Running() {
//	    sy.Spinner("Crunching…")
//	} else {
//	    render(job.Result())
//	}
func Task[T any](key string, fn func() T) TaskHandle[T] {
	rc := current()
	sess := rc.sess
	id := "task:" + key

	sess.mu.Lock()
	raw, ok := sess.store[id]
	if ok {
		sess.mu.Unlock()
		return TaskHandle[T]{st: raw.(*taskState)}
	}
	st := &taskState{running: true}
	sess.store[id] = st
	sess.mu.Unlock()

	go func() {
		defer func() {
			if r := recover(); r != nil {
				st.mu.Lock()
				st.err = fmt.Errorf("task panic: %v", r)
				st.running = false
				st.done = true
				st.mu.Unlock()
				sess.requestRerun()
			}
		}()
		res := fn()
		st.mu.Lock()
		st.result = res
		st.running = false
		st.done = true
		st.mu.Unlock()
		sess.requestRerun()
	}()

	return TaskHandle[T]{st: st}
}
