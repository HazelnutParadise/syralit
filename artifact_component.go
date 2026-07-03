package syralit

import (
	"sort"
	"sync"
)

// ArtifactComponentCompiler compiles a custom artifact component into a render
// Node. It receives the raw ArtifactNode and the spec-level Data map, and must
// return a single Node whose Type is one the client runtime already renders
// (e.g. "dataframe", "table", "bar_chart", "line_chart", "pie_chart", "metric",
// "text"). The core applies the node's ID and layout after the compiler
// returns, so the compiler owns only its own props validation and construction.
// Returning an error fails the whole artifact compile.
//
// Registered components let optional integrations extend the artifact DSL
// without the core package importing them. This keeps the core/integration
// boundary intact: the Insyra integration, for example, registers an "insyra"
// component that runs an Insyra DSL script server-side, while the core package
// never imports Insyra.
type ArtifactComponentCompiler func(node ArtifactNode, data map[string]any) (*Node, error)

var (
	artifactComponentMu       sync.RWMutex
	artifactComponentRegistry = map[string]ArtifactComponentCompiler{}
)

// RegisterArtifactComponent registers a custom artifact component compiler under
// name, making <name> usable as an ArtifactNode.Component in any ArtifactSpec.
// It is intended to be called from an integration package's init (for example,
// the Insyra integration registers "insyra").
//
// It panics on an empty name, a nil compiler, a name that collides with a
// built-in component, or a name already registered — all of which are program
// wiring errors that should surface at startup rather than at compile time.
func RegisterArtifactComponent(name string, compiler ArtifactComponentCompiler) {
	if name == "" {
		panic("syralit: RegisterArtifactComponent: empty name")
	}
	if compiler == nil {
		panic("syralit: RegisterArtifactComponent: nil compiler for " + name)
	}
	if _, ok := artifactComponentTypes[name]; ok {
		panic("syralit: RegisterArtifactComponent: name collides with built-in component: " + name)
	}
	artifactComponentMu.Lock()
	defer artifactComponentMu.Unlock()
	if _, ok := artifactComponentRegistry[name]; ok {
		panic("syralit: RegisterArtifactComponent: already registered: " + name)
	}
	artifactComponentRegistry[name] = compiler
}

func lookupArtifactComponent(name string) (ArtifactComponentCompiler, bool) {
	artifactComponentMu.RLock()
	defer artifactComponentMu.RUnlock()
	compiler, ok := artifactComponentRegistry[name]
	return compiler, ok
}

// builtinArtifactComponents returns the always-available component names, sorted.
func builtinArtifactComponents() []string {
	names := make([]string, 0, len(artifactComponentTypes))
	for name := range artifactComponentTypes {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// registeredArtifactComponents returns the names of custom components added via
// RegisterArtifactComponent (e.g. "insyra" when the Insyra DSL package is
// imported), sorted.
func registeredArtifactComponents() []string {
	artifactComponentMu.RLock()
	defer artifactComponentMu.RUnlock()
	names := make([]string, 0, len(artifactComponentRegistry))
	for name := range artifactComponentRegistry {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// artifactComponentsInfo describes the components this app build supports, for
// the discovery endpoint. "custom" lists the opt-in components an integration
// registered; an agent can check it before using one like "insyra".
func artifactComponentsInfo() map[string]any {
	return map[string]any{
		"builtin": builtinArtifactComponents(),
		"custom":  registeredArtifactComponents(),
	}
}

// applyArtifactLayout writes the canvas grid span for a compiled node. Shared by
// the built-in and registered-component compile paths.
func applyArtifactLayout(node *Node, layout ArtifactLayoutItem) {
	if layout.ColumnSpan <= 0 && layout.RowSpan <= 0 {
		return
	}
	if node.Props == nil {
		node.Props = map[string]any{}
	}
	l := map[string]any{}
	if layout.ColumnSpan > 0 {
		l["column_span"] = layout.ColumnSpan
	}
	if layout.RowSpan > 0 {
		l["row_span"] = layout.RowSpan
	}
	node.Props["artifact_layout"] = l
}
