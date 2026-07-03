package insyradsl

import (
	"sort"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/cli/commands"
)

// CommandDoc is one safe DSL command's synopsis, pulled live from Insyra's
// command registry.
type CommandDoc struct {
	Name        string `json:"name"`
	Usage       string `json:"usage"`
	Description string `json:"description"`
}

// SafeCommandCatalog returns the safe-mode command vocabulary for the linked
// Insyra version: every allowlisted command that exists in Insyra's registry,
// with its live usage and description. It updates automatically when Insyra
// changes an existing command's synopsis, and drops commands that a new Insyra
// version removed — so the artifact skill never has to restate the vocabulary.
//
// It does NOT auto-expose commands a new Insyra version adds: the safe allowlist
// stays deliberately hand-curated so a future file/network command cannot be
// used the moment Insyra ships it.
func SafeCommandCatalog() []CommandDoc {
	out := make([]CommandDoc, 0, len(safeDSLCommands))
	for name := range safeDSLCommands {
		h, ok := commands.Registry[name]
		if !ok {
			continue
		}
		out = append(out, CommandDoc{Name: h.Name, Usage: h.Usage, Description: h.Description})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// insyraCapabilities is the discovery metadata for the "insyra" artifact
// component. The command list is live; the component-specific guidance (safe
// mode, inline data, positional columns, render kinds) is not in Insyra's own
// help, so it is stated here.
func insyraCapabilities() any {
	return map[string]any{
		"description":    "Run an Insyra DSL (.isr) script server-side in safe mode; render the result as a table, chart, metric, or text node.",
		"insyra_version": insyra.Version,
		"safe_mode":      true,
		"render_kinds":   []string{"dataframe", "table", "line_chart", "bar_chart", "area_chart", "pie_chart", "metric", "text"},
		"notes": []string{
			"Embed data inline with newdl/newdt; file, database, and network commands are rejected.",
			"Columns from newdt are positional — reference chart/pie columns by Excel letter (A, B) or name them first with setcolnames.",
			"'commands' below is the authoritative, version-accurate safe vocabulary; prefer it over any hardcoded list.",
		},
		"commands": SafeCommandCatalog(),
	}
}
