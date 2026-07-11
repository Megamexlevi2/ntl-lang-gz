package pkg

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"lunex/internal/bytecode"
	"lunex/internal/meta"
	"math"
	"net"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"lunex/internal/compiler"
	"lunex/internal/runtime"
)

type Module struct {
	Name    string
	Version string
	Source  string
	Path    string
}

type Manifest struct {
	Name         string            `json:"name"`
	Version      string            `json:"version"`
	Description  string            `json:"description"`
	Author       string            `json:"author"`
	License      string            `json:"license"`
	GitHub       string            `json:"github"`
	URL          string            `json:"url"`
	Main         string            `json:"main"`
	Entry        string            `json:"entry"`
	Output       string            `json:"output"`
	Optimize     bool              `json:"optimize"`
	Targets      []string          `json:"targets"`
	Dependencies map[string]string `json:"dependencies"`
}

const moduleCacheRoot = ".lunex/cache"
const moduleMetaFile = ".lunex-module.json"

func CacheDir() string {
	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}
	dir := filepath.Join(cwd, moduleCacheRoot)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return ""
	}
	return dir
}

func modulesRoot() string {
	base := CacheDir()
	if base == "" {
		return ""
	}
	dir := filepath.Join(base, "modules")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return ""
	}
	return dir
}

func ModuleDir(name, version string) string {
	if version == "" {
		version = "main"
	}
	safe := strings.ReplaceAll(name, "/", "__")
	root := modulesRoot()
	if root == "" {
		return ""
	}
	return filepath.Join(root, safe+"@"+version)
}

func moduleMetaPath(dir string) string {
	return filepath.Join(dir, moduleMetaFile)
}

type moduleMeta struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Source  string `json:"source"`
	Entry   string `json:"entry"`
}

func writeModuleMeta(dir string, mod *Module) error {
	if dir == "" || mod == nil {
		return nil
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	m := moduleMeta{
		Name:    mod.Name,
		Version: mod.Version,
		Source:  mod.Source,
		Entry:   filepath.Base(mod.Path),
	}
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(moduleMetaPath(dir), data, 0644)
}

func currentLunexBin() string {
	if exe, err := os.Executable(); err == nil && exe != "" {
		if abs, absErr := filepath.Abs(exe); absErr == nil && abs != "" {
			return abs
		}
		return exe
	}
	return "lunex"
}

func launcherScriptContent(lunexBin, target string) string {
	if lunexBin == "" {
		lunexBin = "lunex"
	}
	return fmt.Sprintf(`#!/bin/sh
LUNEXBIN=%q
exec "$LUNEXBIN" run %q "$@"
`, lunexBin, target)
}

func writeLauncherScript(shimPath, target string) error {
	if shimPath == "" || target == "" {
		return fmt.Errorf("invalid launcher target")
	}
	if err := os.MkdirAll(filepath.Dir(shimPath), 0755); err != nil {
		return err
	}
	if err := os.WriteFile(shimPath, []byte(launcherScriptContent(currentLunexBin(), target)), 0644); err != nil {
		return err
	}
	return os.Chmod(shimPath, 0755)
}

func writeGlobalLauncher(name, target string) error {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return fmt.Errorf("could not determine home directory")
	}
	binDir := filepath.Join(home, ".lunex", "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		return err
	}
	return writeLauncherScript(filepath.Join(binDir, name), target)
}

func writeLocalLauncher(name, target string) error {
	cwd, err := os.Getwd()
	if err != nil || cwd == "" {
		return fmt.Errorf("could not determine current directory")
	}
	return writeLauncherScript(filepath.Join(cwd, name), target)
}

func readModuleMeta(dir string) (*Module, bool) {
	data, err := os.ReadFile(moduleMetaPath(dir))
	if err != nil {
		return nil, false
	}
	var m moduleMeta
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, false
	}
	if m.Name == "" {
		return nil, false
	}
	if m.Version == "" {
		m.Version = "main"
	}
	entry := m.Entry
	if entry == "" {
		entry = "index.nax"
	}
	p := filepath.Join(dir, entry)
	return &Module{
		Name:    m.Name,
		Version: m.Version,
		Source:  m.Source,
		Path:    p,
	}, true
}

