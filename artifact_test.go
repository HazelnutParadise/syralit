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
	artifactAPIs = map[string]*artifactAPI{}
	artifactEndpointMu.Unlock()
	artifactPlacementMu.Lock()
	artifactSessionPlacements = map[string]map[string][]observedArtifactPlacement{}
	artifactPlacementMu.Unlock()
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
	if got, _ := canvas[0].Props["revision"].(uint64); got != 1 {
		t.Fatalf("expected initial revision 1, got %v", canvas[0].Props["revision"])
	}
}

func TestArtifactStoreRevisionOnlyAdvancesOnValidUpdate(t *testing.T) {
	board := NewArtifactStore("main", validArtifactSpec())
	if got := board.Revision(); got != 1 {
		t.Fatalf("expected initial revision 1, got %d", got)
	}
	next := validArtifactSpec()
	next.Nodes[0].Props["text"] = "Revision two"
	if err := board.Set(next); err != nil {
		t.Fatalf("set valid spec: %v", err)
	}
	if got := board.Revision(); got != 2 {
		t.Fatalf("expected revision 2, got %d", got)
	}
	invalid := validArtifactSpec()
	invalid.Nodes[0].Component = "html"
	if err := board.Set(invalid); err == nil {
		t.Fatal("expected invalid update to fail")
	}
	if got := board.Revision(); got != 2 {
		t.Fatalf("invalid update advanced revision to %d", got)
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
	body, _ := json.Marshal(map[string]any{"expected_revision": 1, "spec": spec})

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
	body, _ := json.Marshal(map[string]any{"expected_revision": 1, "spec": spec})
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

func TestUnifiedArtifactAPIDiscoversReadsAndUpdatesStores(t *testing.T) {
	defer resetArtifactEndpoints()
	resetArtifactEndpoints()

	mainBoard := NewArtifactStore("main", validArtifactSpec())
	notesSpec := validArtifactSpec()
	notesSpec.Nodes[0].Props["text"] = "Notes"
	notesBoard := NewArtifactStore("notes", notesSpec)
	HandleArtifactAPI(
		"/api/agent/artifacts",
		StaticAgentKey("local-agent", "secret"),
		mainBoard,
		notesBoard,
	)
	srv := httptest.NewServer((&server{cfg: Config{}, appFn: func() {
		ArtifactCanvas(mainBoard)
		ArtifactCanvas(notesBoard)
	}}).handler())
	defer srv.Close()

	get := func(path string) map[string]any {
		t.Helper()
		req, _ := http.NewRequest(http.MethodGet, srv.URL+path, nil)
		req.Header.Set("Authorization", "Bearer secret")
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("GET %s: %v", path, err)
		}
		defer res.Body.Close()
		if res.StatusCode != http.StatusOK {
			t.Fatalf("GET %s returned %d", path, res.StatusCode)
		}
		var body map[string]any
		if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
			t.Fatalf("decode GET %s: %v", path, err)
		}
		return body
	}

	discovery := get("/api/agent/artifacts")
	artifacts, ok := discovery["artifacts"].([]any)
	if !ok || len(artifacts) != 2 {
		t.Fatalf("expected two discovered artifacts, got %#v", discovery["artifacts"])
	}
	if discovery["app_url"] != srv.URL+"/" {
		t.Fatalf("expected trusted API origin as app_url, got %#v", discovery["app_url"])
	}
	for _, artifact := range artifacts {
		item := artifact.(map[string]any)
		if placements, _ := item["placements"].([]any); len(placements) != 0 {
			t.Fatalf("unrendered artifact received a fake placement: %#v", item)
		}
	}
	detail := get("/api/agent/artifacts?artifact=notes")
	if detail["artifact"] != "notes" {
		t.Fatalf("unexpected detail response %#v", detail)
	}

	update := validArtifactSpec()
	update.Nodes[0].Props["text"] = "Only notes changed"
	payload, _ := json.Marshal(map[string]any{
		"artifact":          "notes",
		"expected_revision": 1,
		"spec":              update,
	})
	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/api/agent/artifacts", bytes.NewReader(payload))
	req.Header.Set("Authorization", "Bearer secret")
	req.Header.Set("Content-Type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST unified API: %v", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("POST unified API returned %d", res.StatusCode)
	}
	var response map[string]any
	if err := json.NewDecoder(res.Body).Decode(&response); err != nil {
		t.Fatalf("decode update response: %v", err)
	}
	if response["artifact"] != "notes" || response["revision"] != float64(2) {
		t.Fatalf("unexpected update response %#v", response)
	}
	if got := notesBoard.Spec().Nodes[0].Props["text"]; got != "Only notes changed" {
		t.Fatalf("notes store did not update: %v", got)
	}
	if got := mainBoard.Spec().Nodes[0].Props["text"]; got != "Live artifact" {
		t.Fatalf("main store changed unexpectedly: %v", got)
	}

	req, _ = http.NewRequest(http.MethodPost, srv.URL+"/api/agent/artifacts", bytes.NewReader(payload))
	req.Header.Set("Authorization", "Bearer secret")
	req.Header.Set("Content-Type", "application/json")
	res, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST stale revision: %v", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409 for stale revision, got %d", res.StatusCode)
	}
}

