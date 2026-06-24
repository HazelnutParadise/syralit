package main

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	sy "github.com/HazelnutParadise/syralit"
)

var board = sy.NewArtifactStore("main", initialArtifact())
var keys = newMemoryAgentKeyStore()

func init() {
	token := sy.Secrets("AGENT_KEY")
	if token == "" {
		token = "dev-agent-key"
	}
	sy.HandleArtifactEndpoint("/api/agent/artifacts/main", board, multiAuth{
		sy.StaticAgentKey("static-local-agent", token),
		keys,
	})
}

func main() {
	sy.App(func() {
		sy.SetPageConfig(sy.PageTitle("Agent Artifact Canvas"), sy.PageLayout("wide"))
		sy.Title("Agent Artifact Canvas")
		sy.Markdown("This example renders a shared artifact canvas from a controlled DSL. POST a new spec to the endpoint and every open browser session updates live.")

		cols := sy.WeightedColumns(2, 1)
		cols[0](func() {
			sy.ArtifactCanvas(board, sy.Height(520))
		})
		cols[1](func() {
			sy.Header("Agent endpoint")
			sy.Code(`curl -X POST http://127.0.0.1:8600/api/agent/artifacts/main \
  -H "Authorization: Bearer dev-agent-key" \
  -H "Content-Type: application/json" \
  -d '{"spec":{"version":"v1","layout":{"columns":2,"gap":14,"padding":16},"data":{"summary":{"revenue":"$58k","delta":"+18%","rows":[["North",22],["South",15],["West",19]]}},"nodes":[{"id":"title","component":"markdown","props":{"text":"## Agent-updated board\nGenerated from a safe DSL."},"layout":{"column_span":2}},{"id":"revenue","component":"metric","props":{"label":"Revenue"},"bind":{"props.value":"/summary/revenue","props.delta":"/summary/delta"}},{"id":"progress","component":"progress","props":{"value":0.72,"text":"Plan confidence"}},{"id":"table","component":"table","props":{"headers":["Region","Deals"]},"bind":{"props.rows":"/summary/rows"},"layout":{"column_span":2}}]}}'`)

			sy.Header("Managed keys")
			sy.AgentKeyManager(keys, sy.Key("agent-key-manager"))
		})
	})
}

func initialArtifact() sy.ArtifactSpec {
	return sy.ArtifactSpec{
		Version: "v1",
		Layout:  sy.ArtifactLayout{Columns: 2, Gap: 14, Padding: 16},
		Data: map[string]any{
			"summary": map[string]any{
				"revenue": "$42k",
				"delta":   "+9%",
				"rows": []any{
					[]any{"North", 12},
					[]any{"South", 18},
					[]any{"West", 9},
				},
			},
		},
		Nodes: []sy.ArtifactNode{
			{
				ID:        "intro",
				Component: "markdown",
				Props: map[string]any{
					"text": "## Live agent artifact\nThe board below is composed from reusable Syralit components.",
				},
				Layout: sy.ArtifactLayoutItem{ColumnSpan: 2},
			},
			{
				ID:        "revenue",
				Component: "metric",
				Props:     map[string]any{"label": "Revenue"},
				Bind: map[string]string{
					"props.value": "/summary/revenue",
					"props.delta": "/summary/delta",
				},
			},
			{
				ID:        "progress",
				Component: "progress",
				Props: map[string]any{
					"value": 0.62,
					"text":  "Artifact readiness",
				},
			},
			{
				ID:        "regions",
				Component: "table",
				Props:     map[string]any{"headers": []any{"Region", "Deals"}},
				Bind:      map[string]string{"props.rows": "/summary/rows"},
				Layout:    sy.ArtifactLayoutItem{ColumnSpan: 2},
			},
		},
	}
}

type multiAuth []sy.AgentAuthenticator

func (m multiAuth) AuthenticateAgent(ctx context.Context, token string) (sy.AgentPrincipal, bool, error) {
	for _, auth := range m {
		principal, ok, err := auth.AuthenticateAgent(ctx, token)
		if err != nil || ok {
			return principal, ok, err
		}
	}
	return sy.AgentPrincipal{}, false, nil
}

type memoryAgentKeyStore struct {
	mu     sync.Mutex
	tokens map[string]string
	infos  map[string]sy.AgentKeyInfo
}

func newMemoryAgentKeyStore() *memoryAgentKeyStore {
	return &memoryAgentKeyStore{
		tokens: map[string]string{},
		infos:  map[string]sy.AgentKeyInfo{},
	}
}

func (s *memoryAgentKeyStore) AuthenticateAgent(ctx context.Context, token string) (sy.AgentPrincipal, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for id, stored := range s.tokens {
		if subtle.ConstantTimeCompare([]byte(stored), []byte(token)) == 1 {
			info := s.infos[id]
			return sy.AgentPrincipal{ID: info.ID, Name: info.Name}, true, nil
		}
	}
	return sy.AgentPrincipal{}, false, nil
}

func (s *memoryAgentKeyStore) ListAgentKeys(ctx context.Context) ([]sy.AgentKeyInfo, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]sy.AgentKeyInfo, 0, len(s.infos))
	for _, info := range s.infos {
		out = append(out, info)
	}
	return out, nil
}

func (s *memoryAgentKeyStore) CreateAgentKey(ctx context.Context, name string) (string, sy.AgentKeyInfo, error) {
	if name == "" {
		name = "agent"
	}
	token, err := randomToken()
	if err != nil {
		return "", sy.AgentKeyInfo{}, err
	}
	id, err := randomToken()
	if err != nil {
		return "", sy.AgentKeyInfo{}, err
	}
	info := sy.AgentKeyInfo{
		ID:        id[:12],
		Name:      name,
		CreatedAt: time.Now(),
	}
	s.mu.Lock()
	s.tokens[info.ID] = token
	s.infos[info.ID] = info
	s.mu.Unlock()
	return token, info, nil
}

func (s *memoryAgentKeyStore) RevokeAgentKey(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.infos[id]; !ok {
		return fmt.Errorf("agent key %q not found", id)
	}
	delete(s.tokens, id)
	delete(s.infos, id)
	return nil
}

func randomToken() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}