func packModuleArchive(dir, outputFile, mainEntry string) error {
	arch := &bytecode.NAXArchive{
		Version:   0x0700,
		BuildTime: time.Now().Unix(),
	}

	type pendingEntry struct {
		name string
		data []byte
	}

	entries := make([]pendingEntry, 0, 32)
	mainIndex := -1

	shouldSkip := func(p string) bool {
		clean := filepath.Clean(p)
		if clean == filepath.Clean(outputFile) {
			return true
		}
		if filepath.Base(p) == moduleMetaFile || filepath.Base(p) == ".lunex-entry" {
			return true
		}
		return false
	}

	markMain := func(sourceName string, sourceEntryIndex int, ncEntryIndex int) {
		base := strings.TrimSuffix(filepath.ToSlash(sourceName), ".lx")
		candidate := filepath.ToSlash(sourceName)
		if mainEntry != "" {
			wanted := filepath.ToSlash(mainEntry)
			if candidate != wanted && base != strings.TrimSuffix(wanted, ".lx") {
				return
			}
		}
		mainIndex = ncEntryIndex
		_ = sourceEntryIndex
	}

	err := filepath.WalkDir(dir, func(p string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		if shouldSkip(p) {
			return nil
		}

		rel, err := filepath.Rel(dir, p)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		base := strings.ToLower(filepath.Base(rel))

		switch strings.ToLower(filepath.Ext(rel)) {
		case ".lx":
			source, err := os.ReadFile(p)
			if err != nil {
				return fmt.Errorf("compile error for %s: %w", rel, err)
			}
			c := compiler.New(compiler.DefaultOptions)
			result := c.CompileSource(string(source), p)
			if !result.Success {
				var msgs []string
				for _, e := range result.Errors {
					msgs = append(msgs, e.Message)
				}
				return fmt.Errorf("compile error for %s: %s", rel, strings.Join(msgs, "; "))
			}
			chunk := &bytecode.ExportedChunk{
				Name:       strings.TrimSuffix(rel, ".lx"),
				SourceFile: p,
				SourceText: string(source),
			}
			objectData, err := bytecode.EncodeExportedWithAST(chunk, result.AST)
			if err != nil {
				return fmt.Errorf("compile error for %s: %w", rel, err)
			}
			entries = append(entries, pendingEntry{name: rel, data: source})
			sourceIndex := len(entries) - 1
			entries = append(entries, pendingEntry{name: strings.TrimSuffix(rel, ".lx") + ".nax", data: objectData})
			ncIndex := len(entries) - 1
			if mainIndex < 0 {
				if mainEntry == "" {

					mainIndex = ncIndex
				} else {
					markMain(rel, sourceIndex, ncIndex)
				}
			} else if mainEntry != "" {
				markMain(rel, sourceIndex, ncIndex)
			}
			if base == "main.lx" && mainEntry == "" {
				mainIndex = ncIndex
			}
		case ".nax":
			data, err := os.ReadFile(p)
			if err != nil {
				return fmt.Errorf("cannot read %s: %w", rel, err)
			}
			entries = append(entries, pendingEntry{name: rel, data: data})
			if mainIndex < 0 {
				if mainEntry == "" || rel == filepath.ToSlash(mainEntry) {
					mainIndex = len(entries) - 1
				}
			}
		}
		return nil
	})
	if err != nil {
		return err
	}

	if len(entries) == 0 {
		return fmt.Errorf("no source files found in %s", dir)
	}
	if mainIndex < 0 {
		mainIndex = 0
	}

	for _, e := range entries {
		arch.Entries = append(arch.Entries, bytecode.NAXEntry{Name: e.name, Data: e.data})
	}
	arch.MainIndex = uint32(mainIndex)
	return bytecode.PackNAXArchive(arch, outputFile)
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

	m := &Manifest{Dependencies: make(map[string]string)}
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

	m.Name = getString("name")
	m.Version = getString("version")
	m.Description = getString("description")
	m.Author = getString("author")
	m.License = getString("license")
	m.GitHub = getString("github")
	m.URL = getString("url")
	m.Main = getString("main")
	m.Entry = getString("entry")
	m.Output = getString("output", "out")

	if optimize, ok := v.ObjVal["optimize"]; ok && optimize != nil {
		m.Optimize = optimize.IsTruthy()
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
	if deps, ok := v.ObjVal["dependencies"]; ok && deps != nil && deps.Tag == runtime.TypeObject {
		for name, ver := range deps.ObjVal {
			if ver == nil {
				continue
			}
			if ver.Tag == runtime.TypeString {
				m.Dependencies[name] = ver.StrVal
			} else {
				m.Dependencies[name] = ver.ToString()
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
	return m, true
}

func Resolve(name string) (string, bool) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", false
	}

	if info, err := os.Stat(name); err == nil && !info.IsDir() {
		return name, true
	}

	for _, ext := range []string{".lx", ".nax", ".nax"} {
		if !strings.HasSuffix(strings.ToLower(name), ext) {
			if info, err := os.Stat(name + ext); err == nil && !info.IsDir() {
				return name + ext, true
			}
		}
	}

	cache := CacheDir()
	entries, err := os.ReadDir(cache)
	if err != nil {
		return "", false
	}

	entryFiles := []string{
		"index.lx", "main.lx",
		"index.nax", "main.nax",
		"config.lx",
	}

	var candidate string
	prefix := strings.ReplaceAll(name, "/", "__") + "@"

	suffix := "__" + strings.ReplaceAll(name, "/", "__") + "@"
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		dirName := e.Name()
		if !strings.HasPrefix(dirName, prefix) && !strings.Contains(dirName, suffix) {
			continue
		}
		p := filepath.Join(cache, dirName)

		if data, err := os.ReadFile(filepath.Join(p, ".lunex-entry")); err == nil {
			entryName := strings.TrimSpace(string(data))
			fp := filepath.Join(p, entryName)
			if st, err := os.Stat(fp); err == nil && !st.IsDir() {
				return fp, true
			}
		}

		for _, file := range entryFiles {
			fp := filepath.Join(p, file)
			if st, err := os.Stat(fp); err == nil && !st.IsDir() {
				return fp, true
			}
		}

		if files, err := os.ReadDir(p); err == nil {
			for _, f := range files {
				if !f.IsDir() && strings.HasSuffix(f.Name(), ".lx") {
					return filepath.Join(p, f.Name()), true
				}
			}
		}

		if candidate == "" {
			candidate = p
		}
	}

	if candidate != "" {
		if data, err := os.ReadFile(filepath.Join(candidate, ".lunex-entry")); err == nil {
			entryName := strings.TrimSpace(string(data))
			fp := filepath.Join(candidate, entryName)
			if st, err := os.Stat(fp); err == nil && !st.IsDir() {
				return fp, true
			}
		}
		for _, file := range entryFiles {
			fp := filepath.Join(candidate, file)
			if st, err := os.Stat(fp); err == nil && !st.IsDir() {
				return fp, true
			}
		}
		if files, err := os.ReadDir(candidate); err == nil {
			for _, f := range files {
				if !f.IsDir() && strings.HasSuffix(f.Name(), ".lx") {
					return filepath.Join(candidate, f.Name()), true
				}
			}
		}
	}

	return "", false
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
	if m.Main == "" {
		m.Main = entry
	}
	if m.Entry == "" {
		m.Entry = entry
	}

	output := m.Output
	if output == "" {
		output = "dist"
	}

	if m.Dependencies == nil {
		m.Dependencies = make(map[string]string)
	}

	description := m.Description
	if description == "" {
		description = "A Lunex project"
	}
	author := m.Author
	if author == "" {
		author = "unknown"
	}
	license := m.License
	if license == "" {
		license = "MIT"
	}
	github := m.GitHub
	url := m.URL

	var sb strings.Builder

	sb.WriteString("config.lx project manifest\n")
	sb.WriteString(fmt.Sprintf("%s\n", description))
	sb.WriteString(fmt.Sprintf("Author: %s\n", author))
	if github != "" {
		sb.WriteString(fmt.Sprintf("GitHub: %s\n", github))
	}
	if url != "" {
		sb.WriteString(fmt.Sprintf("URL: %s\n", url))
	}
	sb.WriteString(fmt.Sprintf("License: %s\n", license))
	sb.WriteString("\n")

	sb.WriteString("val project = {\n")

	sb.WriteString("  identity:\n")
	sb.WriteString(fmt.Sprintf("  name:        %q\n", m.Name))
	sb.WriteString(fmt.Sprintf("  version:     %q\n", m.Version))
	sb.WriteString(fmt.Sprintf("  description: %q\n", description))
	sb.WriteString(fmt.Sprintf("  author:      %q\n", author))
	sb.WriteString(fmt.Sprintf("  license:     %q\n", license))
	if github != "" {
		sb.WriteString(fmt.Sprintf("  github:      %q\n", github))
	} else {
		sb.WriteString("  github:      \"\"\n")
	}
	if url != "" {
		sb.WriteString(fmt.Sprintf("  url:         %q\n", url))
	} else {
		sb.WriteString("  url:         \"\"\n")
	}

	sb.WriteString("\n  build:\n")
	sb.WriteString(fmt.Sprintf("  main:        %q\n", m.Main))
	sb.WriteString(fmt.Sprintf("  entry:       %q\n", m.Entry))
	sb.WriteString(fmt.Sprintf("  output:      %q\n", output))
	if m.Optimize {
		sb.WriteString("  optimize:    true\n")
	} else {
		sb.WriteString("  optimize:    false\n")
	}

	sb.WriteString("\n  dependencies:\n")
	sb.WriteString("  dependencies: {\n")
	for name, ver := range m.Dependencies {
		sb.WriteString(fmt.Sprintf("    %q: %q\n", name, ver))
	}
	sb.WriteString("  }\n")
	sb.WriteString("}\n\n")

	sb.WriteString("build() is called by lunex build and must return the project object.\n")
	sb.WriteString("fn build() {\n")
	sb.WriteString("  project\n")
	sb.WriteString("}\n")

	return os.WriteFile(p, []byte(sb.String()), 0644)
}

func InitManifest(dir string, name string) error {
	p := filepath.Join(dir, "config.lx")
	if _, err := os.Stat(p); err == nil {
		return fmt.Errorf("config.lx already exists")
	}
	m := &Manifest{
		Name:         name,
		Version:      meta.Version(),
		Description:  "A Lunex project",
		Author:       "unknown",
		License:      "MIT",
		GitHub:       "",
		URL:          "",
		Main:         "main.lx",
		Entry:        "main.lx",
		Output:       "dist",
		Optimize:     true,
		Targets:      []string{},
		Dependencies: make(map[string]string),
	}
	return SaveManifest(p, m)
}

func AddToManifest(manifestPath, spec string, mod *Module) error {
	var m *Manifest
	if loaded, err := LoadManifest(manifestPath); err == nil {
		m = loaded
	} else {
		m = &Manifest{Dependencies: make(map[string]string)}
	}
	if m.Dependencies == nil {
		m.Dependencies = make(map[string]string)
	}
	m.Dependencies[mod.Name] = mod.Version
	return SaveManifest(resolveConfigPath(manifestPath), m)
}

func List() []Module {
	root := modulesRoot()
	entries, _ := os.ReadDir(root)
	var mods []Module
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		dir := filepath.Join(root, e.Name())
		if mod, ok := readModuleMeta(dir); ok {
			mods = append(mods, *mod)
			continue
		}
		parts := strings.Split(e.Name(), "@")
		if len(parts) == 2 {

			entryPath := filepath.Join(dir, "index.lx")
			if _, err := os.Stat(entryPath); err != nil {
				entryPath = filepath.Join(dir, "index.nax")
			}
			mods = append(mods, Module{
				Name:    strings.ReplaceAll(parts[0], "__", "/"),
				Version: parts[1],
				Path:    entryPath,
			})
		}
	}
	return mods
}

