package syralit

import (
	"bytes"
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/yuin/goldmark"
)

const maxArtifactEndpointBody = 1 << 20

// ArtifactSpec is the safe, agent-friendly DSL for rendering a dynamic canvas.
// It intentionally maps to a curated subset of Syralit components rather than
// exposing the internal Node protocol as a public API.
type ArtifactSpec struct {
	Version string         `json:"version"`
	Layout  ArtifactLayout `json:"layout,omitempty"`
	Data    map[string]any `json:"data,omitempty"`
	Nodes   []ArtifactNode `json:"nodes"`
}

// ArtifactLayout controls the canvas-level responsive layout.
type ArtifactLayout struct {
	Columns int    `json:"columns,omitempty"`
	Gap     int    `json:"gap,omitempty"`
	Padding int    `json:"padding,omitempty"`
	Mode    string `json:"mode,omitempty"`
}

// ArtifactLayoutItem controls a single child inside an ArtifactCanvas.
type ArtifactLayoutItem struct {
	ColumnSpan int `json:"column_span,omitempty"`
	RowSpan    int `json:"row_span,omitempty"`
}

// ArtifactNode describes one safe component instance in an artifact.
type ArtifactNode struct {
	ID        string             `json:"id"`
	Component string             `json:"component"`
	Props     map[string]any     `json:"props,omitempty"`
	Bind      map[string]string  `json:"bind,omitempty"`
	Layout    ArtifactLayoutItem `json:"layout,omitempty"`
	Children  []ArtifactNode     `json:"children,omitempty"`
}

// ArtifactStore holds one shared artifact. A Set broadcasts a rerun so every
// active browser session sees the latest canvas.
type ArtifactStore struct {
	name     string
	mu       sync.RWMutex
	spec     ArtifactSpec
	nodes    []*Node
	updated  time.Time
	revision uint64
}

// NewArtifactStore creates a shared artifact store. Invalid initial specs are
// represented as an error node so app startup remains recoverable.
func NewArtifactStore(name string, initial ArtifactSpec) *ArtifactStore {
	st := &ArtifactStore{name: name}
	if _, err := st.set(initial, false); err != nil {
		st.mu.Lock()
		st.spec = initial
		st.nodes = []*Node{{Type: "status", Props: map[string]any{
			"level": "error",
			"text":  "Artifact error: " + err.Error(),
		}}}
		st.updated = time.Now()
		st.revision = 1
		st.mu.Unlock()
	}
	return st
}

// Name returns the artifact store name.
func (s *ArtifactStore) Name() string {
	if s == nil {
		return ""
	}
	return s.name
}

// Set replaces the artifact with a validated spec and broadcasts a live rerun.
func (s *ArtifactStore) Set(spec ArtifactSpec) error {
	_, err := s.set(spec, true)
	return err
}

func (s *ArtifactStore) set(spec ArtifactSpec, broadcast bool) (uint64, error) {
	return s.setExpected(spec, broadcast, nil)
}

func (s *ArtifactStore) setExpected(spec ArtifactSpec, broadcast bool, expected *uint64) (uint64, error) {
	if s == nil {
		return 0, errors.New("nil artifact store")
	}
	nodes, err := compileArtifactSpec(spec)
	if err != nil {
		return 0, err
	}
	s.mu.Lock()
	if expected != nil && s.revision != *expected {
		current := s.revision
		s.mu.Unlock()
		return current, artifactRevisionConflictError{expected: *expected, current: current}
	}
	s.spec = cloneArtifactSpec(spec)
	s.nodes = cloneNodes(nodes)
	s.updated = time.Now()
	s.revision++
	revision := s.revision
	s.mu.Unlock()
	if broadcast {
		broadcastRerun()
	}
	return revision, nil
}

type artifactRevisionConflictError struct {
	expected uint64
	current  uint64
}

func (e artifactRevisionConflictError) Error() string {
	return fmt.Sprintf("artifact revision conflict: expected %d, current %d", e.expected, e.current)
}

