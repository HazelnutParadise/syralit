// Command syralit is the Syralit development CLI.
//
//	syralit dev [dir]    start the hot reload supervisor (default dir ".")
//	syralit run [dir]    build and run once, no watching
//
// In dev mode the supervisor owns the outward port for its whole lifetime and
// restarts the app process on changes, so the app never fully stops and session
// state is preserved across reloads.
package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	sy "github.com/HazelnutParadise/syralit"
)

func isDir(path string) bool {
	fi, err := os.Stat(path)
	return err == nil && fi.IsDir()
}

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	cmd := os.Args[1]
	args := os.Args[2:]

	switch cmd {
	case "dev":
		runDev(args)
	case "run":
		runOnce(args)
	case "build":
		runBuild(args)
	case "new":
		runNew(args)
	case "-h", "--help", "help":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "syralit: unknown command %q\n\n", cmd)
		usage()
		os.Exit(2)
	}
}

func runDev(args []string) {
	// Flags default to zero so precedence is flag > syralit.toml > built-in default.
	fs := flag.NewFlagSet("dev", flag.ExitOnError)
	host := fs.String("host", "", "outward bind host (default 127.0.0.1)")
	port := fs.Int("port", 0, "outward port (default 8600)")
	title := fs.String("title", "", "page title")
	assets := fs.String("assets", "", "override: serve front-end assets from this dir")
	_ = fs.Parse(args)

	dir := "."
	if fs.NArg() > 0 {
		dir = fs.Arg(0)
	}

	// Convention over configuration: if the project has an assets/ directory,
	// serve & hot-reload it automatically. No flag needed for the common case.
	assetsDir := *assets
	if assetsDir == "" {
		if cand := filepath.Join(dir, "assets"); isDir(cand) {
			assetsDir = cand
		}
	}

	// Likewise a public/ directory is served at the site root automatically.
	publicDir := ""
	if cand := filepath.Join(dir, "public"); isDir(cand) {
		publicDir = cand
	}

	err := sy.RunDev(sy.DevOptions{
		Dir:       dir,
		Host:      *host,
		Port:      *port,
		Title:     *title,
		AssetsDir: assetsDir,
		PublicDir: publicDir,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "syralit dev: %v\n", err)
		os.Exit(1)
	}
}

// runOnce builds and runs the target package without watching. It simply shells
// out to `go run`, so the app binds the port itself (no supervisor).
func runOnce(args []string) {
	dir := "."
	if len(args) > 0 {
		dir = args[0]
	}
	c := exec.Command("go", "run", dir)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	c.Stdin = os.Stdin
	if err := c.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "syralit run: %v\n", err)
		os.Exit(1)
	}
}

// runBuild compiles the app into a single self-contained executable. The
// framework's front-end is already embedded in the syralit package, so the only
// extra work is folding the project's own static files (public/ and any assets/
// overrides) into the binary: we generate a temporary file with //go:embed
// directives for whichever of those dirs exist, build, then remove it. The
// result needs no runtime files alongside it.
func runBuild(args []string) {
	fs := flag.NewFlagSet("build", flag.ExitOnError)
	out := fs.String("o", "", "output binary path (default: <dir-name>[.exe])")
	_ = fs.Parse(args)

	dir := "."
	if fs.NArg() > 0 {
		dir = fs.Arg(0)
	}
	if !isDir(dir) {
		fmt.Fprintf(os.Stderr, "syralit build: %q is not a directory\n", dir)
		os.Exit(1)
	}

	// Resolve a friendly default output name from the directory.
	output := *out
	if output == "" {
		abs, err := filepath.Abs(dir)
		if err != nil {
			abs = dir
		}
		output = filepath.Base(abs)
		if output == "." || output == string(filepath.Separator) || output == "" {
			output = "app"
		}
		if runtime.GOOS == "windows" {
			output += ".exe"
		}
	}
	// go build resolves -o relative to its working dir (the project), so make a
	// relative output path absolute against the caller's cwd first.
	if !filepath.IsAbs(output) {
		if abs, err := filepath.Abs(output); err == nil {
			output = abs
		}
	}

	// Generate embed wiring for whichever static dirs carry files.
	var embeds []embedSpec
	if dirHasFiles(filepath.Join(dir, "public")) {
		embeds = append(embeds, embedSpec{Dir: "public", Var: "_syralitPublic", Register: "Static"})
	}
	if dirHasFiles(filepath.Join(dir, "assets")) {
		embeds = append(embeds, embedSpec{Dir: "assets", Var: "_syralitAssets", Register: "StaticAssets"})
	}

	// Note: the name must NOT start with "_" or "." — the Go toolchain ignores
	// such files, so go:embed directives in them would never compile in.
	genPath := filepath.Join(dir, "syralit_embed_gen.go")
	if len(embeds) > 0 {
		if err := os.WriteFile(genPath, []byte(genEmbedFile(embeds)), 0o644); err != nil {
			fmt.Fprintf(os.Stderr, "syralit build: %v\n", err)
			os.Exit(1)
		}
		defer os.Remove(genPath)
	}

	c := exec.Command("go", "build", "-o", output, ".")
	c.Dir = dir
	c.Stdout, c.Stderr = os.Stdout, os.Stderr
	c.Env = os.Environ()
	if err := c.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "syralit build: %v\n", err)
		os.Exit(1)
	}

	bundled := []string{"front-end + backend"}
	for _, e := range embeds {
		bundled = append(bundled, e.Dir+"/")
	}
	fmt.Printf("built %s (%s) — single self-contained executable\n", output, strings.Join(bundled, ", "))
}

