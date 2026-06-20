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

// SwitchPage programmatically navigates to a different page. The current
// rerun is stopped and the next render will show the target page.
func SwitchPage(title string) {
	rc := current()
	rc.sess.setCurrentPage(title)
	Stop()
}

// Page represents a page definition for use with Navigation.
type Page struct {
	Title string
	Fn    func()
	Icon  string
}

// Navigation provides a declarative way to define multi-page apps inline.
// It registers pages, renders the active one, and returns the active page title.
// Unlike AddPage (which is called in init()), Navigation is called inside the
// app function and can be used with dynamic page lists.
//
//	active := sy.Navigation([]sy.Page{
//	    {Title: "Home", Fn: homePage, Icon: "🏠"},
//	    {Title: "Settings", Fn: settingsPage, Icon: "⚙️"},
//	})
func Navigation(pages []Page) string {
	rc := current()
	for i, p := range pages {
		found := false
		for _, existing := range pageRegistry {
			if existing.title == p.Title {
				found = true
				break
			}
		}
		if !found {
			entry := pageEntry{title: p.Title, fn: p.Fn, icon: p.Icon, order: i + 1}
			pageRegistry = append(pageRegistry, entry)
		}
	}

	active := rc.sess.activePage()
	found := false
	for _, p := range pages {
		if p.Title == active {
			found = true
			break
		}
	}
	if !found && len(pages) > 0 {
		active = pages[0].Title
		rc.sess.setCurrentPage(active)
	}

	fn := pageByTitle(active)
	if fn != nil {
		fn()
	}
	return active
}

// resetPages clears the registry (for tests only).
func resetPages() {
	pageRegistry = nil
	pageSeq = 0
}