// Spec returns a copy of the current artifact spec.
func (s *ArtifactStore) Spec() ArtifactSpec {
	if s == nil {
		return ArtifactSpec{}
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return cloneArtifactSpec(s.spec)
}

// Revision returns the monotonically increasing revision of the current spec.
// Agents can use it to wait until the browser has finished rendering an update.
func (s *ArtifactStore) Revision() uint64 {
	if s == nil {
		return 0
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.revision
}

func (s *ArtifactStore) snapshot() (ArtifactSpec, uint64) {
	if s == nil {
		return ArtifactSpec{}, 0
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return cloneArtifactSpec(s.spec), s.revision
}

func (s *ArtifactStore) nodesSnapshot() []*Node {
	if s == nil {
		return nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return cloneNodes(s.nodes)
}

func (s *ArtifactStore) layoutSnapshot() ArtifactLayout {
	if s == nil {
		return ArtifactLayout{}
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.spec.Layout
}

// ArtifactCanvas renders a live artifact region. It may be used as a full page
// surface or embedded alongside other Syralit widgets.
func ArtifactCanvas(store *ArtifactStore, opts ...Option) {
	o := applyOpts(opts)
	props := map[string]any{}
	id := "artifact:unknown"
	if store != nil {
		id = "artifact:" + store.Name()
		props["name"] = store.Name()
		props["revision"] = store.Revision()
		layout := store.layoutSnapshot()
		props["layout"] = map[string]any{
			"columns": layout.Columns,
			"gap":     layout.Gap,
			"padding": layout.Padding,
			"mode":    layout.Mode,
		}
	}
	if o.key != "" {
		id = "artifact:" + o.key
	}
	if o.height > 0 {
		props["height"] = o.height
	}
	if o.width > 0 {
		props["width"] = o.width
	}
	current().add(&Node{ID: id, Type: "artifact_canvas", Props: props, Children: store.nodesSnapshot()})
}

func compileArtifactSpec(spec ArtifactSpec) ([]*Node, error) {
	seen := map[string]struct{}{}
	nodes := make([]*Node, 0, len(spec.Nodes))
	for i, n := range spec.Nodes {
		node, err := compileArtifactNode(n, spec.Data, seen, fmt.Sprintf("nodes[%d]", i))
		if err != nil {
			return nil, err
		}
		nodes = append(nodes, node)
	}
	return nodes, nil
}

var artifactAllowedProps = map[string]map[string]bool{
	"text":       {"text": true},
	"markdown":   {"text": true},
	"metric":     {"label": true, "value": true, "delta": true, "delta_color": true, "border": true, "help": true},
	"table":      {"headers": true, "rows": true},
	"dataframe":  {"headers": true, "rows": true, "height": true, "column_config": true},
	"line_chart": {"series": true, "height": true, "width": true, "title": true, "x_labels": true, "stacked": true, "colors": true},
	"bar_chart":  {"series": true, "height": true, "width": true, "title": true, "x_labels": true, "horizontal": true, "stacked": true, "colors": true},
	"pie_chart":  {"data": true, "height": true, "width": true, "title": true, "colors": true},
	"image":      {"src": true, "alt": true, "width": true, "caption": true, "containerWidth": true},
	"progress":   {"value": true, "text": true},
	"container":  {"border": true, "height": true},
}

var artifactComponentTypes = map[string]string{
	"text":       "text",
	"markdown":   "markdown",
	"metric":     "metric",
	"table":      "table",
	"dataframe":  "dataframe",
	"line_chart": "line_chart",
	"bar_chart":  "bar_chart",
	"pie_chart":  "pie_chart",
	"image":      "image",
	"progress":   "progress",
	"container":  "container",
}

func compileArtifactNode(src ArtifactNode, data map[string]any, seen map[string]struct{}, path string) (*Node, error) {
	if src.ID == "" {
		return nil, fmt.Errorf("%s: id is required", path)
	}
	if _, ok := seen[src.ID]; ok {
		return nil, fmt.Errorf("%s: duplicate id %q", path, src.ID)
	}
	seen[src.ID] = struct{}{}
	typ, ok := artifactComponentTypes[src.Component]
	if !ok {
		// Custom component supplied by an integration (e.g. "insyra"). The
		// compiler owns its props validation and produces a render Node; the
		// core only stamps the id and canvas layout onto the result.
		if compiler, found := lookupArtifactComponent(src.Component); found {
			node, err := compiler(src, data)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", path, err)
			}
			if node == nil {
				return nil, fmt.Errorf("%s: component %q produced no node", path, src.Component)
			}
			node.ID = src.ID
			applyArtifactLayout(node, src.Layout)
			return node, nil
		}
		return nil, fmt.Errorf("%s: unsupported component %q", path, src.Component)
	}
	if len(src.Children) > 0 && src.Component != "container" {
		return nil, fmt.Errorf("%s: component %q cannot have children", path, src.Component)
	}

	props := cloneAnyMap(src.Props)
	for target, ptr := range src.Bind {
		val, err := resolveJSONPointer(data, ptr)
		if err != nil {
			return nil, fmt.Errorf("%s: bind %q: %w", path, target, err)
		}
		if err := setArtifactTarget(props, target, val); err != nil {
			return nil, fmt.Errorf("%s: bind %q: %w", path, target, err)
		}
	}
	if err := validateArtifactProps(src.Component, props, path); err != nil {
		return nil, err
	}
	if src.Component == "markdown" {
		text, _ := props["text"].(string)
		props = map[string]any{"html": renderArtifactMarkdown(text)}
	}

	out := &Node{ID: src.ID, Type: typ, Props: props}
	applyArtifactLayout(out, src.Layout)
	for i, child := range src.Children {
		node, err := compileArtifactNode(child, data, seen, fmt.Sprintf("%s.children[%d]", path, i))
		if err != nil {
			return nil, err
		}
		out.Children = append(out.Children, node)
	}
	return out, nil
}

func validateArtifactProps(component string, props map[string]any, path string) error {
	allowed := artifactAllowedProps[component]
	for k := range props {
		if !allowed[k] {
			return fmt.Errorf("%s: prop %q is not allowed for component %q", path, k, component)
		}
	}
	switch component {
	case "text", "markdown":
		if _, ok := props["text"].(string); !ok {
			return fmt.Errorf("%s: component %q requires string prop text", path, component)
		}
	case "metric":
		if _, ok := props["label"].(string); !ok {
			return fmt.Errorf("%s: metric requires string prop label", path)
		}
		if _, ok := props["value"].(string); !ok {
			return fmt.Errorf("%s: metric requires string prop value", path)
		}
	case "table", "dataframe":
		if err := validateStringSliceProp(props, "headers"); err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
		if err := validateRowsProp(props, "rows"); err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
	case "line_chart", "bar_chart":
		if _, ok := props["series"]; !ok {
			return fmt.Errorf("%s: %s requires prop series", path, component)
		}
	case "pie_chart":
		if _, ok := props["data"]; !ok {
			return fmt.Errorf("%s: pie_chart requires prop data", path)
		}
	case "image":
		if _, ok := props["src"].(string); !ok {
			return fmt.Errorf("%s: image requires string prop src", path)
		}
	case "progress":
		if _, ok := props["value"]; !ok {
			return fmt.Errorf("%s: progress requires prop value", path)
		}
	}
	return nil
}

func validateStringSliceProp(props map[string]any, key string) error {
	v, ok := props[key]
	if !ok {
		return fmt.Errorf("%s is required", key)
	}
	switch arr := v.(type) {
	case []string:
		return nil
	case []any:
		for _, item := range arr {
			if _, ok := item.(string); !ok {
				return fmt.Errorf("%s must contain strings", key)
			}
		}
		return nil
	default:
		return fmt.Errorf("%s must be a string array", key)
	}
}

func validateRowsProp(props map[string]any, key string) error {
	v, ok := props[key]
	if !ok {
		return fmt.Errorf("%s is required", key)
	}
	switch rows := v.(type) {
	case [][]string, [][]any:
		return nil
	case []any:
		for _, row := range rows {
			if _, ok := row.([]any); !ok {
				return fmt.Errorf("%s must contain row arrays", key)
			}
		}
		return nil
	default:
		return fmt.Errorf("%s must be an array of rows", key)
	}
}

func setArtifactTarget(props map[string]any, target string, val any) error {
	const prefix = "props."
	if !strings.HasPrefix(target, prefix) {
		return errors.New("target must start with props.")
	}
	key := strings.TrimPrefix(target, prefix)
	if key == "" || strings.Contains(key, ".") {
		return errors.New("target must address one direct prop")
	}
	props[key] = val
	return nil
}

func resolveJSONPointer(data any, ptr string) (any, error) {
	if ptr == "" {
		return data, nil
	}
	if !strings.HasPrefix(ptr, "/") {
		return nil, errors.New("JSON pointer must start with /")
	}
	cur := data
	for _, raw := range strings.Split(ptr[1:], "/") {
		part := strings.ReplaceAll(strings.ReplaceAll(raw, "~1", "/"), "~0", "~")
		switch v := cur.(type) {
		case map[string]any:
			next, ok := v[part]
			if !ok {
				return nil, fmt.Errorf("path %q not found", ptr)
			}
			cur = next
		case []any:
			idx, err := strconv.Atoi(part)
			if err != nil || idx < 0 || idx >= len(v) {
				return nil, fmt.Errorf("array index %q out of range", part)
			}
			cur = v[idx]
		default:
			return nil, fmt.Errorf("path %q crosses non-container value", ptr)
		}
	}
	return cur, nil
}

func renderArtifactMarkdown(text string) string {
	var buf bytes.Buffer
	if err := goldmark.Convert([]byte(text), &buf); err != nil {
		return htmlEscape(text)
	}
	return buf.String()
}

func cloneArtifactSpec(spec ArtifactSpec) ArtifactSpec {
	b, _ := json.Marshal(spec)
	var out ArtifactSpec
	_ = json.Unmarshal(b, &out)
	return out
}

func cloneNodes(nodes []*Node) []*Node {
	if len(nodes) == 0 {
		return nil
	}
	b, _ := json.Marshal(nodes)
	var out []*Node
	_ = json.Unmarshal(b, &out)
	return out
}

// AgentPrincipal identifies the authenticated agent caller.
type AgentPrincipal struct {
	ID   string `json:"id"`
	Name string `json:"name,omitempty"`
}

// AgentAuthenticator verifies a bearer token presented to an artifact endpoint.
type AgentAuthenticator interface {
	AuthenticateAgent(ctx context.Context, token string) (AgentPrincipal, bool, error)
}

// AgentKeyInfo is display metadata for an agent API key. It must not contain
// the plaintext token.
type AgentKeyInfo struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at,omitempty"`
}

// AgentKeyStore lets apps provide their own API-key persistence while Syralit
// supplies the management UI.
type AgentKeyStore interface {
	AgentAuthenticator
	ListAgentKeys(ctx context.Context) ([]AgentKeyInfo, error)
	CreateAgentKey(ctx context.Context, name string) (plainToken string, info AgentKeyInfo, err error)
	RevokeAgentKey(ctx context.Context, id string) error
}

type staticAgentKey struct {
	name  string
	token string
}

// StaticAgentKey creates an authenticator backed by one app-provided token.
func StaticAgentKey(name, token string) AgentAuthenticator {
	return staticAgentKey{name: name, token: token}
}

func (a staticAgentKey) AuthenticateAgent(ctx context.Context, token string) (AgentPrincipal, bool, error) {
	if a.token == "" || token == "" {
		return AgentPrincipal{}, false, nil
	}
	if subtle.ConstantTimeCompare([]byte(a.token), []byte(token)) != 1 {
		return AgentPrincipal{}, false, nil
	}
	return AgentPrincipal{ID: a.name, Name: a.name}, true, nil
}

// AgentKeyManager renders a small key-management panel using an app-provided
// AgentKeyStore. It returns the one-time plaintext token after creation.
func AgentKeyManager(store AgentKeyStore, opts ...Option) string {
	if store == nil {
		Error("Agent key manager requires a key store")
		return ""
	}
	o := applyOpts(opts)
	key := o.key
	if key == "" {
		key = "agent_keys"
	}
	ctx := context.Background()
	var created string

	Container(func() {
		name := TextInput("Key name", Key(key+":name"), Placeholder("local-agent"))
		if Button("Create key", Key(key+":create")) {
			token, info, err := store.CreateAgentKey(ctx, strings.TrimSpace(name))
			if err != nil {
				Error(err.Error())
			} else {
				created = token
				Success("Created key " + info.Name)
				Code(token)
			}
		}

		keys, err := store.ListAgentKeys(ctx)
		if err != nil {
			Error(err.Error())
			return
		}
		if len(keys) == 0 {
			Caption("No agent keys yet.")
			return
		}
		headers := []string{"Name", "ID", "Created"}
		rows := make([][]string, 0, len(keys))
		for _, k := range keys {
			rows = append(rows, []string{k.Name, k.ID, k.CreatedAt.Format(time.RFC3339)})
		}
		Table(headers, rows)
		for _, k := range keys {
			if Button("Revoke "+k.Name, Key(key+":revoke:"+k.ID), ButtonType("secondary")) {
				if err := store.RevokeAgentKey(ctx, k.ID); err != nil {
					Error(err.Error())
				} else {
					Success("Revoked key " + k.Name)
				}
			}
		}
	}, Border())
	return created
}