func Remove(name string) error {
	root := modulesRoot()
	entries, _ := os.ReadDir(root)
	prefix := strings.ReplaceAll(name, "/", "__") + "@"
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), prefix) {
			return os.RemoveAll(filepath.Join(root, e.Name()))
		}
	}
	return fmt.Errorf("package %q not found", name)
}

func resolveSource(spec string) (owner, repo, ref, subpath string) {
	spec = strings.TrimPrefix(spec, "github.com/")
	spec = strings.TrimPrefix(spec, "https://github.com/")

	atIdx := strings.Index(spec, "@")
	ref = "main"
	if atIdx > 0 {
		ref = spec[atIdx+1:]
		spec = spec[:atIdx]
	}

	parts := strings.Split(spec, "/")
	if len(parts) >= 2 {
		owner = parts[0]
		repo = parts[1]
	}
	if len(parts) > 2 {
		subpath = strings.Join(parts[2:], "/")
	}
	return
}

type githubEntry struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	DownloadURL string `json:"download_url"`
	Type        string `json:"type"`
	Size        int    `json:"size"`
	HTMLURL     string `json:"html_url"`
}

type githubClient struct {
	http *http.Client
}

func newGitHubClient() *githubClient {

	resolver := &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			d := net.Dialer{Timeout: 10 * time.Second}
			conn, err := d.DialContext(ctx, "tcp", "8.8.8.8:53")
			if err != nil {

				return d.DialContext(ctx, "tcp", address)
			}
			return conn, nil
		},
	}

	dialer := &net.Dialer{
		Timeout:   20 * time.Second,
		KeepAlive: 30 * time.Second,
		Resolver:  resolver,
	}

	transport := &http.Transport{

		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           dialer.DialContext,
		TLSHandshakeTimeout:   15 * time.Second,
		ResponseHeaderTimeout: 30 * time.Second,
		ForceAttemptHTTP2:     true,
	}

	return &githubClient{
		http: &http.Client{
			Transport: transport,
			Timeout:   90 * time.Second,
		},
	}
}