func TestArtifactAPIHandlerCanRunOnAnotherServer(t *testing.T) {
	defer resetArtifactEndpoints()
	resetArtifactEndpoints()

	board := NewArtifactStore("main", validArtifactSpec())
	srv := httptest.NewServer(ArtifactAPIHandler(StaticAgentKey("agent", "secret"), board))
	defer srv.Close()

	req, _ := http.NewRequest(http.MethodGet, srv.URL, nil)
	req.Header.Set("Authorization", "Bearer secret")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET external artifact API: %v", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	var body struct {
		Artifacts []struct {
			ID         string              `json:"id"`
			Placements []ArtifactPlacement `json:"placements"`
		} `json:"artifacts"`
	}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("decode discovery: %v", err)
	}
	if len(body.Artifacts) != 1 || body.Artifacts[0].ID != "main" {
		t.Fatalf("unexpected discovery %#v", body.Artifacts)
	}
	if len(body.Artifacts[0].Placements) != 0 {
		t.Fatalf("external handler invented a same-origin placement: %#v", body.Artifacts[0].Placements)
	}
}

func TestArtifactRoutesRejectDuplicatePaths(t *testing.T) {
	defer resetArtifactEndpoints()
	resetArtifactEndpoints()

	board := NewArtifactStore("main", validArtifactSpec())
	auth := StaticAgentKey("agent", "secret")
	HandleArtifactEndpoint("/api/artifacts", board, auth)
	defer func() {
		if recover() == nil {
			t.Fatal("expected duplicate artifact route to panic")
		}
	}()
	HandleArtifactAPI("/api/artifacts", auth, board)
}

func TestArtifactPlacementsTrackPageAndSelector(t *testing.T) {
	defer resetArtifactEndpoints()
	resetArtifactEndpoints()

	sess := newSession(nil)
	sess.currentPage = "Analytics"
	sess.reqCtx = RequestContext{
		Host: "app.example.test",
		Headers: map[string]string{
			"Origin": "https://app.example.test",
		},
	}
	root := &Node{Type: "root", Children: []*Node{{
		ID:   "artifact:overview",
		Type: "artifact_canvas",
		Props: map[string]any{
			"name": "main",
		},
	}}}
	updateArtifactPlacements(sess, root)
	placements := artifactPlacements("main", nil, false)
	if len(placements) != 1 {
		t.Fatalf("expected one placement, got %#v", placements)
	}
	if placements[0].Page != "Analytics" ||
		placements[0].URL != "" ||
		placements[0].Selector != `[data-artifact-id="artifact:overview"]` {
		t.Fatalf("unexpected placement %#v", placements[0])
	}

	req := httptest.NewRequest(http.MethodGet, "http://trusted.example/api/artifacts", nil)
	req.Header.Set("X-Forwarded-Proto", "https")
	trusted := artifactPlacements("main", req, true)
	if len(trusted) != 1 || trusted[0].URL != "https://trusted.example/" {
		t.Fatalf("same-origin placement did not use API request origin: %#v", trusted)
	}
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
