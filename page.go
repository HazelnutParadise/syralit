package syralit

import "sort"

// PageOption configures a page entry in the sidebar.
type PageOption func(*pageEntry)

type pageEntry struct {
	title string
	fn    func()
	icon  string
	order int
}

// PageIcon sets the emoji icon displayed next to the page title in the sidebar.
func PageIcon(icon string) PageOption {
	return func(p *pageEntry) { p.icon = icon }
}

// PageOrder sets an explicit sort position for this page. Lower values appear
// first. Pages without an explicit order use their registration sequence
// (which in Go is file-name-alphabetical within a package).
func PageOrder(n int) PageOption {
	return func(p *pageEntry) { p.order = n }
}

var (
	pageRegistry []pageEntry
	pageSeq      int
)

// AddPage registers a page for sidebar navigation. Call it in init() so that
// adding a new .go file with an init+AddPage call is all that is needed to
// create a page — the compiled-language equivalent of Streamlit's pages/.
//
// Pages are sorted by PageOrder (ascending), then by registration order.
func AddPage(title string, fn func(), opts ...PageOption) {
	pageSeq++
	p := pageEntry{
		title: title,
		fn:    fn,
		order: pageSeq * 1000, // spaced so manual PageOrder values can fit between
	}
	for _, o := range opts {
		o(&p)
	}
	pageRegistry = append(pageRegistry, p)
}

func hasPages() bool { return len(pageRegistry) > 0 }

func sortedPages() []pageEntry {
	sorted := make([]pageEntry, len(pageRegistry))
	copy(sorted, pageRegistry)
	sort.SliceStable(sorted, func(i, j int) bool {
		return sorted[i].order < sorted[j].order
	})
	return sorted
}

func pageByTitle(title string) func() {
	for _, p := range pageRegistry {
		if p.title == title {
			return p.fn
		}
	}
	return nil
}

func defaultPageTitle() string {
	sorted := sortedPages()
	if len(sorted) == 0 {
		return ""
	}
	return sorted[0].title
}

// pageInfos returns sidebar metadata for the browser client.
func pageInfos() []map[string]any {
	sorted := sortedPages()
	infos := make([]map[string]any, len(sorted))
	for i, p := range sorted {
		info := map[string]any{"title": p.title}
		if p.icon != "" {
			info["icon"] = p.icon
		}
		infos[i] = info
	}
	return infos
}

// resetPages clears the registry (for tests only).
func resetPages() {
	pageRegistry = nil
	pageSeq = 0
}
