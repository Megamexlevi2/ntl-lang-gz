package manifest

import (
	"fmt"
	"os"
	"path/filepath"

	"lunex/internal/toml"
)

const (
	ManifestFile = "lunex.toml"
	LockFile     = "lunex.lock"
)

type Library struct {
	Name    string
	Source  string
	URL     string
	Path    string
	Version string
	Release string
}

type Project struct {
	Name        string
	Version     string
	Description string
	License     string
	Repository  string
	Entry       string

	Bin      map[string]string
	binOrder []string

	MinVersion string
	MaxVersion string

	Libraries map[string]*Library
	libOrder  []string
}

func NewProject(name string) *Project {
	return &Project{
		Name:      name,
		Version:   "0.1.0",
		Entry:     "main.lx",
		Libraries: make(map[string]*Library),
	}
}

func (p *Project) AddLibrary(lib *Library) {
	if p.Libraries == nil {
		p.Libraries = make(map[string]*Library)
	}
	if _, exists := p.Libraries[lib.Name]; !exists {
		p.libOrder = append(p.libOrder, lib.Name)
	}
	p.Libraries[lib.Name] = lib
}

func (p *Project) OrderedLibraries() []*Library {
	out := make([]*Library, 0, len(p.libOrder))
	for _, name := range p.libOrder {
		if lib, ok := p.Libraries[name]; ok {
			out = append(out, lib)
		}
	}
	return out
}

func (p *Project) AddBin(command, entry string) {
	if p.Bin == nil {
		p.Bin = make(map[string]string)
	}
	if _, exists := p.Bin[command]; !exists {
		p.binOrder = append(p.binOrder, command)
	}
	p.Bin[command] = entry
}

func (p *Project) OrderedBin() []string {
	return append([]string(nil), p.binOrder...)
}

func Path(p string) string {
	info, err := os.Stat(p)
	if err == nil && info.IsDir() {
		return filepath.Join(p, ManifestFile)
	}
	return p
}

func Find() (string, bool) {
	if _, err := os.Stat(ManifestFile); err == nil {
		return ManifestFile, true
	}
	return "", false
}

func Load(p string) (*Project, error) {
	p = Path(p)
	data, err := os.ReadFile(p)
	if err != nil {
		return nil, err
	}
	doc, err := toml.Parse(string(data))
	if err != nil {
		return nil, fmt.Errorf("parsing %s: %w", p, err)
	}

	proj := &Project{Libraries: make(map[string]*Library)}

	if projTable, ok := doc.Root.GetTable("project"); ok {
		proj.Name = projTable.GetString("name", "")
		proj.Version = projTable.GetString("version", "0.1.0")
		proj.Description = projTable.GetString("description", "")
		proj.License = projTable.GetString("license", "")
		proj.Repository = projTable.GetString("repository", "")
		proj.Entry = projTable.GetString("entry", "main.lx")

		if binTable, ok := projTable.GetTable("bin"); ok {
			for _, cmd := range binTable.Keys() {
				if entry := binTable.GetString(cmd, ""); entry != "" {
					proj.AddBin(cmd, entry)
				}
			}
		} else if binStr := projTable.GetString("bin", ""); binStr != "" {
			cmdName := proj.Name
			if cmdName == "" {
				cmdName = "cli"
			}
			proj.AddBin(cmdName, binStr)
		}
	}
	if proj.Entry == "" {
		proj.Entry = "main.lx"
	}

	if lunexTable, ok := doc.Root.GetTable("lunex"); ok {
		proj.MinVersion = lunexTable.GetString("min_version", "")
		proj.MaxVersion = lunexTable.GetString("max_version", "")
	}

	if libsTable, ok := doc.Root.GetTable("libraries"); ok {
		for _, name := range libsTable.Keys() {
			sub, ok := libsTable.GetTable(name)
			if !ok {
				continue
			}
			lib := &Library{
				Name:    name,
				Source:  sub.GetString("source", "github"),
				URL:     sub.GetString("url", ""),
				Path:    sub.GetString("path", ""),
				Version: sub.GetString("version", ""),
				Release: sub.GetString("release", ""),
			}
			if lib.Source == "" {
				lib.Source = "github"
			}
			proj.AddLibrary(lib)
		}
	}

	return proj, nil
}

