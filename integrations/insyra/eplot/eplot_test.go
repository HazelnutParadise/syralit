package syiplot

import (
	"strings"
	"testing"

	sy "github.com/HazelnutParadise/syralit"

	"github.com/HazelnutParadise/insyra"
)

func TestWordCloudRenders(t *testing.T) {
	dl := insyra.NewDataList(10.0, 8, 6, 4, 2).SetName("weights")
	tree := sy.RenderOnce(func() { WordCloud(dl, "Tags") })

	comps := tree.Find("component")
	if len(comps) != 1 {
		t.Fatalf("expected 1 component, got %d", len(comps))
	}
	html, _ := comps[0].Props["html"].(string)
	if !strings.Contains(html, "echarts") {
		t.Fatalf("expected echarts markup in component HTML (%d bytes)", len(html))
	}
}

func TestEChartNil(t *testing.T) {
	tree := sy.RenderOnce(func() { EChart(nil) })
	if n := len(tree.Find("component")); n != 0 {
		t.Fatalf("a nil chart must not render a component, got %d", n)
	}
}

func wcHTML(t *testing.T) string {
	t.Helper()
	dl := insyra.NewDataList(5.0, 3, 2, 4).SetName("weights")
	tree := sy.RenderOnce(func() { WordCloud(dl, "Tags") })
	comps := tree.Find("component")
	if len(comps) != 1 {
		t.Fatalf("expected 1 component, got %d", len(comps))
	}
	html, _ := comps[0].Props["html"].(string)
	return html
}

func TestOfflineInlinesAssets(t *testing.T) {
	SetOffline(true)
	defer SetOffline(false)

	html := wcHTML(t)
	if strings.Contains(html, `src="https://go-echarts`) {
		t.Fatal("offline mode must not reference the go-echarts CDN")
	}
	// All three bundled scripts (echarts, echarts@4, wordcloud) inlined → large.
	if len(html) < 1_000_000 {
		t.Fatalf("expected inlined assets to enlarge the HTML, got %d bytes", len(html))
	}
}

func TestOnlineKeepsCDN(t *testing.T) {
	SetOffline(false)
	html := wcHTML(t)
	if !strings.Contains(html, "go-echarts.github.io") {
		t.Fatal("online mode should reference the CDN")
	}
}
