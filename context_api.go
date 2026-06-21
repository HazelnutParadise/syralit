package syralit

import "net/http"

// RequestContext exposes read-only information about the browsing context's HTTP
// request — the equivalent of Streamlit's st.context. Retrieve it with Context().
//
// In production (sy.Run / a compiled app) the values reflect the browser's
// WebSocket upgrade request. Under `syralit dev` the supervisor proxies the
// socket, so Headers/Cookies/IP describe the local supervisor connection rather
// than the end user.
type RequestContext struct {
	Headers map[string]string // canonical header name -> first value
	Cookies map[string]string // cookie name -> value
	Host    string            // request host (e.g. "localhost:8600")
	IP      string            // remote address (host portion)
	Locale  string            // best-effort primary language from Accept-Language
}

// Context returns information about the current session's HTTP request.
func Context() RequestContext {
	return current().sess.reqCtx
}

// captureRequest snapshots the parts of an HTTP request exposed via Context.
func captureRequest(r *http.Request) RequestContext {
	rc := RequestContext{
		Headers: map[string]string{},
		Cookies: map[string]string{},
		Host:    r.Host,
		IP:      hostOnly(r.RemoteAddr),
	}
	for k, v := range r.Header {
		if len(v) > 0 {
			rc.Headers[k] = v[0]
		}
	}
	for _, ck := range r.Cookies() {
		rc.Cookies[ck.Name] = ck.Value
	}
	rc.Locale = primaryLocale(rc.Headers["Accept-Language"])
	return rc
}

// hostOnly strips the port from a "host:port" address.
func hostOnly(addr string) string {
	for i := len(addr) - 1; i >= 0; i-- {
		if addr[i] == ':' {
			return addr[:i]
		}
	}
	return addr
}

// primaryLocale extracts the first language tag from an Accept-Language header,
// e.g. "en-US,en;q=0.9" -> "en-US".
func primaryLocale(accept string) string {
	if accept == "" {
		return ""
	}
	end := len(accept)
	for i := 0; i < len(accept); i++ {
		if accept[i] == ',' || accept[i] == ';' || accept[i] == ' ' {
			end = i
			break
		}
	}
	return accept[:end]
}
