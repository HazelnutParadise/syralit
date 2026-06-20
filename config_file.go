package syralit

import (
	"log"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// ConfigFileName is the conventional per-project settings file. It is optional;
// when present at the project root it fills in any value not set explicitly in
// code or via CLI flags (precedence: flag > syralit.toml > built-in default).
const ConfigFileName = "syralit.toml"

// Theme controls the front-end look (SPEC §17).
type Theme struct {
	Mode   string // "light" | "dark" | "system"
	Accent string // CSS color, e.g. "#7C3AED"
	Radius string // CSS length, e.g. "12px"
}

// fileConfig mirrors syralit.toml.
type fileConfig struct {
	Title string `toml:"title"`
	Host  string `toml:"host"`
	Port  int    `toml:"port"`
	Theme struct {
		Mode   string `toml:"mode"`
		Accent string `toml:"accent"`
		Radius string `toml:"radius"`
	} `toml:"theme"`
}

// loadFileConfig reads dir/syralit.toml if present. A missing file is not an
// error (returns nil); a malformed file is logged and ignored.
func loadFileConfig(dir string) *fileConfig {
	path := filepath.Join(dir, ConfigFileName)
	if _, err := os.Stat(path); err != nil {
		return nil
	}
	var fc fileConfig
	if _, err := toml.DecodeFile(path, &fc); err != nil {
		log.Printf("syralit: ignoring %s: %v", path, err)
		return nil
	}
	return &fc
}

func (fc *fileConfig) applyToConfig(cfg *Config) {
	if fc == nil {
		return
	}
	if cfg.Title == "" {
		cfg.Title = fc.Title
	}
	if cfg.Host == "" {
		cfg.Host = fc.Host
	}
	if cfg.Port == 0 {
		cfg.Port = fc.Port
	}
	fc.applyTheme(&cfg.Theme)
}

func (fc *fileConfig) applyToDev(o *DevOptions) {
	if fc == nil {
		return
	}
	if o.Title == "" {
		o.Title = fc.Title
	}
	if o.Host == "" {
		o.Host = fc.Host
	}
	if o.Port == 0 {
		o.Port = fc.Port
	}
	fc.applyTheme(&o.Theme)
}

func (fc *fileConfig) applyTheme(t *Theme) {
	if t.Mode == "" {
		t.Mode = fc.Theme.Mode
	}
	if t.Accent == "" {
		t.Accent = fc.Theme.Accent
	}
	if t.Radius == "" {
		t.Radius = fc.Theme.Radius
	}
}
