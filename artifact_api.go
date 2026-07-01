package syralit

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"sync"
)

type artifactEndpoint struct {
	path       string
	store      *ArtifactStore
	auth       AgentAuthenticator
	sameOrigin bool
}

type artifactAPI struct {
	auth       AgentAuthenticator
	stores     map[string]*ArtifactStore
	sameOrigin bool
}

// ArtifactPlacement identifies one observed ArtifactCanvas in the rendered app.
// Page is empty for a single-page app.
type ArtifactPlacement struct {
	Page     string `json:"page,omitempty"`
	URL      string `json:"url,omitempty"`
	CanvasID string `json:"canvas_id"`
	Selector string `json:"selector"`
}

type observedArtifactPlacement struct {
	ArtifactPlacement
	artifact string
}

var (
	artifactEndpointMu sync.RWMutex
	artifactEndpoints  = map[string]artifactEndpoint{}
	artifactAPIs       = map[string]*artifactAPI{}

	artifactPlacementMu       sync.RWMutex
	artifactSessionPlacements = map[string]map[string][]observedArtifactPlacement{}
)

// HandleArtifactEndpoint registers one opt-in endpoint on the Syralit app
// server. POST replaces the store and GET returns its current spec and preview
// metadata.
func HandleArtifactEndpoint(path string, store *ArtifactStore, auth AgentAuthenticator) {
	validateArtifactRoute(path, auth)
	if store == nil {
		panic("syralit: artifact endpoint requires a store")
	}
	artifactEndpointMu.Lock()
	if _, exists := artifactEndpoints[path]; exists {
		artifactEndpointMu.Unlock()
		panic(fmt.Sprintf("syralit: artifact endpoint %q is already registered", path))
	}
	if _, exists := artifactAPIs[path]; exists {
		artifactEndpointMu.Unlock()
		panic(fmt.Sprintf("syralit: artifact route %q is already registered as a unified API", path))
	}
	artifactEndpoints[path] = artifactEndpoint{
		path: path, store: store, auth: auth, sameOrigin: true,
	}
	artifactEndpointMu.Unlock()
}

// HandleArtifactAPI registers one discovery and update endpoint for several
// stores on the Syralit app server. GET discovers stores; POST selects a store
// with the request's "artifact" field.
func HandleArtifactAPI(path string, auth AgentAuthenticator, stores ...*ArtifactStore) {
	validateArtifactRoute(path, auth)
	api := newArtifactAPI(auth, true, stores...)
	artifactEndpointMu.Lock()
	if _, exists := artifactAPIs[path]; exists {
		artifactEndpointMu.Unlock()
		panic(fmt.Sprintf("syralit: artifact API %q is already registered", path))
	}
	if _, exists := artifactEndpoints[path]; exists {
		artifactEndpointMu.Unlock()
		panic(fmt.Sprintf("syralit: artifact route %q is already registered as a single-store endpoint", path))
	}
	artifactAPIs[path] = api
	artifactEndpointMu.Unlock()
}

// ArtifactHandler returns a mountable handler for one store. It can be served
// from another mux or port while the ArtifactStore still broadcasts updates to
// the Syralit app.
func ArtifactHandler(store *ArtifactStore, auth AgentAuthenticator) http.Handler {
	if store == nil {
		panic("syralit: artifact handler requires a store")
	}
	if auth == nil {
		panic("syralit: artifact handler requires an authenticator")
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handleArtifactEndpoint(w, r, artifactEndpoint{store: store, auth: auth})
	})
}

// ArtifactAPIHandler returns a mountable unified discovery/update handler. It
// can be served from another mux or port.
func ArtifactAPIHandler(auth AgentAuthenticator, stores ...*ArtifactStore) http.Handler {
	return newArtifactAPI(auth, false, stores...)
}

