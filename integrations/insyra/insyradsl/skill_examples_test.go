package insyradsl

import (
	"encoding/json"
	"testing"

	sy "github.com/HazelnutParadise/syralit"
)

// The exact JSON snippets shown in skills/syralit-artifact-dsl/SKILL.md.
func TestSkillExamplesAreValid(t *testing.T) {
	examples := map[string]string{
		"bar_chart": `{"id":"deals","component":"insyra","props":{"script":"newdl North South West as region\nnewdl 12 18 9 as deals\nnewdt region deals as t\nsetcolnames t region deals\ngroupby t by region agg deals:sum:total as report","render":"bar_chart","output":"report","x":"region","y":"total","title":"Deals by region"}}`,
		"filter":    `{"id":"top_regions","component":"insyra","props":{"script":"newdl North South West as region\nnewdl 120 45 90 as deals\nnewdt region deals as t\nsetcolnames t region deals\nfilter t \"['deals'] >= 90\" as top","render":"dataframe","output":"top"}}`,
	}
	wantType := map[string]string{"bar_chart": "bar_chart", "filter": "dataframe"}

	for name, js := range examples {
		var node sy.ArtifactNode
		if err := json.Unmarshal([]byte(js), &node); err != nil {
			t.Fatalf("[%s] invalid JSON: %v", name, err)
		}
		spec := sy.ArtifactSpec{Version: "v1", Nodes: []sy.ArtifactNode{node}}
		store := sy.NewArtifactStore("skill-"+name, spec)
		if err := store.Set(spec); err != nil {
			t.Fatalf("[%s] compile: %v", name, err)
		}
		root := sy.RenderOnce(func() { sy.ArtifactCanvas(store) })
		if got := len(root.Find(wantType[name])); got != 1 {
			t.Fatalf("[%s] want 1 %s node, got %d", name, wantType[name], got)
		}
		t.Logf("[%s] OK -> %s", name, wantType[name])
	}
}
