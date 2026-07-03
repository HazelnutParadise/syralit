package syralit

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func echoCompiler(node ArtifactNode, data map[string]any) (*Node, error) {
	msg, _ := node.Props["msg"].(string)
	return &Node{Type: "text", Props: map[string]any{"text": "echo:" + msg}}, nil
}

func TestRegisterArtifactComponentCompiles(t *testing.T) {
	RegisterArtifactComponent("test_echo", echoCompiler)

	spec := ArtifactSpec{
		Version: "v1",
		Nodes: []ArtifactNode{{
			ID:        "e",
			Component: "test_echo",
			Props:     map[string]any{"msg": "hi"},
			Layout:    ArtifactLayoutItem{ColumnSpan: 2},
		}},
	}
	nodes, err := compileArtifactSpec(spec)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	if len(nodes) != 1 {
		t.Fatalf("want 1 node, got %d", len(nodes))
	}
	if nodes[0].Type != "text" {
		t.Fatalf("type = %q, want text", nodes[0].Type)
	}
	if nodes[0].ID != "e" {
		t.Fatalf("id = %q; core should stamp the node id", nodes[0].ID)
	}
	if got, _ := nodes[0].Props["text"].(string); got != "echo:hi" {
		t.Fatalf("text = %q, want echo:hi", got)
	}
	layout, _ := nodes[0].Props["artifact_layout"].(map[string]any)
	if layout == nil || layout["column_span"] != 2 {
		t.Fatalf("layout = %v; core should apply canvas layout", nodes[0].Props["artifact_layout"])
	}
}

func TestRegisterArtifactComponentErrorsPropagate(t *testing.T) {
	RegisterArtifactComponent("test_boom", func(ArtifactNode, map[string]any) (*Node, error) {
		return nil, errors.New("kaboom")
	})
	_, err := compileArtifactSpec(ArtifactSpec{Nodes: []ArtifactNode{{ID: "b", Component: "test_boom"}}})
	if err == nil || !strings.Contains(err.Error(), "kaboom") {
		t.Fatalf("err = %v, want kaboom", err)
	}
}

func TestRegisterArtifactComponentNilNodeErrors(t *testing.T) {
	RegisterArtifactComponent("test_nilnode", func(ArtifactNode, map[string]any) (*Node, error) {
		return nil, nil
	})
	_, err := compileArtifactSpec(ArtifactSpec{Nodes: []ArtifactNode{{ID: "n", Component: "test_nilnode"}}})
	if err == nil || !strings.Contains(err.Error(), "produced no node") {
		t.Fatalf("err = %v, want 'produced no node'", err)
	}
}

func TestRegisterArtifactComponentPanics(t *testing.T) {
	noop := func(ArtifactNode, map[string]any) (*Node, error) { return &Node{Type: "text"}, nil }
	cases := []struct {
		name string
		fn   func()
	}{
		{"empty name", func() { RegisterArtifactComponent("", noop) }},
		{"nil compiler", func() { RegisterArtifactComponent("test_nilcompiler", nil) }},
		{"builtin collision", func() { RegisterArtifactComponent("metric", noop) }},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Fatalf("expected panic")
				}
			}()
			c.fn()
		})
	}
}

func TestArtifactDiscoveryReportsComponents(t *testing.T) {
	RegisterArtifactComponent("test_disco", echoCompiler)
	store := NewArtifactStore("disco", ArtifactSpec{
		Version: "v1",
		Nodes:   []ArtifactNode{{ID: "n", Component: "text", Props: map[string]any{"text": "hi"}}},
	})
	handler := ArtifactAPIHandler(StaticAgentKey("agent", "tok"), store)

	req := httptest.NewRequest(http.MethodGet, "/api/agent/artifacts", nil)
	req.Header.Set("Authorization", "Bearer tok")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var resp struct {
		Components struct {
			Builtin []string `json:"builtin"`
			Custom  []string `json:"custom"`
		} `json:"components"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v (body=%s)", err, rec.Body.String())
	}
	if !containsStr(resp.Components.Builtin, "metric") {
		t.Fatalf("builtin should include metric: %v", resp.Components.Builtin)
	}
	if !containsStr(resp.Components.Custom, "test_disco") {
		t.Fatalf("custom should include the registered test_disco: %v", resp.Components.Custom)
	}
}

func containsStr(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}

func TestRegisterArtifactComponentDuplicatePanics(t *testing.T) {
	noop := func(ArtifactNode, map[string]any) (*Node, error) { return &Node{Type: "text"}, nil }
	RegisterArtifactComponent("test_dup", noop)
	defer func() {
		if recover() == nil {
			t.Fatalf("expected panic on duplicate registration")
		}
	}()
	RegisterArtifactComponent("test_dup", noop)
}
