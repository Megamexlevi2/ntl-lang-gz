package manifest

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"lunex/internal/compiler"
	"lunex/internal/meta"
	"lunex/internal/runtime"
)

type ManifestRepository struct {
	Type   string `json:"type"`
	URL    string `json:"url"`
	Branch string `json:"branch"`
}

type ManifestScripts struct {
	Build string `json:"build"`
	Dev   string `json:"dev"`
	Test  string `json:"test"`
	Clean string `json:"clean"`
	Lint  string `json:"lint"`
}

type ManifestEngines struct {
	Lunex string `json:"lunex"`
}

type ManifestMetadata struct {
	CreatedAt string   `json:"createdAt"`
	UpdatedAt string   `json:"updatedAt"`
	Keywords  []string `json:"keywords"`
	Tags      []string `json:"tags"`
}

type Manifest struct {
	Name          string             `json:"name"`
	Version       string             `json:"version"`
	Description   string             `json:"description"`
	Author        string             `json:"author"`
	License       string             `json:"license"`
	GitHub        string             `json:"github"`
	URL           string             `json:"url"`
	Homepage      string             `json:"homepage"`
	Documentation string             `json:"documentation"`
	Issues        string             `json:"issues"`
	Main          string             `json:"main"`
	Entry         string             `json:"entry"`
	Output        string             `json:"output"`
	Optimize      bool               `json:"optimize"`
	Minify        bool               `json:"minify"`
	Sourcemap     bool               `json:"sourcemap"`
	Environment   string             `json:"environment"`
	Repository    ManifestRepository `json:"repository"`
	Scripts       ManifestScripts    `json:"scripts"`
	Engines       ManifestEngines    `json:"engines"`
	Metadata      ManifestMetadata   `json:"metadata"`
	Targets       []string           `json:"targets"`
	Dependencies  map[string]string  `json:"dependencies"`
	Bin           map[string]string  `json:"bin"`
}

func resolveConfigPath(p string) string {
	info, err := os.Stat(p)
	if err == nil && info.IsDir() {
		candidate := filepath.Join(p, "config.lx")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
		return candidate
	}
	return p
}

func LoadManifest(p string) (*Manifest, error) {
	p = resolveConfigPath(p)
	data, err := os.ReadFile(p)
	if err != nil {
		return nil, err
	}

	trimmed := bytesTrimSpace(data)
	if len(trimmed) > 0 && trimmed[0] == '{' {
		var m Manifest
		if err := json.Unmarshal(trimmed, &m); err == nil {
			if m.Dependencies == nil {
				m.Dependencies = make(map[string]string)
			}
			if m.Main == "" && m.Entry != "" {
				m.Main = m.Entry
			}
			if m.Entry == "" && m.Main != "" {
				m.Entry = m.Main
			}
			return &m, nil
		}
	}

	if looksLikeConfigScript(string(data)) {
		m, err := loadManifestScript(p, string(data))
		if err != nil {
			return nil, err
		}
		if m != nil {
			return m, nil
		}
	}

	return parseConfigLX(string(data)), nil
}

func looksLikeConfigScript(content string) bool {
	for _, needle := range []string{"val ", "fn ", "const ", "@import(", "project = {", "config = {"} {
		if strings.Contains(content, needle) {
			return true
		}
	}
	return false
}

func loadManifestScript(p, content string) (*Manifest, error) {
	c := compiler.New(compiler.DefaultOptions)
	result := c.CompileSource(content, p)
	if !result.Success || result.AST == nil {
		return nil, fmt.Errorf("manifest script compile failed")
	}

	interp := c.Interpreter()
	interp.SetFilename(p)
	interp.SetSourceLines(strings.Split(content, "\n"))
	if _, err := interp.Exec(result.AST); err != nil {
		return nil, err
	}

	for _, name := range []string{"project", "config"} {
		if v, ok := interp.GetGlobal(name); ok && v != nil {
			if m, ok := manifestFromValue(v); ok {
				return m, nil
			}
		}
	}

	return nil, nil
}