func newArtifactAPI(auth AgentAuthenticator, sameOrigin bool, stores ...*ArtifactStore) *artifactAPI {
	if auth == nil {
		panic("syralit: artifact API requires an authenticator")
	}
	api := &artifactAPI{
		auth:       auth,
		stores:     make(map[string]*ArtifactStore, len(stores)),
		sameOrigin: sameOrigin,
	}
	for _, store := range stores {
		if store == nil {
			panic("syralit: artifact API requires non-nil stores")
		}
		name := strings.TrimSpace(store.Name())
		if name == "" {
			panic("syralit: artifact API store name must not be empty")
		}
		if _, exists := api.stores[name]; exists {
			panic(fmt.Sprintf("syralit: duplicate artifact store name %q", name))
		}
		api.stores[name] = store
	}
	return api
}

func validateArtifactRoute(path string, auth AgentAuthenticator) {
	if !strings.HasPrefix(path, "/") {
		panic("syralit: artifact endpoint path must start with /")
	}
	if auth == nil {
		panic("syralit: artifact endpoint requires an authenticator")
	}
}

func registerArtifactEndpoints(mux *http.ServeMux) {
	artifactEndpointMu.RLock()
	defer artifactEndpointMu.RUnlock()
	for path, ep := range artifactEndpoints {
		ep := ep
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			handleArtifactEndpoint(w, r, ep)
		})
		mux.Handle("GET "+path, handler)
		mux.Handle("POST "+path, handler)
	}
	for path, api := range artifactAPIs {
		mux.Handle("GET "+path, api)
		mux.Handle("POST "+path, api)
	}
}

func (api *artifactAPI) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !authenticateArtifactRequest(w, r, api.auth) {
		return
	}
	switch r.Method {
	case http.MethodGet:
		api.handleGet(w, r)
	case http.MethodPost:
		api.handlePost(w, r)
	default:
		w.Header().Set("Allow", "GET, POST")
		writeArtifactError(w, http.StatusMethodNotAllowed, "method_not_allowed", "use GET or POST")
	}
}

func (api *artifactAPI) handleGet(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimSpace(r.URL.Query().Get("artifact"))
	if name != "" {
		store := api.stores[name]
		if store == nil {
			writeArtifactError(w, http.StatusNotFound, "artifact_not_found", "unknown artifact "+name)
			return
		}
		spec, revision := store.snapshot()
		response := map[string]any{
			"artifact":   name,
			"revision":   revision,
			"spec":       spec,
			"placements": artifactPlacements(name, r, api.sameOrigin),
		}
		if api.sameOrigin {
			response["app_url"] = requestOrigin(r)
		}
		writeArtifactJSON(w, http.StatusOK, response)
		return
	}

	names := make([]string, 0, len(api.stores))
	for name := range api.stores {
		names = append(names, name)
	}
	sort.Strings(names)
	artifacts := make([]map[string]any, 0, len(names))
	for _, name := range names {
		store := api.stores[name]
		artifacts = append(artifacts, map[string]any{
			"id":         name,
			"revision":   store.Revision(),
			"placements": artifactPlacements(name, r, api.sameOrigin),
		})
	}
	response := map[string]any{"artifacts": artifacts}
	if api.sameOrigin {
		response["app_url"] = requestOrigin(r)
	}
	writeArtifactJSON(w, http.StatusOK, response)
}

func (api *artifactAPI) handlePost(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Artifact         string       `json:"artifact"`
		ExpectedRevision *uint64      `json:"expected_revision"`
		Spec             ArtifactSpec `json:"spec"`
	}
	if !decodeArtifactRequest(w, r, &req) {
		return
	}
	if req.ExpectedRevision == nil {
		writeArtifactError(w, http.StatusBadRequest, "bad_request", "expected_revision is required")
		return
	}
	store := api.stores[req.Artifact]
	if store == nil {
		writeArtifactError(w, http.StatusNotFound, "artifact_not_found", "unknown artifact "+req.Artifact)
		return
	}
	revision, err := store.setExpected(req.Spec, true, req.ExpectedRevision)
	if err != nil {
		if conflict, ok := err.(artifactRevisionConflictError); ok {
			writeArtifactConflict(w, store.Name(), conflict)
			return
		}
		writeArtifactError(w, http.StatusUnprocessableEntity, "invalid_artifact", err.Error())
		return
	}
	writeArtifactUpdateResponse(w, r, store, revision, api.sameOrigin)
}