type embedSpec struct {
	Dir      string // directory to embed (relative to project root)
	Var      string // generated embed.FS variable name
	Register string // syralit registration func: "Static" or "StaticAssets"
}

// genEmbedFile renders the temporary //go:embed file. "all:" includes files that
// start with "." or "_", so nothing in the static dirs is silently dropped.
func genEmbedFile(embeds []embedSpec) string {
	var b strings.Builder
	b.WriteString("// Code generated by `syralit build`. DO NOT EDIT.\n")
	b.WriteString("package main\n\n")
	b.WriteString("import (\n\t\"embed\"\n\t\"io/fs\"\n\n\tsy \"github.com/HazelnutParadise/syralit\"\n)\n\n")
	for _, e := range embeds {
		fmt.Fprintf(&b, "//go:embed all:%s\nvar %s embed.FS\n\n", e.Dir, e.Var)
	}
	b.WriteString("func init() {\n")
	for _, e := range embeds {
		fmt.Fprintf(&b, "\tif sub, err := fs.Sub(%s, %q); err == nil {\n\t\tsy.%s(sub)\n\t}\n", e.Var, e.Dir, e.Register)
	}
	b.WriteString("}\n")
	return b.String()
}

// dirHasFiles reports whether dir exists and contains at least one regular file
// (recursively). go:embed of an empty directory is a compile error, so we skip
// generating a directive for dirs with nothing to embed.
func dirHasFiles(dir string) bool {
	if !isDir(dir) {
		return false
	}
	found := false
	_ = filepath.WalkDir(dir, func(_ string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if !d.IsDir() {
			found = true
			return filepath.SkipAll
		}
		return nil
	})
	return found
}