func (c *githubClient) githubAPIRequest(url string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	return c.http.Do(req)
}

func (c *githubClient) getJSON(url string, v interface{}) error {
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			time.Sleep(time.Duration(attempt) * 500 * time.Millisecond)
		}
		resp, err := c.githubAPIRequest(url)
		if err != nil {
			lastErr = err
			continue
		}
		defer resp.Body.Close()
		if resp.StatusCode == 403 {
			return fmt.Errorf("GitHub rate limit exceeded — wait a moment and try again")
		}
		if resp.StatusCode == 404 {
			return fmt.Errorf("not found: %s", url)
		}
		if resp.StatusCode != 200 {
			return fmt.Errorf("GitHub API returned HTTP %d for %s", resp.StatusCode, url)
		}
		if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
			lastErr = err
			continue
		}
		return nil
	}
	return fmt.Errorf("GitHub API request failed after 3 attempts: %w", lastErr)
}

type installProgress struct {
	pkg       string
	total     int64
	done      int64
	bytes     int64
	startTime time.Time
}

func newInstallProgress(pkg string, total int) *installProgress {
	p := &installProgress{
		pkg:       pkg,
		total:     int64(total),
		startTime: time.Now(),
	}

	fmt.Printf("\n  Installing \033[1;36m%s\033[0m\n", pkg)
	if total > 0 {
		fmt.Printf("  %d file(s) to download\n\n", total)
	} else {
		fmt.Printf("  Resolving file list...\n\n")
	}
	return p
}