func manifestFromValue(v *runtime.Value) (*Manifest, bool) {
	if v == nil || v.Tag != runtime.TypeObject || v.ObjVal == nil {
		return nil, false
	}

	m := &Manifest{
		Dependencies: make(map[string]string),
		Bin:          make(map[string]string),
	}

	getString := func(keys ...string) string {
		for _, key := range keys {
			if val, ok := v.ObjVal[key]; ok && val != nil {
				if val.Tag == runtime.TypeString {
					return val.StrVal
				}
				return val.ToString()
			}
		}
		return ""
	}

	getBool := func(key string) bool {
		if val, ok := v.ObjVal[key]; ok && val != nil {
			return val.IsTruthy()
		}
		return false
	}

	getStrMap := func(key string) map[string]string {
		result := make(map[string]string)
		if val, ok := v.ObjVal[key]; ok && val != nil && val.Tag == runtime.TypeObject {
			for k, mv := range val.ObjVal {
				if mv == nil {
					continue
				}
				if mv.Tag == runtime.TypeString {
					result[k] = mv.StrVal
				} else {
					result[k] = mv.ToString()
				}
			}
		}
		return result
	}

	getStrObj := func(key string, fields ...string) map[string]string {
		result := make(map[string]string)
		if val, ok := v.ObjVal[key]; ok && val != nil && val.Tag == runtime.TypeObject {
			for _, f := range fields {
				if fv, ok := val.ObjVal[f]; ok && fv != nil {
					if fv.Tag == runtime.TypeString {
						result[f] = fv.StrVal
					} else {
						result[f] = fv.ToString()
					}
				}
			}
		}
		return result
	}

	getStrArr := func(key string) []string {
		var result []string
		if val, ok := v.ObjVal[key]; ok && val != nil && val.Tag == runtime.TypeArray {
			for _, item := range val.ArrVal {
				if item == nil {
					continue
				}
				if item.Tag == runtime.TypeString {
					result = append(result, item.StrVal)
				} else {
					result = append(result, item.ToString())
				}
			}
		}
		return result
	}

	m.Name = getString("name")
	m.Version = getString("version")
	m.Description = getString("description")
	m.Author = getString("author")
	m.License = getString("license")
	m.GitHub = getString("github")
	m.URL = getString("url")
	m.Homepage = getString("homepage")
	m.Documentation = getString("documentation")
	m.Issues = getString("issues")
	m.Main = getString("main")
	m.Entry = getString("entry")
	m.Output = getString("output", "out")
	m.Environment = getString("environment")
	m.Optimize = getBool("optimize")
	m.Minify = getBool("minify")
	m.Sourcemap = getBool("sourcemap")
	m.Dependencies = getStrMap("dependencies")
	m.Bin = getStrMap("bin")

	repoFields := getStrObj("repository", "type", "url", "branch")
	m.Repository = ManifestRepository{
		Type:   repoFields["type"],
		URL:    repoFields["url"],
		Branch: repoFields["branch"],
	}

	scriptFields := getStrObj("scripts", "build", "dev", "test", "clean", "lint")
	m.Scripts = ManifestScripts{
		Build: scriptFields["build"],
		Dev:   scriptFields["dev"],
		Test:  scriptFields["test"],
		Clean: scriptFields["clean"],
		Lint:  scriptFields["lint"],
	}

	engFields := getStrObj("engines", "lunex")
	m.Engines = ManifestEngines{
		Lunex: engFields["lunex"],
	}

	if metaVal, ok := v.ObjVal["metadata"]; ok && metaVal != nil && metaVal.Tag == runtime.TypeObject {
		metaFields := getStrObj("metadata", "createdAt", "updatedAt")
		m.Metadata = ManifestMetadata{
			CreatedAt: metaFields["createdAt"],
			UpdatedAt: metaFields["updatedAt"],
			Keywords:  getStrArr("keywords"),
			Tags:      getStrArr("tags"),
		}
	}

	if targets, ok := v.ObjVal["targets"]; ok && targets != nil && targets.Tag == runtime.TypeArray {
		for _, item := range targets.ArrVal {
			if item == nil {
				continue
			}
			if item.Tag == runtime.TypeString {
				m.Targets = append(m.Targets, item.StrVal)
			} else {
				m.Targets = append(m.Targets, item.ToString())
			}
		}
	}

	if m.Main == "" && m.Entry != "" {
		m.Main = m.Entry
	}
	if m.Entry == "" && m.Main != "" {
		m.Entry = m.Main
	}
	if m.Output == "" {
		m.Output = "dist"
	}
	if m.Dependencies == nil {
		m.Dependencies = make(map[string]string)
	}
	if m.Bin == nil {
		m.Bin = make(map[string]string)
	}
	return m, true
}

