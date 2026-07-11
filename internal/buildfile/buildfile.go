package buildfile

import (
	"fmt"
	"os"
	"path/filepath"

	"lunex/internal/meta"
	"lunex/internal/pkg"
)

type Config struct {
	Name     string
	Version  string
	Entry    string
	Output   string
	Targets  []string
	Optimize bool
}

func DefaultConfig() Config {
	return Config{
		Name:     "app",
		Version:  meta.Version(),
		Entry:    "main.lx",
		Output:   ".",
		Optimize: true,
	}
}

func Find() (string, bool) {
	for _, name := range []string{"config.lx"} {
		if _, err := os.Stat(name); err == nil {
			return name, true
		}
	}
	return "", false
}

func Parse(path string) (Config, error) {
	cfg := DefaultConfig()
	m, err := pkg.LoadManifest(path)
	if err != nil {
		return cfg, err
	}
	if m == nil {
		return cfg, nil
	}
	if m.Name != "" {
		cfg.Name = m.Name
	}
	if m.Version != "" {
		cfg.Version = m.Version
	}
	if m.Entry != "" {
		cfg.Entry = m.Entry
	} else if m.Main != "" {
		cfg.Entry = m.Main
	}
	if m.Output != "" {
		cfg.Output = m.Output
	}
	cfg.Targets = append(cfg.Targets, m.Targets...)
	cfg.Optimize = m.Optimize
	return cfg, nil
}

func Generate(path string, name string) error {
	content := fmt.Sprintf(`
val project = {
  name: %q
  version: %q
  main: "main.lx"
  entry: "main.lx"
  output: "dist"
  optimize: true
  targets: []
  dependencies: {
  }
}

fn build() {
  project
}
`, name, meta.Version())

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0644)
}