func formatBytes(n int64) string {
	switch {
	case n >= 1024*1024:
		return fmt.Sprintf("%.1f MB", float64(n)/1024/1024)
	case n >= 1024:
		return fmt.Sprintf("%.1f KB", float64(n)/1024)
	default:
		return fmt.Sprintf("%d B", n)
	}
}

func formatSpeed(bytesPerSec float64) string {
	switch {
	case bytesPerSec >= 1024*1024:
		return fmt.Sprintf("%.1f MB/s", bytesPerSec/1024/1024)
	case bytesPerSec >= 1024:
		return fmt.Sprintf("%.1f KB/s", bytesPerSec/1024)
	default:
		return fmt.Sprintf("%.0f B/s", bytesPerSec)
	}
}

func (p *installProgress) renderBar(filename string) {
	done := atomic.LoadInt64(&p.done)
	total := atomic.LoadInt64(&p.total)
	rx := atomic.LoadInt64(&p.bytes)

	elapsed := time.Since(p.startTime).Seconds()
	var speed float64
	if elapsed > 0 {
		speed = float64(rx) / elapsed
	}

	const barWidth = 24
	var pct float64
	filled := 0
	if total > 0 {
		pct = math.Min(float64(done)/float64(total)*100, 100)
		filled = int(math.Round(float64(barWidth) * pct / 100))
	}

	bar := ""
	for i := 0; i < barWidth; i++ {
		if i < filled {
			bar += "█"
		} else {
			bar += "░"
		}
	}

	var line string
	if total > 0 {
		line = fmt.Sprintf(
			"  \033[36m[%s]\033[0m \033[1m%5.1f%%\033[0m  "+
				"\033[2m%d/%d files\033[0m  %s  \033[33m%s\033[0m  \033[2m%-20s\033[0m",
			bar, pct, done, total,
			formatBytes(rx), formatSpeed(speed),
			truncateFilename(filename, 20),
		)
	} else {
		line = fmt.Sprintf(
			"  \033[36m[%s]\033[0m  "+
				"\033[2m%d files\033[0m  %s  \033[33m%s\033[0m  \033[2m%-20s\033[0m",
			bar, done,
			formatBytes(rx), formatSpeed(speed),
			truncateFilename(filename, 20),
		)
	}

	fmt.Printf("\r%-100s", line)
}

func (p *installProgress) finish() {
	done := atomic.LoadInt64(&p.done)
	rx := atomic.LoadInt64(&p.bytes)
	elapsed := time.Since(p.startTime)

	fmt.Printf("\r\033[K")
	fmt.Printf(
		"  \033[1;32m✓\033[0m  \033[1m%s\033[0m — %d file(s), %s in %s\n\n",
		p.pkg, done, formatBytes(rx), elapsed.Round(time.Millisecond),
	)
}

func truncateFilename(name string, maxLen int) string {
	base := filepath.Base(name)
	if len(base) <= maxLen {
		return base
	}
	return "…" + base[len(base)-maxLen+1:]
}

type progressReader struct {
	r    io.Reader
	prog *installProgress
	name string
}

func (pr *progressReader) Read(p []byte) (int, error) {
	n, err := pr.r.Read(p)
	if n > 0 {
		atomic.AddInt64(&pr.prog.bytes, int64(n))
		pr.prog.renderBar(pr.name)
	}
	return n, err
}

func (c *githubClient) downloadFile(url string) ([]byte, error) {
	return c.downloadFileProgress(url, "", nil)
}