func bytesTrimSpace(b []byte) []byte {
	start := 0
	end := len(b)
	for start < end && (b[start] == ' ' || b[start] == '\n' || b[start] == '\r' || b[start] == '\t') {
		start++
	}
	for end > start && (b[end-1] == ' ' || b[end-1] == '\n' || b[end-1] == '\r' || b[end-1] == '\t') {
		end--
	}
	return b[start:end]
}

func parseConfigLX(content string) *Manifest {
	m := &Manifest{Dependencies: make(map[string]string)}
	lines := strings.Split(content, "\n")
	block := ""
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "//") {
			continue
		}

		switch line {
		case "project {", "config {", "build {":
			block = "project"
			continue
		case "dependencies {", "deps {", "[dependencies]", "[deps]":
			block = "dependencies"
			continue
		case "}":
			block = ""
			continue
		}

		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		val = strings.TrimSpace(val)
		val = strings.Trim(val, "\"'")

		if block == "dependencies" {
			m.Dependencies[key] = val
			continue
		}

		switch key {
		case "name":
			m.Name = val
		case "version":
			m.Version = val
		case "description":
			m.Description = val
		case "author":
			m.Author = val
		case "license":
			m.License = val
		case "github":
			m.GitHub = val
		case "url":
			m.URL = val
		case "main":
			m.Main = val
		case "entry":
			m.Entry = val
		case "output", "out":
			m.Output = val
		case "optimize":
			m.Optimize = val == "true" || val == "1" || val == "yes"
		case "targets", "target":
			val = strings.Trim(val, "[]")
			for _, t := range strings.Split(val, ",") {
				t = strings.TrimSpace(strings.Trim(t, "\"' "))
				if t != "" {
					m.Targets = append(m.Targets, t)
				}
			}
		}
	}

	if m.Main == "" && m.Entry != "" {
		m.Main = m.Entry
	}
	if m.Entry == "" && m.Main != "" {
		m.Entry = m.Main
	}
	if m.Dependencies == nil {
		m.Dependencies = make(map[string]string)
	}
	return m
}

func SaveManifest(p string, m *Manifest) error {
	p = resolveConfigPath(p)
	if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
		return err
	}

	entry := m.Entry
	if entry == "" {
		entry = m.Main
	}
	if entry == "" {
		entry = "main.lx"
	}

	deps := ""
	for name, ver := range m.Dependencies {
		deps += fmt.Sprintf("    %q: %q\n", name, ver)
	}

	targets := "[]"
	if len(m.Targets) > 0 {
		quoted := make([]string, len(m.Targets))
		for i, t := range m.Targets {
			quoted[i] = fmt.Sprintf("%q", t)
		}
		targets = "[" + strings.Join(quoted, ", ") + "]"
	}

	content := fmt.Sprintf(`val project = {
  name: %q
  version: %q
  description: %q
  author: %q
  main: %q
  entry: %q
  output: %q
  optimize: %v
  targets: %s
  dependencies: {
%s  }
}

fn build() {
  project
}
`,
		m.Name, m.Version, m.Description, m.Author,
		entry, entry, m.Output, m.Optimize, targets, deps)

	return os.WriteFile(p, []byte(content), 0644)
}

func InitManifest(dir string, name string) error {
	p := filepath.Join(dir, "config.lx")
	if _, err := os.Stat(p); err == nil {
		return fmt.Errorf("config.lx already exists in %s", dir)
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	content := fmt.Sprintf(`val project = {
  name: %q
  version: %q
  description: ""
  author: ""
  license: "MIT"

  repository: {
    type: "git"
    url: ""
    branch: "main"
  }

  homepage: ""
  documentation: ""
  issues: ""

  main: "main.lx"
  entry: "main.lx"
  output: "dist"

  optimize: true
  minify: true
  sourcemap: false

  environment: "development"

  scripts: {
    build: "lunex build"
    dev: "lunex run main.lx"
    test: "lunex run tests/main.lx"
    clean: "rm -rf dist"
    lint: "lunex check main.lx"
  }

  dependencies: {
  }

  bin: {
  }

  engines: {
    lunex: ">=0.8.0"
  }

  metadata: {
    createdAt: ""
    updatedAt: ""
    keywords: []
    tags: []
  }
}

fn build() {
  project
}
`, name, meta.Version())

	return os.WriteFile(p, []byte(content), 0644)
}
