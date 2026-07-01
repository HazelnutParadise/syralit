package main

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"strconv"
	"sync"
	"time"

	sy "github.com/HazelnutParadise/syralit"
)

var mainBoard = sy.NewArtifactStore("main", executiveMainSpec())
var notesBoard = sy.NewArtifactStore("notes", executiveNotesSpec())
var keys = newMemoryAgentKeyStore()

func init() {
	token := sy.Secrets("AGENT_KEY")
	if token == "" {
		token = "dev-agent-key"
	}
	auth := multiAuth{
		sy.StaticAgentKey("static-local-agent", token),
		keys,
	}
	sy.HandleArtifactAPI("/api/agent/artifacts", auth, mainBoard, notesBoard)
}

func main() {
	sy.App(func() {
		sy.SetPageConfig(sy.PageTitle("Artifact Canvas Studio"), sy.PageLayout("wide"))
		sy.Title("Artifact Canvas Studio")
		sy.Markdown("This example app shows how one Syralit page can host multiple live Artifact Canvas regions. You can update them from local controls or from an external agent through opt-in bearer-token endpoints.")

		cols := sy.WeightedColumns(2.2, 1)
		cols[0](func() {
			sy.Header("Main board")
			sy.ArtifactCanvas(mainBoard, sy.Height(430))

			sy.Header("Agent notes")
			sy.ArtifactCanvas(notesBoard, sy.Height(210))
		})
		cols[1](func() {
			renderControls()
		})

		sy.Divider()
		sy.Header("Current DSL")
		tabs := sy.Tabs([]string{"Main spec", "Notes spec"})
		tabs("Main spec", func() {
			sy.JSON(mainBoard.Spec())
		})
		tabs("Notes spec", func() {
			sy.JSON(notesBoard.Spec())
		})
	})
}

func renderControls() {
	sy.Header("Scenario presets")
	if sy.Button("Executive brief", sy.Key("preset-exec")) {
		if err := applyPreset(executiveMainSpec(), executiveNotesSpec()); err != nil {
			sy.Error(err.Error())
		} else {
			sy.Rerun()
		}
	}
	if sy.Button("Pipeline review", sy.Key("preset-pipeline"), sy.ButtonType("secondary")) {
		if err := applyPreset(pipelineMainSpec(), pipelineNotesSpec()); err != nil {
			sy.Error(err.Error())
		} else {
			sy.Rerun()
		}
	}
	if sy.Button("Incident room", sy.Key("preset-incident"), sy.ButtonType("secondary")) {
		if err := applyPreset(incidentMainSpec(), incidentNotesSpec()); err != nil {
			sy.Error(err.Error())
		} else {
			sy.Rerun()
		}
	}

	sy.Header("Compose a quick update")
	sy.Form("compose-update", func() {
		headline := sy.TextInput("Headline", sy.Key("compose-headline"), sy.DefaultValue("## Agent update\nA safe DSL can still feel alive."))
		kpiLabel := sy.TextInput("KPI label", sy.Key("compose-kpi-label"), sy.DefaultValue("Confidence"))
		kpiValue := sy.TextInput("KPI value", sy.Key("compose-kpi-value"), sy.DefaultValue("82%"))
		progress := sy.NumberInput("Progress", sy.Key("compose-progress"), sy.DefaultValue(0.82), sy.Min(0), sy.Max(1), sy.Step(0.01))
		note := sy.TextArea("Agent note", sy.Key("compose-note"), sy.DefaultValue("Shift the board toward risks, blockers, and decisions."), sy.Height(110))
		if sy.FormSubmitButton("Apply local update", sy.Key("compose-submit")) {
			if err := applyPreset(customMainSpec(headline, kpiLabel, kpiValue, progress), customNotesSpec(note)); err != nil {
				sy.Error(err.Error())
			} else {
				sy.Rerun()
			}
		}
	})

	sy.Header("Agent endpoints")
	endpoint := "${SYRALIT_URL}/api/agent/artifacts"
	sy.Code(`curl ` + endpoint + ` \
  -H "Authorization: Bearer dev-agent-key"`)
	revision := strconv.FormatUint(mainBoard.Revision(), 10)
	sy.Code(`curl -X POST ` + endpoint + ` \
  -H "Authorization: Bearer dev-agent-key" \
  -H "Content-Type: application/json" \
  -d '{"artifact":"main","expected_revision":` + revision + `,"spec":{"version":"v1","layout":{"columns":2,"gap":14,"padding":16},"data":{"summary":{"headline":"## Board refresh\nShifted by an agent.","revenue":"$58k","delta":"+18%","rows":[["North",22],["South",15],["West",19]]}},"nodes":[{"id":"headline","component":"markdown","props":{"text":"placeholder"},"bind":{"props.text":"/summary/headline"},"layout":{"column_span":2}},{"id":"revenue","component":"metric","props":{"label":"Revenue"},"bind":{"props.value":"/summary/revenue","props.delta":"/summary/delta"}},{"id":"progress","component":"progress","props":{"value":0.72,"text":"Plan confidence"}},{"id":"table","component":"table","props":{"headers":["Region","Deals"]},"bind":{"props.rows":"/summary/rows"},"layout":{"column_span":2}}]}}'`)

	sy.Header("Managed keys")
	sy.AgentKeyManager(keys, sy.Key("agent-key-manager"))
}