func (c *githubClient) downloadFileProgress(url, filename string, prog *installProgress) ([]byte, error) {
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			time.Sleep(time.Duration(attempt) * 300 * time.Millisecond)
		}
		resp, err := c.http.Get(url)
		if err != nil {
			lastErr = err
			continue
		}
		defer resp.Body.Close()
		if resp.StatusCode == 404 {
			return nil, fmt.Errorf("file not found: %s", url)
		}
		if resp.StatusCode != 200 {
			lastErr = fmt.Errorf("HTTP %d: %s", resp.StatusCode, url)
			continue
		}

		var reader io.Reader = resp.Body
		if prog != nil && filename != "" {
			reader = &progressReader{r: resp.Body, prog: prog, name: filename}
		}
		data, err := io.ReadAll(reader)
		if err != nil {
			lastErr = err
			continue
		}
		return data, nil
	}
	return nil, fmt.Errorf("download failed after 3 attempts: %w", lastErr)
}

func (c *githubClient) fetchDirRecursive(owner, repo, ref, remotePath, localDir string) error {
	return c.fetchDirRecursiveProgress(owner, repo, ref, remotePath, localDir, nil)
}

func (c *githubClient) fetchDirRecursiveProgress(owner, repo, ref, remotePath, localDir string, prog *installProgress) error {

	apiURL := fmt.Sprintf(
		"https://api.github.com/repos/%s/%s/contents/%s?ref=%s",
		owner, repo, remotePath, ref)

	resp, err := c.githubAPIRequest(apiURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == 403 {
		return fmt.Errorf("GitHub rate limit exceeded — wait a moment and try again")
	}
	if resp.StatusCode == 404 {
		return fmt.Errorf("path not found: %s/%s/%s (ref: %s)", owner, repo, remotePath, ref)
	}
	if resp.StatusCode != 200 {
		return fmt.Errorf("GitHub Contents API returned HTTP %d for %s/%s/%s@%s", resp.StatusCode, owner, repo, remotePath, ref)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading GitHub API response: %w", err)
	}

	first := bytes.TrimSpace(body)
	if len(first) == 0 {
		return fmt.Errorf("empty response from GitHub API for %s", remotePath)
	}

	var entries []githubEntry
	if first[0] == '[' {
		if err := json.Unmarshal(body, &entries); err != nil {
			return fmt.Errorf("parsing GitHub directory listing: %w", err)
		}
	} else {
		var single githubEntry
		if err := json.Unmarshal(body, &single); err != nil {
			return fmt.Errorf("parsing GitHub file entry: %w", err)
		}
		entries = []githubEntry{single}
	}

	for _, entry := range entries {
		localPath := filepath.Join(localDir, filepath.FromSlash(entry.Name))
		switch entry.Type {
		case "file":
			if entry.DownloadURL == "" {
				continue
			}
			if err := os.MkdirAll(filepath.Dir(localPath), 0755); err != nil {
				return err
			}
			data, err := c.downloadFileProgress(entry.DownloadURL, entry.Name, prog)
			if err != nil {
				return fmt.Errorf("downloading %s: %w", entry.Path, err)
			}
			if err := os.WriteFile(localPath, data, 0644); err != nil {
				return err
			}
			if prog != nil {
				atomic.AddInt64(&prog.done, 1)
				prog.renderBar(entry.Name)
			}
		case "dir":
			if err := os.MkdirAll(localPath, 0755); err != nil {
				return err
			}
			if err := c.fetchDirRecursiveProgress(owner, repo, ref, entry.Path, localPath, prog); err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *githubClient) fetchViaTreeAPI(owner, repo, ref, subpath, localDir string) error {
	return c.fetchViaTreeAPIProgress(owner, repo, ref, subpath, localDir, nil)
}

func (c *githubClient) fetchViaTreeAPIProgress(owner, repo, ref, subpath, localDir string, prog *installProgress) error {
	apiURL := fmt.Sprintf(
		"https://api.github.com/repos/%s/%s/git/trees/%s?recursive=1",
		owner, repo, ref)

	resp, err := c.githubAPIRequest(apiURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == 403 {
		return fmt.Errorf("GitHub rate limit exceeded — wait a moment and try again")
	}
	if resp.StatusCode == 404 {
		return fmt.Errorf("repository not found: %s/%s@%s", owner, repo, ref)
	}
	if resp.StatusCode != 200 {
		return fmt.Errorf("GitHub Trees API returned HTTP %d for %s/%s@%s", resp.StatusCode, owner, repo, ref)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var tree struct {
		Tree []struct {
			Path string `json:"path"`
			Type string `json:"type"`
			URL  string `json:"url"`
		} `json:"tree"`
		Truncated bool `json:"truncated"`
	}
	if err := json.Unmarshal(body, &tree); err != nil {
		return fmt.Errorf("parsing Git Trees response: %w", err)
	}
	if tree.Truncated {
		fmt.Println("  note: GitHub tree response was truncated; some files may be missing")
	}

	prefix := strings.Trim(subpath, "/")
	rawBase := fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s", owner, repo, ref)

	type blobEntry struct {
		nodePath string
		relPath  string
	}
	var blobs []blobEntry
	for _, node := range tree.Tree {
		if node.Type != "blob" {
			continue
		}
		relPath := node.Path
		if prefix != "" {
			if !strings.HasPrefix(node.Path, prefix+"/") {
				continue
			}
			relPath = strings.TrimPrefix(node.Path, prefix+"/")
		}
		if !strings.HasSuffix(strings.ToLower(relPath), ".lx") {
			continue
		}
		blobs = append(blobs, blobEntry{nodePath: node.Path, relPath: relPath})
	}

	if len(blobs) == 0 {
		return fmt.Errorf("no .lx files found under %q in %s/%s@%s", prefix, owner, repo, ref)
	}

	if prog != nil {
		atomic.StoreInt64(&prog.total, int64(len(blobs)))
	}

	for _, blob := range blobs {
		localPath := filepath.Join(localDir, filepath.FromSlash(blob.relPath))
		if err := os.MkdirAll(filepath.Dir(localPath), 0755); err != nil {
			return err
		}
		data, err := c.downloadFileProgress(rawBase+"/"+blob.nodePath, blob.relPath, prog)
		if err != nil {
			return fmt.Errorf("downloading %s: %w", blob.nodePath, err)
		}
		if err := os.WriteFile(localPath, data, 0644); err != nil {
			return err
		}
		if prog != nil {
			atomic.AddInt64(&prog.done, 1)
			prog.renderBar(blob.relPath)
		}
	}
	return nil
}

func (c *githubClient) fetchViaCodeload(owner, repo, ref, subpath, localDir string, prog *installProgress) error {

	branchURL := fmt.Sprintf(
		"https://codeload.github.com/%s/%s/zip/refs/heads/%s",
		owner, repo, ref)
	tagURL := fmt.Sprintf(
		"https://codeload.github.com/%s/%s/zip/refs/tags/%s",
		owner, repo, ref)

	fmt.Printf("  \033[2m→ fallback: downloading ZIP from codeload.github.com\033[0m\n")

	var zipData []byte
	var dlErr error

	for _, url := range []string{branchURL, tagURL} {
		zipData, dlErr = c.downloadZipProgress(url, owner+"/"+repo, prog)
		if dlErr == nil {
			break
		}

		if strings.Contains(dlErr.Error(), "404") {
			continue
		}

		break
	}
	if dlErr != nil {
		return fmt.Errorf("codeload download failed: %w", dlErr)
	}

	zr, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
	if err != nil {
		return fmt.Errorf("reading ZIP archive: %w", err)
	}

	rootPrefix := repo + "-" + ref + "/"

	prefix := strings.Trim(subpath, "/")
	var filePrefix string
	if prefix != "" {
		filePrefix = rootPrefix + prefix + "/"
	} else {
		filePrefix = rootPrefix
	}

	var qualifying []*zip.File
	for _, f := range zr.File {
		if f.FileInfo().IsDir() {
			continue
		}
		if !strings.HasPrefix(f.Name, filePrefix) {
			continue
		}
		relPath := strings.TrimPrefix(f.Name, filePrefix)
		if relPath == "" {
			continue
		}
		if !strings.HasSuffix(strings.ToLower(relPath), ".lx") {
			continue
		}
		qualifying = append(qualifying, f)
	}

	if len(qualifying) == 0 {
		return fmt.Errorf("no .lx files found under %q in %s/%s@%s (ZIP)", prefix, owner, repo, ref)
	}

	if prog != nil {
		atomic.StoreInt64(&prog.total, int64(len(qualifying)))
		atomic.StoreInt64(&prog.done, 0)
	}

	for _, f := range qualifying {
		relPath := strings.TrimPrefix(f.Name, filePrefix)
		localPath := filepath.Join(localDir, filepath.FromSlash(relPath))

		if err := os.MkdirAll(filepath.Dir(localPath), 0755); err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			return fmt.Errorf("opening ZIP entry %s: %w", f.Name, err)
		}

		var reader io.Reader = rc
		if prog != nil {
			reader = &progressReader{r: rc, prog: prog, name: relPath}
		}
		data, err := io.ReadAll(reader)
		rc.Close()
		if err != nil {
			return fmt.Errorf("reading ZIP entry %s: %w", f.Name, err)
		}

		if err := os.WriteFile(localPath, data, 0644); err != nil {
			return err
		}
		if prog != nil {
			atomic.AddInt64(&prog.done, 1)
			prog.renderBar(relPath)
		}
	}
	return nil
}

func (c *githubClient) downloadZipProgress(url, label string, prog *installProgress) ([]byte, error) {
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			time.Sleep(time.Duration(attempt) * 500 * time.Millisecond)
		}
		resp, err := c.http.Get(url)
		if err != nil {
			lastErr = err
			continue
		}
		defer resp.Body.Close()
		if resp.StatusCode == 404 {
			return nil, fmt.Errorf("404: %s", url)
		}
		if resp.StatusCode != 200 {
			lastErr = fmt.Errorf("HTTP %d: %s", resp.StatusCode, url)
			continue
		}

		if cl := resp.ContentLength; cl > 0 && prog != nil {

			_ = cl
		}

		var reader io.Reader = resp.Body
		if prog != nil {
			reader = &progressReader{r: resp.Body, prog: prog, name: label + ".zip"}
		}
		data, err := io.ReadAll(reader)
		if err != nil {
			lastErr = err
			continue
		}
		return data, nil
	}
	return nil, fmt.Errorf("ZIP download failed after 3 attempts: %w", lastErr)
}

func InstallFromGitHub(name, owner, repo, ref, subpath string) (*Module, error) {
	if owner == "" || repo == "" {
		return nil, fmt.Errorf("invalid GitHub source: owner and repo are required")
	}
	if ref == "" {
		ref = "main"
	}

	dir := ModuleDir(name, ref)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("creating module directory: %w", err)
	}

	client := newGitHubClient()

	prog := newInstallProgress(name, 0)

	err := client.fetchDirRecursiveProgress(owner, repo, ref, subpath, dir, prog)
	if err != nil {

		atomic.StoreInt64(&prog.done, 0)
		atomic.StoreInt64(&prog.bytes, 0)
		treeErr := client.fetchViaTreeAPIProgress(owner, repo, ref, subpath, dir, prog)
		if treeErr != nil {

			atomic.StoreInt64(&prog.done, 0)
			atomic.StoreInt64(&prog.bytes, 0)
			atomic.StoreInt64(&prog.total, 0)
			zipErr := client.fetchViaCodeload(owner, repo, ref, subpath, dir, prog)
			if zipErr != nil {
				fmt.Println()
				return nil, fmt.Errorf(
					"could not install %s/%s@%s\n"+
						"  contents API : %v\n"+
						"  trees API    : %v\n"+
						"  codeload ZIP : %v",
					owner, repo, ref, err, treeErr, zipErr,
				)
			}
		}
	}

	prog.finish()

	entry := ""

	if configData, err := os.ReadFile(filepath.Join(dir, "config.lx")); err == nil {
		configStr := string(configData)
		for _, field := range []string{"entry", "main"} {
			needle := field + ":"
			idx := strings.Index(configStr, needle)
			if idx < 0 {
				continue
			}
			rest := strings.TrimSpace(configStr[idx+len(needle):])
			rest = strings.Trim(rest, "\"'")
			if end := strings.IndexAny(rest, " \t\r\n"); end > 0 {
				rest = rest[:end]
			}
			rest = strings.TrimSpace(rest)
			if rest == "" {
				continue
			}
			if _, statErr := os.Stat(filepath.Join(dir, rest)); statErr == nil {
				entry = rest
				break
			}
		}
	}

	if entry == "" {
		for _, candidate := range []string{"index.lx", "main.lx", "index.nax", "main.nax"} {
			if _, err := os.Stat(filepath.Join(dir, candidate)); err == nil {
				entry = candidate
				break
			}
		}
	}

	if entry == "" {
		if files, err := os.ReadDir(dir); err == nil {
			for _, f := range files {
				if !f.IsDir() && strings.HasSuffix(f.Name(), ".lx") && f.Name() != "config.lx" {
					entry = f.Name()
					break
				}
			}
		}
	}

	if entry == "" {
		entry = "index.lx"
	}

	_ = os.WriteFile(filepath.Join(dir, ".lunex-entry"), []byte(entry), 0644)

	mod := &Module{
		Name:    name,
		Version: ref,
		Source:  fmt.Sprintf("github.com/%s/%s/%s@%s", owner, repo, subpath, ref),
		Path:    filepath.Join(dir, entry),
	}
	_ = writeModuleMeta(dir, mod)
	_ = writeGlobalLauncher(name, mod.Path)

	return mod, nil
}

func Install(spec string) (*Module, error) {
	spec = strings.TrimSpace(spec)
	if spec == "" {
		return nil, fmt.Errorf("empty package spec")
	}

	if strings.Contains(spec, "/") {
		owner, repo, ref, subpath := resolveSource(spec)
		if owner == "" || repo == "" {
			return nil, fmt.Errorf("invalid package spec %q: expected owner/repo[@ref]", spec)
		}
		pkgName := repo
		if subpath != "" {

			pkgName = path.Base(subpath)
		}
		return InstallFromGitHub(pkgName, owner, repo, ref, subpath)
	}

	return nil, fmt.Errorf("invalid package spec %q: expected github.com/user/repo or https://github.com/user/repo", spec)
}
