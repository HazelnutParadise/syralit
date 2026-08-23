package syralit

import (
	"os"
	"path/filepath"
	"testing"
)

// ResolveConfig is the embedding path's way to read what syralit.toml
// configured (issue #4): Handler ignores Host/Port because the caller owns the
// listener, so the caller must be able to resolve the same Config itself.
func TestResolveConfig(t *testing.T) {
	t.Run("defaults without syralit.toml", func(t *testing.T) {
		t.Chdir(t.TempDir())
		got := ResolveConfig(Config{})
		if got.Host != "127.0.0.1" || got.Port != 8600 || got.Title != "Syralit App" {
			t.Fatalf("got host=%q port=%d title=%q", got.Host, got.Port, got.Title)
		}
	})

	t.Run("syralit.toml then defaults, code wins", func(t *testing.T) {
		dir := t.TempDir()
		toml := "title = \"From File\"\nhost = \"0.0.0.0\"\nport = 9100\n"
		if err := os.WriteFile(filepath.Join(dir, ConfigFileName), []byte(toml), 0o644); err != nil {
			t.Fatal(err)
		}
		t.Chdir(dir)
		in := Config{Title: "From Code"}
		got := ResolveConfig(in)
		if got.Host != "0.0.0.0" || got.Port != 9100 {
			t.Fatalf("file values not applied: host=%q port=%d", got.Host, got.Port)
		}
		if got.Title != "From Code" {
			t.Fatalf("code value overridden: title=%q", got.Title)
		}
		if in.Host != "" || in.Port != 0 {
			t.Fatalf("input mutated: %+v", in)
		}
	})
}
