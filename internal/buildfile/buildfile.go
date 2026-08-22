package buildfile

import (
	"os"
	"path/filepath"

	"lunex/internal/manifest"
	"lunex/internal/meta"
)

type Config struct {
	Name       string
	Version    string
	Entry      string
	Output     string
	MinVersion string
	MaxVersion string
}

func DefaultConfig() Config {
	return Config{
		Name:    "app",
		Version: meta.Version(),
		Entry:   "main.lx",
		Output:  ".",
	}
}

func Find() (string, bool) {
	return manifest.Find()
}

func Parse(path string) (Config, error) {
	cfg := DefaultConfig()
	proj, err := manifest.Load(path)
	if err != nil {
		return cfg, err
	}
	if proj == nil {
		return cfg, nil
	}
	if proj.Name != "" {
		cfg.Name = proj.Name
	}
	if proj.Version != "" {
		cfg.Version = proj.Version
	}
	if proj.Entry != "" {
		cfg.Entry = proj.Entry
	}
	cfg.MinVersion = proj.MinVersion
	cfg.MaxVersion = proj.MaxVersion
	return cfg, nil
}

func Generate(path string, name string) error {
	proj := manifest.NewProject(name)
	proj.Version = "0.1.0"
	proj.Description = "A Lunex project"
	proj.MinVersion = meta.Version()
	if dir := filepath.Dir(path); dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}
	return manifest.Save(path, proj)
}