func handleArtifactEndpoint(w http.ResponseWriter, r *http.Request, ep artifactEndpoint) {
	if !authenticateArtifactRequest(w, r, ep.auth) {
		return
	}
	switch r.Method {
	case http.MethodGet:
		spec, revision := ep.store.snapshot()
		response := map[string]any{
			"artifact":   ep.store.Name(),
			"revision":   revision,
			"spec":       spec,
			"placements": artifactPlacements(ep.store.Name(), r, ep.sameOrigin),
		}
		if ep.sameOrigin {
			response["app_url"] = requestOrigin(r)
		}
		writeArtifactJSON(w, http.StatusOK, response)
	case http.MethodPost:
		var req struct {
			ExpectedRevision *uint64      `json:"expected_revision"`
			Spec             ArtifactSpec `json:"spec"`
		}
		if !decodeArtifactRequest(w, r, &req) {
			return
		}
		if req.ExpectedRevision == nil {
			writeArtifactError(w, http.StatusBadRequest, "bad_request", "expected_revision is required")
			return
		}
		revision, err := ep.store.setExpected(req.Spec, true, req.ExpectedRevision)
		if err != nil {
			if conflict, ok := err.(artifactRevisionConflictError); ok {
				writeArtifactConflict(w, ep.store.Name(), conflict)
				return
			}
			writeArtifactError(w, http.StatusUnprocessableEntity, "invalid_artifact", err.Error())
			return
		}
		writeArtifactUpdateResponse(w, r, ep.store, revision, ep.sameOrigin)
	default:
		w.Header().Set("Allow", "GET, POST")
		writeArtifactError(w, http.StatusMethodNotAllowed, "method_not_allowed", "use GET or POST")
	}
}

func authenticateArtifactRequest(w http.ResponseWriter, r *http.Request, auth AgentAuthenticator) bool {
	token := bearerToken(r.Header.Get("Authorization"))
	if token == "" {
		writeArtifactError(w, http.StatusUnauthorized, "unauthorized", "missing bearer token")
		return false
	}
	if _, ok, err := auth.AuthenticateAgent(r.Context(), token); err != nil {
		writeArtifactError(w, http.StatusInternalServerError, "auth_error", err.Error())
		return false
	} else if !ok {
		writeArtifactError(w, http.StatusUnauthorized, "unauthorized", "invalid bearer token")
		return false
	}
	return true
}

func decodeArtifactRequest(w http.ResponseWriter, r *http.Request, dst any) bool {
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxArtifactEndpointBody))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		writeArtifactError(w, http.StatusBadRequest, "bad_request", err.Error())
		return false
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			err = fmt.Errorf("request body must contain one JSON object")
		}
		writeArtifactError(w, http.StatusBadRequest, "bad_request", err.Error())
		return false
	}
	return true
}

func writeArtifactUpdateResponse(w http.ResponseWriter, r *http.Request, store *ArtifactStore, revision uint64, sameOrigin bool) {
	placements := artifactPlacements(store.Name(), r, sameOrigin)
	response := map[string]any{
		"ok":         true,
		"artifact":   store.Name(),
		"revision":   revision,
		"placements": placements,
	}
	if sameOrigin {
		response["app_url"] = requestOrigin(r)
	}
	if len(placements) > 0 {
		response["preview"] = placements[0]
	}
	writeArtifactJSON(w, http.StatusOK, response)
}

