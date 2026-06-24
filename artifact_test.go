package syralit

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func resetArtifactEndpoints() {
	artifactEndpointMu.Lock()
	artifactEndpoints = map[string]artifactEndpoint{}
	artifactEndpointMu.Unlock()
}

func validArtifactSpec() ArtifactSpec {
	return ArtifactSpec{
		Version: "v1",
		Layout:  ArtifactLayout{Columns: 2, Gap: 12, Padding: 16},
		Data: map[string]any{
			"summary": map[string]any{
				"revenue": "$42k",
				"rows": []any{
					[]any{"North", 12},
					[]any{"South", 18},
				},
			},
		},
		Nodes: []ArtifactNode{
			{
				ID:        "headline",
				Component: "text",
				Props:     map[string]any{"text": "Live artifact"},
			},
			{
				ID:        "revenue",
				Component: "metric",
				Props:     map[string]any{"label": "Revenue"},
				Bind:      map[string]string{"props.value": "/summary/revenue"},
			},
			{
				ID:        "table",
				Component: "table",
				Props:     map[string]any{"headers": []any{"Region", "Deals"}},
				Bind:      map[string]string{"props.rows": "/summary/rows"},
			},
		},
	}
}

func TestArtifactSpecValidationAndBinding(t *testing.T) {
	nodes, err := compileArtifactSpec(validArtifactSpec())
	if err != nil {
		t.Fatalf("compile artifact: %v", err)
	}
	if len(nodes) != 3 {
		t.Fatalf("expected 3 nodes, got %d", len(nodes))
	}
	if nodes[1].Type != "metric" {
		t.Fatalf("expected metric node, got %q", nodes[1].Type)
	}
	if got, _ := nodes[1].Props["value"].(string); got != "$42k" {
		t.Fatalf("expected bound revenue '$42k', got %q", got)
	}
}

func TestArtifactSpecRejectsUnsafeShape(t *testing.T) {
	tests := []struct {
		name string
		mut  func(*ArtifactSpec)
	}{
		{
			name: "missing id",
			mut:  func(s *ArtifactSpec) { s.Nodes[0].ID = "" },
		},
		{
			name: "unknown component",
			mut:  func(s *ArtifactSpec) { s.Nodes[0].Component = "html" },
		},
		{
			name: "unsafe prop",
			mut:  func(s *ArtifactSpec) { s.Nodes[0].Props["html"] = "<script>alert(1)</script>" },
		},
		{
			name: "bad pointer",
			mut:  func(s *ArtifactSpec) { s.Nodes[1].Bind["props.value"] = "summary/revenue" },
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec := validArtifactSpec()
			tt.mut(&spec)
			if _, err := compileArtifactSpec(spec); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestArtifactCanvasRendersNodeTree(t *testing.T) {
	board := NewArtifactStore("main", validArtifactSpec())
	root := RenderOnce(func() {
		ArtifactCanvas(board, Height(420))
	})
	canvas := root.Find("artifact_canvas")
	if len(canvas) != 1 {
		t.Fatalf("expected one artifact_canvas, got %d", len(canvas))
	}
	if canvas[0].ID != "artifact:main" {
		t.Fatalf("unexpected canvas id %q", canvas[0].ID)
	}
	if got, _ := canvas[0].Props["height"].(int); got != 420 {
		t.Fatalf("expected height 420, got %v", canvas[0].Props["height"])
	}
	if len(canvas[0].Children) != 3 {
		t.Fatalf("expected 3 artifact children, got %d", len(canvas[0].Children))
	}
}

func TestArtifactEndpointAuthAndUpdate(t *testing.T) {
	defer resetArtifactEndpoints()
	resetArtifactEndpoints()

	board := NewArtifactStore("main", validArtifactSpec())
	HandleArtifactEndpoint("/api/agent/artifacts/main", board, StaticAgentKey("local-agent", "secret"))
	srv := httptest.NewServer((&server{cfg: Config{}, appFn: func() { ArtifactCanvas(board) }}).handler())
	defer srv.Close()

	spec := validArtifactSpec()
	spec.Nodes[0].Props["text"] = "Updated artifact"
	body, _ := json.Marshal(map[string]any{"spec": spec})

	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/api/agent/artifacts/main", bytes.NewReader(body))
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("post without token: %v", err)
	}
	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 without token, got %d", res.StatusCode)
	}
	_ = res.Body.Close()

	req, _ = http.NewRequest(http.MethodPost, srv.URL+"/api/agent/artifacts/main", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer wrong")
	res, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("post wrong token: %v", err)
	}
	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 for wrong token, got %d", res.StatusCode)
	}
	_ = res.Body.Close()

	req, _ = http.NewRequest(http.MethodPost, srv.URL+"/api/agent/artifacts/main", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer secret")
	req.Header.Set("Content-Type", "application/json")
	res, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("post valid artifact: %v", err)
	}
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	_ = res.Body.Close()

	nodes := board.nodesSnapshot()
	if got, _ := nodes[0].Props["text"].(string); got != "Updated artifact" {
		t.Fatalf("store did not update, got %q", got)
	}
}

func TestArtifactEndpointRejectsBadPayload(t *testing.T) {
	defer resetArtifactEndpoints()
	resetArtifactEndpoints()

	board := NewArtifactStore("main", validArtifactSpec())
	HandleArtifactEndpoint("/api/agent/artifacts/main", board, StaticAgentKey("local-agent", "secret"))
	srv := httptest.NewServer((&server{cfg: Config{}, appFn: func() { ArtifactCanvas(board) }}).handler())
	defer srv.Close()

	spec := validArtifactSpec()
	spec.Nodes[0].Component = "component"
	body, _ := json.Marshal(map[string]any{"spec": spec})
	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/api/agent/artifacts/main", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer secret")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("post bad artifact: %v", err)
	}
	if res.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d", res.StatusCode)
	}
	_ = res.Body.Close()
}

func TestStaticAgentKey(t *testing.T) {
	auth := StaticAgentKey("local-agent", "secret")
	principal, ok, err := auth.AuthenticateAgent(context.Background(), "secret")
	if err != nil {
		t.Fatalf("auth returned error: %v", err)
	}
	if !ok {
		t.Fatal("expected token to authenticate")
	}
	if principal.ID != "local-agent" {
		t.Fatalf("unexpected principal %q", principal.ID)
	}
}