// runNew scaffolds a conventional Syralit project so the user gets a fixed,
// ready-to-run layout instead of wiring flags by hand.
//
//	syralit new myapp   scaffold into a new myapp/ directory
//	syralit new .       scaffold into the current directory (no wrapper folder)
func runNew(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: syralit new <name>|.")
		os.Exit(2)
	}
	arg := args[0]

	// "." (or "./") scaffolds in place: target the current directory and take
	// the module name from its base, instead of creating a wrapper folder.
	inPlace := arg == "." || arg == "./"
	targetDir := arg
	name := arg
	if inPlace {
		targetDir = "."
		abs, err := filepath.Abs(".")
		if err != nil {
			fmt.Fprintf(os.Stderr, "syralit new: %v\n", err)
			os.Exit(1)
		}
		name = filepath.Base(abs)
		// Don't clobber an existing project.
		for _, f := range []string{"main.go", "go.mod"} {
			if _, err := os.Stat(filepath.Join(targetDir, f)); err == nil {
				fmt.Fprintf(os.Stderr, "syralit new: %s already exists here; refusing to overwrite\n", f)
				os.Exit(1)
			}
		}
	} else {
		if isDir(arg) {
			fmt.Fprintf(os.Stderr, "syralit new: %q already exists\n", arg)
			os.Exit(1)
		}
		if err := os.MkdirAll(arg, 0o755); err != nil {
			fmt.Fprintf(os.Stderr, "syralit new: %v\n", err)
			os.Exit(1)
		}
	}

	if err := os.MkdirAll(filepath.Join(targetDir, "pages"), 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "syralit new: %v\n", err)
		os.Exit(1)
	}

	files := map[string]string{
		"main.go":       scaffoldMain(name),
		"pages/home.go": scaffoldPageHome(name),
		"syralit.toml":  scaffoldToml(name),
		"README.md":     scaffoldReadme(name),
		".gitignore":    "/" + name + "\n*.exe\n_syralit_pages.go\nsyralit_embed_gen.go\n",
	}
	for fname, content := range files {
		if err := os.WriteFile(filepath.Join(targetDir, fname), []byte(content), 0o644); err != nil {
			fmt.Fprintf(os.Stderr, "syralit new: %v\n", err)
			os.Exit(1)
		}
	}

	// go mod init so the project is a module from the start.
	c := exec.Command("go", "mod", "init", name)
	c.Dir = targetDir
	c.Stdout, c.Stderr = os.Stdout, os.Stderr
	_ = c.Run()

	if inPlace {
		fmt.Printf(`created Syralit project in current directory

  go mod tidy
  syralit dev

`)
		return
	}
	fmt.Printf(`created %s/

  cd %s
  go mod tidy
  syralit dev

`, name, name)
}

func scaffoldMain(name string) string {
	return `package main

import (
	_ "` + name + `/pages"

	sy "github.com/HazelnutParadise/syralit"
)

func main() {
	sy.App(nil)
}
`
}

func scaffoldPageHome(name string) string {
	return `package pages

import sy "github.com/HazelnutParadise/syralit"

func init() {
	sy.AddPage("Home", homePage, sy.PageIcon("🏠"), sy.PageOrder(1))
}

func homePage() {
	sy.Title("` + name + `")

	name := sy.TextInput("Your name", sy.Key("name"))
	if name != "" {
		sy.Text("Hello, " + name + "!")
	}
}
`
}

func scaffoldToml(name string) string {
	return `# Syralit project settings (optional). Run "syralit dev" with no flags.
title = "` + name + `"
port = 8600

[theme]
mode = "system"
accent = "#7C3AED"
radius = "12px"
`
}

func scaffoldReadme(name string) string {
	return "# " + name + `

A [Syralit](https://github.com/HazelnutParadise/syralit) data app.

` + "```bash" + `
syralit dev       # hot reload, app never fully stops
syralit run       # run once, no watching
syralit build     # compile to a single self-contained executable
` + "```" + `

Project layout:

` + "```" + `
` + name + `/
  main.go          entry point
  pages/           each .go file = one page (sidebar auto-generated)
    home.go        default page
  syralit.toml     settings (port / title / theme)
  public/          optional: static files served at the site root
  assets/          optional: custom front-end overrides
` + "```" + `

` + "`syralit build`" + ` folds public/ and assets/ into the binary, so the
compiled app runs anywhere with no files alongside it.

Add a page: create a new .go file in pages/ with an init() that calls
sy.AddPage(). Save, and the page appears in the sidebar.
`
}

func usage() {
	fmt.Fprint(os.Stderr, `syralit — Go-native interactive data apps

usage:
  syralit new <name>           scaffold a new project in a new folder
  syralit new .                scaffold into the current directory (no wrapper)
  syralit dev [flags] [dir]    hot reload supervisor (app never fully stops)
  syralit run [dir]            build & run once
  syralit build [-o out] [dir] compile to one self-contained executable

Settings live in syralit.toml at the project root; "syralit dev" needs no flags.
Precedence: flag > syralit.toml > built-in default.

dev flags (override syralit.toml):
  -host string   outward bind host (default 127.0.0.1)
  -port int      outward port (default 8600)
  -title string  page title
  -assets dir    serve front-end assets from dir (auto-detected from ./assets)

build flags:
  -o string      output binary path (default: <dir-name>[.exe])

build folds the project's public/ (served at the site root) and any assets/
overrides into the binary, so the result runs with no files alongside it.
`)
}
