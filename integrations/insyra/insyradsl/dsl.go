// Package insyradsl runs the Insyra CLI DSL from inside Syralit apps, turning
// dynamic .isr computations into rendered widgets and Artifact Canvas nodes.
//
// It is intentionally a separate package from the lightweight
// integrations/insyra helpers because importing the Insyra DSL engine pulls in
// the full Insyra CLI dependency tree (cobra, database drivers, parquet/arrow,
// readline, ...). Depend on this package only when you actually want DSL-driven
// computation.
//
// Import with alias syidsl:
//
//	import syidsl "github.com/HazelnutParadise/syralit/integrations/insyra/insyradsl"
package insyradsl

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/HazelnutParadise/insyra/cli/env"
	"github.com/HazelnutParadise/insyra/engine/dsl"
)

// DSLResult is the outcome of running an Insyra DSL script via RunDSL.
type DSLResult struct {
	// Vars holds the variables the script produced, keyed by name. Values are
	// *insyra.DataTable, *insyra.DataList, or scalars — the same types the
	// Insyra CLI stores. Commands run without an "as <name>" clause (groupby,
	// filter, describe, pivot, fillna, ...) land under the "$result" key.
	Vars map[string]any
	// Output is the textual, REPL-style transcript the script printed — e.g.
	// from show, summary, or the scalar stats (mean, sum, ...), which print
	// their result rather than storing a variable.
	Output string
	// Err is the first execution error, if any. Vars and Output still reflect
	// whatever ran successfully before the failing line.
	Err error
}

const (
	// defaultDSLTimeout bounds a single RunDSL call by default.
	defaultDSLTimeout = 15 * time.Second
	// defaultDSLMaxLines caps command lines per script by default.
	defaultDSLMaxLines = 500
)

type dslOpts struct {
	vars         map[string]any
	timeout      time.Duration
	unrestricted bool
	maxLines     int
	envRoot      string
}

// DSLOption configures RunDSL.
type DSLOption func(*dslOpts)

// WithVars injects variables into the DSL session before the script runs, so a
// script can compute over in-memory data without loading files. Values should
// be *insyra.DataTable, *insyra.DataList, or JSON-serializable scalars.
func WithVars(vars map[string]any) DSLOption { return func(o *dslOpts) { o.vars = vars } }

// DSLTimeout bounds how long the script may run. Zero disables the limit. On
// timeout RunDSL returns an error; the abandoned session still cleans up its
// temporary environment.
func DSLTimeout(d time.Duration) DSLOption { return func(o *dslOpts) { o.timeout = d } }

// MaxLines caps how many command lines a script may contain (blank and comment
// lines excluded). Zero uses the default. It guards against oversized
// agent-supplied scripts.
func MaxLines(n int) DSLOption { return func(o *dslOpts) { o.maxLines = n } }

// Unrestricted disables the safe-mode allowlist, permitting the full Insyra DSL
// including file, database, and network commands (load, save, db, fetch, run,
// ...). Use it ONLY for trusted, app-author-written scripts — never for
// agent-supplied input.
func Unrestricted() DSLOption { return func(o *dslOpts) { o.unrestricted = true } }

// EnvRoot runs the script in a persistent environment rooted at path instead of
// the default per-call temporary directory. The environment
// (path/envs/default/) is created if missing and is NOT deleted afterwards, so
// variables persist across calls — Insyra restores them from state.json on the
// next run. Use it for stateful or debuggable sessions.
//
// Unlike the default temp directory, a shared EnvRoot is not isolated between
// concurrent RunDSL calls: they read and write the same state.json. Give each
// logical session its own root if you run them concurrently.
func EnvRoot(path string) DSLOption { return func(o *dslOpts) { o.envRoot = path } }