func bearerToken(header string) string {
	const prefix = "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(header, prefix))
}

func writeArtifactError(w http.ResponseWriter, status int, code, message string) {
	writeArtifactJSON(w, status, map[string]any{
		"ok": false,
		"error": map[string]any{
			"code":    code,
			"message": message,
		},
	})
}

func writeArtifactConflict(w http.ResponseWriter, artifact string, conflict artifactRevisionConflictError) {
	writeArtifactJSON(w, http.StatusConflict, map[string]any{
		"ok":                false,
		"artifact":          artifact,
		"expected_revision": conflict.expected,
		"current_revision":  conflict.current,
		"error": map[string]any{
			"code":    "revision_conflict",
			"message": conflict.Error(),
		},
	})
}

func writeArtifactJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func updateArtifactPlacements(sess *session, root *Node) {
	if sess == nil || root == nil {
		return
	}
	page := sess.activePage()
	observed := make([]observedArtifactPlacement, 0)
	var walk func(*Node)
	walk = func(node *Node) {
		if node == nil {
			return
		}
		if node.Type == "artifact_canvas" {
			name, _ := node.Props["name"].(string)
			if name != "" {
				observed = append(observed, observedArtifactPlacement{
					artifact: name,
					ArtifactPlacement: ArtifactPlacement{
						Page:     page,
						CanvasID: node.ID,
						Selector: artifactCanvasSelector(node.ID),
					},
				})
			}
		}
		for _, child := range node.Children {
			walk(child)
		}
	}
	walk(root)

	artifactPlacementMu.Lock()
	pages := artifactSessionPlacements[sess.id]
	if pages == nil {
		pages = map[string][]observedArtifactPlacement{}
		artifactSessionPlacements[sess.id] = pages
	}
	pages[page] = observed
	artifactPlacementMu.Unlock()
}

func clearArtifactPlacements(sessionID string) {
	artifactPlacementMu.Lock()
	delete(artifactSessionPlacements, sessionID)
	artifactPlacementMu.Unlock()
}

func artifactPlacements(name string, r *http.Request, sameOrigin bool) []ArtifactPlacement {
	artifactPlacementMu.RLock()
	seen := map[string]ArtifactPlacement{}
	for _, pages := range artifactSessionPlacements {
		for _, placements := range pages {
			for _, placement := range placements {
				if placement.artifact != name {
					continue
				}
				p := placement.ArtifactPlacement
				if sameOrigin {
					p.URL = requestOrigin(r)
				}
				key := p.Page + "\x00" + p.URL + "\x00" + p.CanvasID
				seen[key] = p
			}
		}
	}
	artifactPlacementMu.RUnlock()

	out := make([]ArtifactPlacement, 0, len(seen))
	for _, placement := range seen {
		out = append(out, placement)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Page != out[j].Page {
			return out[i].Page < out[j].Page
		}
		if out[i].URL != out[j].URL {
			return out[i].URL < out[j].URL
		}
		return out[i].CanvasID < out[j].CanvasID
	})
	return out
}

func artifactCanvasSelector(canvasID string) string {
	return `[data-artifact-id="` + cssStringEscape(canvasID) + `"]`
}

func cssStringEscape(s string) string {
	return strings.NewReplacer(`\`, `\\`, `"`, `\"`, "\n", `\a `, "\r", `\d `).Replace(s)
}

func requestOrigin(r *http.Request) string {
	if r == nil {
		return ""
	}
	scheme := firstHeaderValue(r.Header.Get("X-Forwarded-Proto"))
	if scheme != "https" {
		if r.TLS != nil {
			scheme = "https"
		} else {
			scheme = "http"
		}
	}
	return scheme + "://" + r.Host + "/"
}

func firstHeaderValue(raw string) string {
	if i := strings.IndexByte(raw, ','); i >= 0 {
		raw = raw[:i]
	}
	return strings.ToLower(strings.TrimSpace(raw))
}
