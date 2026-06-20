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

	err := sy.RunDev(sy.DevOptions{
		Dir:       dir,
		Host:      *host,
		Port:      *port,
		Title:     *title,
		AssetsDir: assetsDir,
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

// runNew scaffolds a conventional Syralit project so the user gets a fixed,
// ready-to-run layout instead of wiring flags by hand.
func runNew(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: syralit new <name>")
		os.Exit(2)
	}
	name := args[0]
	if isDir(name) {
		fmt.Fprintf(os.Stderr, "syralit new: %q already exists\n", name)
		os.Exit(1)
	}
	if err := os.MkdirAll(name, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "syralit new: %v\n", err)
		os.Exit(1)
	}

	pagesDir := filepath.Join(name, "pages")
	if err := os.MkdirAll(pagesDir, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "syralit new: %v\n", err)
		os.Exit(1)
	}

	files := map[string]string{
		"main.go":        scaffoldMain(name),
		"pages/home.go":  scaffoldPageHome(name),
		"syralit.toml":   scaffoldToml(name),
		"README.md":      scaffoldReadme(name),
		".gitignore":     "/" + name + "\n*.exe\n_syralit_pages.go\n",
	}
	for fname, content := range files {
		if err := os.WriteFile(filepath.Join(name, fname), []byte(content), 0o644); err != nil {
			fmt.Fprintf(os.Stderr, "syralit new: %v\n", err)
			os.Exit(1)
		}
	}

	// go mod init so the project is a module from the start.
	c := exec.Command("go", "mod", "init", name)
	c.Dir = name
	c.Stdout, c.Stderr = os.Stdout, os.Stderr
	_ = c.Run()

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
` + "```" + `

Project layout:

` + "```" + `
` + name + `/
  main.go          entry point
  pages/           each .go file = one page (sidebar auto-generated)
    home.go        default page
  syralit.toml     settings (port / title / theme)
  assets/          optional: custom front-end overrides
` + "```" + `

Add a page: create a new .go file in pages/ with an init() that calls
sy.AddPage(). Save, and the page appears in the sidebar.

> Syralit is pre-release and not yet tagged. Until it is, add a replace
> directive in go.mod pointing at your local checkout, e.g.
>
>     replace github.com/HazelnutParadise/syralit => ../syralit
`
}

func usage() {
	fmt.Fprint(os.Stderr, `syralit — Go-native interactive data apps

usage:
  syralit new <name>           scaffold a new project (fixed layout)
  syralit dev [flags] [dir]    hot reload supervisor (app never fully stops)
  syralit run [dir]            build & run once

Settings live in syralit.toml at the project root; "syralit dev" needs no flags.
Precedence: flag > syralit.toml > built-in default.

dev flags (override syralit.toml):
  -host string   outward bind host (default 127.0.0.1)
  -port int      outward port (default 8600)
  -title string  page title
  -assets dir    serve front-end assets from dir (auto-detected from ./assets)
`)
}