func Save(p string, proj *Project) error {
	p = Path(p)
	if dir := filepath.Dir(p); dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	doc := toml.NewDocument()

	projTable := doc.Root.SubTable("project")
	projTable.Set("name", proj.Name)
	projTable.Set("version", proj.Version)
	if proj.Description != "" {
		projTable.Set("description", proj.Description)
	}
	if proj.License != "" {
		projTable.Set("license", proj.License)
	}
	if proj.Repository != "" {
		projTable.Set("repository", proj.Repository)
	}
	entry := proj.Entry
	if entry == "" {
		entry = "main.lx"
	}
	projTable.Set("entry", entry)

	if len(proj.binOrder) == 1 && proj.binOrder[0] == proj.Name {
		projTable.Set("bin", proj.Bin[proj.binOrder[0]])
	} else if len(proj.binOrder) > 0 {
		binTable := projTable.SubTable("bin")
		for _, cmd := range proj.binOrder {
			if entryPath := proj.Bin[cmd]; entryPath != "" {
				binTable.Set(cmd, entryPath)
			}
		}
	}

	lunexTable := doc.Root.SubTable("lunex")
	if proj.MinVersion != "" {
		lunexTable.Set("min_version", proj.MinVersion)
	}
	if proj.MaxVersion != "" {
		lunexTable.Set("max_version", proj.MaxVersion)
	}

	if len(proj.libOrder) > 0 {
		libsTable := doc.Root.SubTable("libraries")
		for _, name := range proj.libOrder {
			lib := proj.Libraries[name]
			if lib == nil {
				continue
			}
			sub := libsTable.SubTable(name)
			if lib.Source != "" && lib.Source != "github" {
				sub.Set("source", lib.Source)
			}
			if lib.URL != "" {
				sub.Set("url", lib.URL)
			}
			if lib.Path != "" {
				sub.Set("path", lib.Path)
			}
			if lib.Version != "" {
				sub.Set("version", lib.Version)
			}
			if lib.Release != "" {
				sub.Set("release", lib.Release)
			}
		}
	}

	return os.WriteFile(p, []byte(toml.Write(doc)), 0644)
}

type LockedModule struct {
	Name    string
	Version string
	Hash    string
	Source  string
	Commit  string
	URL     string
}

type Lock struct {
	Modules map[string]*LockedModule
	order   []string
}

func NewLock() *Lock {
	return &Lock{Modules: make(map[string]*LockedModule)}
}

func (l *Lock) Set(m *LockedModule) {
	if l.Modules == nil {
		l.Modules = make(map[string]*LockedModule)
	}
	if _, exists := l.Modules[m.Name]; !exists {
		l.order = append(l.order, m.Name)
	}
	l.Modules[m.Name] = m
}

func (l *Lock) Ordered() []*LockedModule {
	out := make([]*LockedModule, 0, len(l.order))
	for _, name := range l.order {
		if m, ok := l.Modules[name]; ok {
			out = append(out, m)
		}
	}
	return out
}

func LockPath(dir string) string {
	info, err := os.Stat(dir)
	if err == nil && info.IsDir() {
		return filepath.Join(dir, LockFile)
	}
	return dir
}

func LoadLock(dir string) (*Lock, error) {
	p := LockPath(dir)
	data, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return NewLock(), nil
		}
		return nil, err
	}
	doc, err := toml.Parse(string(data))
	if err != nil {
		return nil, fmt.Errorf("parsing %s: %w", p, err)
	}
	lock := NewLock()
	if modsTable, ok := doc.Root.GetTable("modules"); ok {
		for _, name := range modsTable.Keys() {
			sub, ok := modsTable.GetTable(name)
			if !ok {
				continue
			}
			lock.Set(&LockedModule{
				Name:    name,
				Version: sub.GetString("version", ""),
				Hash:    sub.GetString("hash", ""),
				Source:  sub.GetString("source", ""),
				Commit:  sub.GetString("commit", ""),
				URL:     sub.GetString("url", ""),
			})
		}
	}
	return lock, nil
}

func SaveLock(dir string, lock *Lock) error {
	p := LockPath(dir)
	doc := toml.NewDocument()
	modsTable := doc.Root.SubTable("modules")
	for _, name := range lock.order {
		m := lock.Modules[name]
		if m == nil {
			continue
		}
		sub := modsTable.SubTable(name)
		sub.Set("version", m.Version)
		if m.Hash != "" {
			sub.Set("hash", m.Hash)
		}
		if m.Source != "" {
			sub.Set("source", m.Source)
		}
		if m.Commit != "" {
			sub.Set("commit", m.Commit)
		}
		if m.URL != "" {
			sub.Set("url", m.URL)
		}
	}
	return os.WriteFile(p, []byte(toml.Write(doc)), 0644)
}