func applyPreset(mainSpec, noteSpec sy.ArtifactSpec) error {
	if err := mainBoard.Set(mainSpec); err != nil {
		return err
	}
	if err := notesBoard.Set(noteSpec); err != nil {
		return err
	}
	return nil
}

func executiveMainSpec() sy.ArtifactSpec {
	return sy.ArtifactSpec{
		Version: "v1",
		Layout:  sy.ArtifactLayout{Columns: 2, Gap: 14, Padding: 16},
		Data: map[string]any{
			"summary": map[string]any{
				"headline": "## Executive brief\nRevenue is ahead of plan and the conversion gap is narrowing.",
				"revenue":  "$42k",
				"delta":    "+9%",
				"rows": []any{
					[]any{"North", 12},
					[]any{"South", 18},
					[]any{"West", 9},
				},
			},
		},
		Nodes: []sy.ArtifactNode{
			{
				ID:        "headline",
				Component: "markdown",
				Props:     map[string]any{"text": "placeholder"},
				Bind:      map[string]string{"props.text": "/summary/headline"},
				Layout:    sy.ArtifactLayoutItem{ColumnSpan: 2},
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

func executiveNotesSpec() sy.ArtifactSpec {
	return singleNoteSpec("### Agent note\nThe brief is strong enough for a leadership sync. Next best improvement is a second KPI block for margin or pipeline risk.")
}

func pipelineMainSpec() sy.ArtifactSpec {
	return sy.ArtifactSpec{
		Version: "v1",
		Layout:  sy.ArtifactLayout{Columns: 3, Gap: 14, Padding: 16},
		Data: map[string]any{
			"summary": map[string]any{
				"qualified": "31",
				"delta":     "+6",
				"coverage":  "3.4x",
			},
		},
		Nodes: []sy.ArtifactNode{
			{
				ID:        "headline",
				Component: "markdown",
				Props: map[string]any{
					"text": "## Pipeline review\nCoverage is healthy, but the top funnel is uneven across regions.",
				},
				Layout: sy.ArtifactLayoutItem{ColumnSpan: 3},
			},
			{
				ID:        "qualified",
				Component: "metric",
				Props: map[string]any{
					"label": "Qualified deals",
				},
				Bind: map[string]string{
					"props.value": "/summary/qualified",
					"props.delta": "/summary/delta",
				},
			},
			{
				ID:        "coverage",
				Component: "metric",
				Props: map[string]any{
					"label": "Coverage",
					"value": "3.4x",
				},
				Bind: map[string]string{
					"props.value": "/summary/coverage",
				},
			},
			{
				ID:        "health",
				Component: "progress",
				Props: map[string]any{
					"value": 0.74,
					"text":  "Pipeline health",
				},
			},
			{
				ID:        "mix",
				Component: "bar_chart",
				Props: map[string]any{
					"title":    "Regional mix",
					"x_labels": []any{"North", "South", "West", "APAC"},
					"series": map[string]any{
						"Deals": []any{14, 9, 5, 3},
					},
				},
				Layout: sy.ArtifactLayoutItem{ColumnSpan: 2},
			},
			{
				ID:        "stages",
				Component: "table",
				Props: map[string]any{
					"headers": []any{"Stage", "Count"},
					"rows": []any{
						[]any{"Qualified", 31},
						[]any{"Proposal", 18},
						[]any{"Commit", 7},
					},
				},
			},
		},
	}
}

func pipelineNotesSpec() sy.ArtifactSpec {
	return singleNoteSpec("### Agent note\nTop-of-funnel softness is the real story. The board should steer the team toward regional sourcing, not just celebrate overall coverage.")
}

func incidentMainSpec() sy.ArtifactSpec {
	return sy.ArtifactSpec{
		Version: "v1",
		Layout:  sy.ArtifactLayout{Columns: 2, Gap: 14, Padding: 16},
		Nodes: []sy.ArtifactNode{
			{
				ID:        "headline",
				Component: "markdown",
				Props: map[string]any{
					"text": "## Incident room\nCheckout latency is elevated. The artifact should bias toward decisions, owners, and mitigation confidence.",
				},
				Layout: sy.ArtifactLayoutItem{ColumnSpan: 2},
			},
			{
				ID:        "severity",
				Component: "metric",
				Props: map[string]any{
					"label": "Severity",
					"value": "SEV-2",
					"delta": "mitigating",
				},
			},
			{
				ID:        "confidence",
				Component: "progress",
				Props: map[string]any{
					"value": 0.43,
					"text":  "Mitigation confidence",
				},
			},
			{
				ID:        "owners",
				Component: "table",
				Props: map[string]any{
					"headers": []any{"Workstream", "Owner", "Status"},
					"rows": []any{
						[]any{"Traffic shaping", "Mina", "Running"},
						[]any{"DB telemetry", "Chris", "Watching"},
						[]any{"Rollback window", "Dana", "Prepared"},
					},
				},
				Layout: sy.ArtifactLayoutItem{ColumnSpan: 2},
			},
		},
	}
}

func incidentNotesSpec() sy.ArtifactSpec {
	return singleNoteSpec("### Agent note\nThis view is intentionally simpler. During an incident, fewer components with sharper ownership usually read better than a dense dashboard.")
}

func customMainSpec(headline, label, value string, progress float64) sy.ArtifactSpec {
	return sy.ArtifactSpec{
		Version: "v1",
		Layout:  sy.ArtifactLayout{Columns: 2, Gap: 14, Padding: 16},
		Nodes: []sy.ArtifactNode{
			{
				ID:        "headline",
				Component: "markdown",
				Props: map[string]any{
					"text": headline,
				},
				Layout: sy.ArtifactLayoutItem{ColumnSpan: 2},
			},
			{
				ID:        "metric",
				Component: "metric",
				Props: map[string]any{
					"label": label,
					"value": value,
				},
			},
			{
				ID:        "progress",
				Component: "progress",
				Props: map[string]any{
					"value": progress,
					"text":  "Local update confidence",
				},
			},
		},
	}
}

func customNotesSpec(note string) sy.ArtifactSpec {
	return singleNoteSpec("### Agent note\n" + note)
}

func singleNoteSpec(text string) sy.ArtifactSpec {
	return sy.ArtifactSpec{
		Version: "v1",
		Layout:  sy.ArtifactLayout{Columns: 1, Gap: 12, Padding: 16},
		Nodes: []sy.ArtifactNode{
			{
				ID:        "note",
				Component: "markdown",
				Props: map[string]any{
					"text": text,
				},
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
