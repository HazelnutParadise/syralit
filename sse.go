package syralit

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"github.com/coder/websocket"
)

// uiSink is where a render is delivered. It abstracts over the two transports:
// a WebSocket frame, or a Server-Sent Events message. This lets pushUI work the
// same whether the browser uses WS or the HTTP/SSE fallback.
type uiSink interface {
	send(data []byte) error
}

// wsSink writes UI patches as WebSocket text frames.
type wsSink struct {
	c   *websocket.Conn
	ctx context.Context
}

func (s wsSink) send(data []byte) error {
	return s.c.Write(s.ctx, websocket.MessageText, data)
}

// sseSink writes UI patches as SSE events on a long-lived HTTP response. Writes
// are serialized because the streaming response is shared by the SSE pump
// goroutine and inline POST handling.
type sseSink struct {
	w  http.ResponseWriter
	f  http.Flusher
	mu *sync.Mutex
}

func (s sseSink) send(data []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, err := fmt.Fprintf(s.w, "data: %s\n\n", data); err != nil {
		return err
	}
	s.f.Flush()
	return nil
}

// sseRegistry maps a session id to its session so POST /_syralit/msg (the
// upstream half of the SSE transport) can find the session opened by GET
// /_syralit/sse (the downstream half).
var (
	sseMu       sync.Mutex
	sseSessions = map[string]*session{}
)

func sseRegister(sess *session) { sseMu.Lock(); sseSessions[sess.id] = sess; sseMu.Unlock() }
func sseDeregister(id string)   { sseMu.Lock(); delete(sseSessions, id); sseMu.Unlock() }
func sseLookup(id string) (*session, bool) {
	sseMu.Lock()
	defer sseMu.Unlock()
	s, ok := sseSessions[id]
	return s, ok
}

// handleSSE is the downstream half of the SSE transport: a normal HTTP GET whose
// response streams text/event-stream. The browser falls back to it (via
// EventSource) when a WebSocket can't be established — e.g. behind a proxy that
// blocks WS upgrades. The upstream half is POST /_syralit/msg.
func (s *server) handleSSE(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") // disable proxy buffering

	sess := s.newSession()
	qp := make(map[string]string)
	for k, v := range r.URL.Query() {
		if len(v) > 0 {
			qp[k] = v[0]
		}
	}
	sess.mu.Lock()
	sess.currentPage = takeInitialPage(qp)
	sess.queryParams = qp
	sess.reqCtx = captureRequest(r)
	sess.mu.Unlock()
	resolveSessionUser(sess)

	sink := sseSink{w: w, f: flusher, mu: &sync.Mutex{}}
	sess.sink = sink
	sseRegister(sess)
	registerSession(sess)
	defer func() {
		sseDeregister(sess.id)
		deregisterSession(sess)
	}()

	// Hand the client its session id (for POST correlation), then first render.
	sink.mu.Lock()
	fmt.Fprintf(w, "event: session\ndata: %s\n\n", sess.id)
	flusher.Flush()
	sink.mu.Unlock()
	if err := pushUI(sink, sess); err != nil {
		return
	}

	// Pump server-initiated reruns (background Tasks, Shared updates) until the
	// client disconnects.
	for {
		select {
		case <-r.Context().Done():
			return
		case <-sess.wake:
			if err := pushUI(sink, sess); err != nil {
				return
			}
		}
	}
}

// handleMsg is the upstream half of the SSE transport: the browser POSTs client
// frames here (the request carries session_id); any resulting UI is pushed back
// out over that session's SSE stream.
func (s *server) handleMsg(w http.ResponseWriter, r *http.Request) {
	var msg clientMsg
	if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	sess, ok := sseLookup(msg.SessionID)
	if !ok {
		http.Error(w, "unknown session", http.StatusNotFound)
		return
	}
	if sess.sink != nil {
		s.handleClientMsg(sess.sink, sess, msg)
	}
	w.WriteHeader(http.StatusNoContent)
}