// safeDSLCommands is the default-deny allowlist of pure, in-memory computation
// commands. It deliberately excludes everything that touches the file system, a
// database, or the network (load, read, save, convert, db, fetch, run) and
// everything that mutates CLI or session state outside the ephemeral
// environment (env, config, history, clear, exit, help). It also excludes plot,
// which writes image files — charts are rendered by Syralit instead.
var safeDSLCommands = map[string]struct{}{
	// inspection (read-only / in-memory)
	"show": {}, "summary": {}, "describe": {}, "shape": {}, "types": {}, "vars": {}, "version": {},
	// data creation (in-memory)
	"newdl": {}, "newdt": {},
	// structure & access
	"addcol": {}, "addrow": {}, "dropcol": {}, "droprow": {}, "swap": {}, "transpose": {},
	"rows": {}, "cols": {}, "row": {}, "col": {}, "get": {}, "set": {},
	"setrownames": {}, "setcolnames": {}, "rename": {}, "clone": {}, "drop": {},
	// processing
	"filter": {}, "sort": {}, "sample": {}, "split": {}, "find": {}, "replace": {}, "clean": {},
	"merge": {}, "groupby": {}, "pivot": {}, "unpivot": {}, "encode": {}, "scale": {},
	"ccl": {}, "addcolccl": {}, "fillna": {}, "fillnan": {},
	// statistics
	"sum": {}, "mean": {}, "median": {}, "mode": {}, "stdev": {}, "var": {}, "min": {}, "max": {},
	"range": {}, "quartile": {}, "iqr": {}, "percentile": {}, "count": {}, "counter": {},
	"corr": {}, "cov": {}, "corrmatrix": {}, "skewness": {}, "kurtosis": {},
	// transform / time-series
	"rank": {}, "normalize": {}, "standardize": {}, "reverse": {}, "upper": {}, "lower": {},
	"capitalize": {}, "parsenums": {}, "parsestrings": {}, "movavg": {}, "expsmooth": {},
	"diff": {}, "diffn": {}, "shift": {}, "pctchange": {}, "cumsum": {}, "cumprod": {},
	"cummax": {}, "cummin": {}, "rolling": {}, "expanding": {},
	// modeling / inference
	"regression": {}, "pca": {}, "kmeans": {}, "hclust": {}, "cutree": {}, "dbscan": {},
	"silhouette": {}, "knn_classify": {}, "knn_regress": {}, "knn_neighbors": {},
	"ttest": {}, "ztest": {}, "anova": {}, "ftest": {}, "chisq": {},
}

// RunDSL executes an Insyra DSL (.isr) script and returns the variables it
// produced plus its textual output. Each call runs in an isolated, ephemeral
// environment under a temporary directory that is removed afterwards, so runs
// never touch or share the user's ~/.insyra state and are safe to call
// concurrently.
//
// By default RunDSL runs in safe mode: only pure, in-memory computation
// commands are allowed (see safeDSLCommands); file, database, and network
// commands are rejected before execution. Pass Unrestricted to lift that limit
// for trusted scripts.
func RunDSL(script string, opts ...DSLOption) DSLResult {
	o := dslOpts{timeout: defaultDSLTimeout, maxLines: defaultDSLMaxLines}
	for _, opt := range opts {
		opt(&o)
	}

	lines, err := prepareDSLLines(script, o)
	if err != nil {
		return DSLResult{Err: err}
	}
	if len(lines) == 0 {
		return DSLResult{Vars: map[string]any{}}
	}

	run := func() DSLResult { return runDSLLines(lines, o) }

	if o.timeout <= 0 {
		return run()
	}
	resCh := make(chan DSLResult, 1)
	go func() { resCh <- run() }()
	select {
	case res := <-resCh:
		return res
	case <-time.After(o.timeout):
		return DSLResult{Err: fmt.Errorf("insyra dsl: timed out after %s", o.timeout)}
	}
}

// prepareDSLLines strips blanks and comments, enforces the safe-mode allowlist
// (unless unrestricted), and caps the number of command lines.
func prepareDSLLines(script string, o dslOpts) ([]string, error) {
	var lines []string
	for _, raw := range strings.Split(script, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if !o.unrestricted {
			cmd := strings.Fields(line)[0]
			if _, ok := safeDSLCommands[cmd]; !ok {
				return nil, fmt.Errorf("insyra dsl: command %q is not allowed in safe mode", cmd)
			}
		}
		lines = append(lines, line)
	}
	maxLines := o.maxLines
	if maxLines <= 0 {
		maxLines = defaultDSLMaxLines
	}
	if len(lines) > maxLines {
		return nil, fmt.Errorf("insyra dsl: script has %d command lines, exceeds limit of %d", len(lines), maxLines)
	}
	return lines, nil
}

// runDSLLines executes the prepared lines and returns the resulting variables
// and captured output. By default it uses a throwaway temp environment that is
// removed afterwards; if o.envRoot is set, it uses that persistent root instead
// and leaves it in place.
func runDSLLines(lines []string, o dslOpts) DSLResult {
	root := o.envRoot
	if root == "" {
		tmp, err := os.MkdirTemp("", "syralit-dsl-")
		if err != nil {
			return DSLResult{Err: fmt.Errorf("insyra dsl: create temp env: %w", err)}
		}
		defer func() { _ = os.RemoveAll(tmp) }()
		root = tmp
	}

	var buf bytes.Buffer
	mgr := env.NewManager(root, "")
	session, err := dsl.NewSession(mgr, "", &buf)
	if err != nil {
		return DSLResult{Err: fmt.Errorf("insyra dsl: new session: %w", err), Output: buf.String()}
	}

	ctx := session.Context()
	for name, v := range o.vars {
		ctx.Vars[name] = v
	}

	var runErr error
	for _, line := range lines {
		if execErr := session.Execute(line); execErr != nil {
			runErr = fmt.Errorf("insyra dsl: %q: %w", line, execErr)
			break
		}
	}

	vars := make(map[string]any, len(ctx.Vars))
	for k, v := range ctx.Vars {
		vars[k] = v
	}
	return DSLResult{Vars: vars, Output: buf.String(), Err: runErr}
}
