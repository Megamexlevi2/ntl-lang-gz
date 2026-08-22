package pkg

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"lunex/internal/manifest"
	"lunex/internal/semver"
)

type Module struct {
	Name    string
	Version string
	Source  string
	URL     string
	Path    string
	Global  bool
}

const moduleMetaFile = ".lunex-module.toml"

func GlobalRoot() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return ""
	}
	dir := filepath.Join(home, ".lunex")
	_ = os.MkdirAll(dir, 0755)
	return dir
}

func globalModulesRoot() string {
	root := GlobalRoot()
	if root == "" {
		return ""
	}
	dir := filepath.Join(root, "modules")
	_ = os.MkdirAll(dir, 0755)
	return dir
}

func LocalRoot() string {
	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}
	dir := filepath.Join(cwd, ".lunex")
	_ = os.MkdirAll(dir, 0755)
	return dir
}

func localModulesRoot() string {
	root := LocalRoot()
	if root == "" {
		return ""
	}
	dir := filepath.Join(root, "modules")
	_ = os.MkdirAll(dir, 0755)
	return dir
}

func sanitizeName(name string) string {
	return strings.ReplaceAll(name, "/", "__")
}

func moduleDir(root, name, version string) string {
	if version == "" {
		version = "0.0.0"
	}
	return filepath.Join(root, sanitizeName(name)+"@"+version)
}

func moduleMetaPath(dir string) string {
	return filepath.Join(dir, moduleMetaFile)
}

type moduleMeta struct {
	Name    string
	Version string
	Source  string
	URL     string
	Entry   string
}

func writeModuleMeta(dir string, mod *Module, entryRelPath string) error {
	if dir == "" || mod == nil {
		return nil
	}
	p := moduleMetaPath(dir)
	var sb strings.Builder
	fmt.Fprintf(&sb, "name = %q\n", mod.Name)
	fmt.Fprintf(&sb, "version = %q\n", mod.Version)
	fmt.Fprintf(&sb, "source = %q\n", mod.Source)
	fmt.Fprintf(&sb, "url = %q\n", mod.URL)
	fmt.Fprintf(&sb, "entry = %q\n", entryRelPath)
	return os.WriteFile(p, []byte(sb.String()), 0644)
}

func readModuleMeta(dir string) (*moduleMeta, bool) {
	data, err := os.ReadFile(moduleMetaPath(dir))
	if err != nil {
		return nil, false
	}
	m := &moduleMeta{}
	for _, line := range strings.Split(string(data), "\n") {
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		val = strings.Trim(strings.TrimSpace(val), "\"")
		switch key {
		case "name":
			m.Name = val
		case "version":
			m.Version = val
		case "source":
			m.Source = val
		case "url":
			m.URL = val
		case "entry":
			m.Entry = val
		}
	}
	if m.Name == "" {
		return nil, false
	}
	return m, true
}

func InitManifest(dir, name string) error {
	p := manifest.Path(dir)
	if _, err := os.Stat(p); err == nil {
		return fmt.Errorf("lunex.toml already exists")
	}
	proj := manifest.NewProject(name)
	return manifest.Save(p, proj)
}

func AddLibraryToManifest(manifestPath string, lib *manifest.Library) error {
	p := manifest.Path(manifestPath)
	var proj *manifest.Project
	if loaded, err := manifest.Load(p); err == nil {
		proj = loaded
	} else {
		proj = manifest.NewProject(filepath.Base(filepath.Dir(p)))
	}
	proj.AddLibrary(lib)
	return manifest.Save(p, proj)
}

func Resolve(name string) (string, bool) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", false
	}

	if info, err := os.Stat(name); err == nil && !info.IsDir() {
		return name, true
	}
	for _, ext := range []string{".lx", ".nax"} {
		if !strings.HasSuffix(strings.ToLower(name), ext) {
			if info, err := os.Stat(name + ext); err == nil && !info.IsDir() {
				return name + ext, true
			}
		}
	}

	wantVersion := lockedVersion(name)

	if p, ok := resolveFromStore(localModulesRoot(), name, wantVersion); ok {
		return p, true
	}
	if p, ok := resolveFromStore(globalModulesRoot(), name, wantVersion); ok {
		return p, true
	}
	return "", false
}

func lockedVersion(name string) string {
	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}
	lock, err := manifest.LoadLock(cwd)
	if err != nil {
		return ""
	}
	if m, ok := lock.Modules[name]; ok {
		return m.Version
	}
	return ""
}

func resolveFromStore(root, name, wantVersion string) (string, bool) {
	if root == "" {
		return "", false
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		return "", false
	}

	safe := sanitizeName(name)
	prefix := safe + "@"

	var candidates []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if strings.HasPrefix(e.Name(), prefix) {
			candidates = append(candidates, e.Name())
		}
	}
	if len(candidates) == 0 {
		return "", false
	}

	var chosenDir string
	if wantVersion != "" {
		target := prefix + wantVersion
		for _, c := range candidates {
			if c == target {
				chosenDir = c
				break
			}
		}
	}
	if chosenDir == "" {
		sort.Slice(candidates, func(i, j int) bool {
			vi, _ := semver.Parse(strings.TrimPrefix(candidates[i], prefix))
			vj, _ := semver.Parse(strings.TrimPrefix(candidates[j], prefix))
			return semver.Compare(vi, vj) > 0
		})
		chosenDir = candidates[0]
	}

	dir := filepath.Join(root, chosenDir)
	if meta, ok := readModuleMeta(dir); ok && meta.Entry != "" {
		fp := filepath.Join(dir, meta.Entry)
		if st, err := os.Stat(fp); err == nil && !st.IsDir() {
			return fp, true
		}
	}
	for _, candidate := range []string{"index.lx", "main.lx", "index.nax", "main.nax"} {
		fp := filepath.Join(dir, candidate)
		if st, err := os.Stat(fp); err == nil && !st.IsDir() {
			return fp, true
		}
	}
	if files, err := os.ReadDir(dir); err == nil {
		for _, f := range files {
			if !f.IsDir() && strings.HasSuffix(f.Name(), ".lx") {
				return filepath.Join(dir, f.Name()), true
			}
		}
	}
	return "", false
}

func binRoot(opts InstallOptions) string {
	var root string
	if opts.Global {
		root = GlobalRoot()
	} else {
		root = LocalRoot()
	}
	if root == "" {
		return ""
	}
	dir := filepath.Join(root, "bin")
	_ = os.MkdirAll(dir, 0755)
	return dir
}

func linkBinaries(moduleDir string, opts InstallOptions) ([]string, error) {
	manifestPath := filepath.Join(moduleDir, manifest.ManifestFile)
	if _, err := os.Stat(manifestPath); err != nil {
		return nil, nil
	}
	proj, err := manifest.Load(manifestPath)
	if err != nil || len(proj.Bin) == 0 {
		return nil, nil
	}

	binDir := binRoot(opts)
	if binDir == "" {
		return nil, fmt.Errorf("could not resolve bin directory")
	}

	var linked []string
	for _, cmd := range proj.OrderedBin() {
		entryRel := proj.Bin[cmd]
		entryAbs := filepath.Join(moduleDir, entryRel)
		if err := writeBinShim(binDir, cmd, entryAbs); err != nil {
			return linked, fmt.Errorf("linking bin %q: %w", cmd, err)
		}
		linked = append(linked, cmd)
	}
	return linked, nil
}

func writeBinShim(binDir, command, entryAbsPath string) error {
	shimPath := filepath.Join(binDir, command)
	script := "#!/bin/sh\nexec lunex run " + shellQuote(entryAbsPath) + " \"$@\"\n"
	if err := os.WriteFile(shimPath, []byte(script), 0755); err != nil {
		return err
	}
	return os.Chmod(shimPath, 0755)
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

func printLinkedBins(commands []string, opts InstallOptions) {
	binDir := binRoot(opts)
	scope := "locally"
	if opts.Global {
		scope = "globally"
	}
	for _, cmd := range commands {
		fmt.Printf("  linked command %q %s (%s)\n", cmd, scope, filepath.Join(binDir, cmd))
	}
	if opts.Global {
		fmt.Printf("  make sure %s is on your PATH to run it directly\n", binDir)
	}
}

func unlinkBinaries(entryAbsPath string) {
	for _, opts := range []InstallOptions{{Global: true}, {Global: false}} {
		binDir := binRoot(opts)
		if binDir == "" {
			continue
		}
		entries, err := os.ReadDir(binDir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			p := filepath.Join(binDir, e.Name())
			data, err := os.ReadFile(p)
			if err != nil {
				continue
			}
			if strings.Contains(string(data), shellQuote(entryAbsPath)) {
				_ = os.Remove(p)
			}
		}
	}
}

func LinkProject() ([]string, error) {
	proj, err := manifest.Load(".")
	if err != nil {
		return nil, err
	}
	if len(proj.Bin) == 0 {
		return nil, fmt.Errorf("lunex.toml has no [project.bin] entries to link")
	}
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	binDir := binRoot(InstallOptions{Global: true})
	if binDir == "" {
		return nil, fmt.Errorf("could not resolve global bin directory")
	}
	var linked []string
	for _, cmd := range proj.OrderedBin() {
		entryAbs := filepath.Join(cwd, proj.Bin[cmd])
		if err := writeBinShim(binDir, cmd, entryAbs); err != nil {
			return linked, fmt.Errorf("linking bin %q: %w", cmd, err)
		}
		linked = append(linked, cmd)
	}
	return linked, nil
}

type InstallOptions struct {
	Global bool
}

func InstallLibrary(lib *manifest.Library, opts InstallOptions) (*Module, error) {
	switch lib.Source {
	case "local":
		return installLocalLibrary(lib, opts)
	case "github-release":
		return installGitHubRelease(lib, opts)
	default:
		return installGitHubSource(lib, opts)
	}
}

func storeRoot(opts InstallOptions) string {
	if opts.Global {
		return globalModulesRoot()
	}
	return localModulesRoot()
}

func installLocalLibrary(lib *manifest.Library, opts InstallOptions) (*Module, error) {
	if lib.Path == "" {
		return nil, fmt.Errorf("local library %q needs a path", lib.Name)
	}
	abs, err := filepath.Abs(lib.Path)
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(abs); err != nil {
		return nil, fmt.Errorf("local library path not found: %s", abs)
	}
	entry := findEntryInDir(abs)
	if entry == "" {
		return nil, fmt.Errorf("no .lx entry file found in %s", abs)
	}
	if linked, linkErr := linkBinaries(abs, opts); linkErr == nil && len(linked) > 0 {
		printLinkedBins(linked, opts)
	}
	return &Module{
		Name:    lib.Name,
		Version: "local",
		Source:  "local",
		URL:     lib.Path,
		Path:    entry,
		Global:  opts.Global,
	}, nil
}

func findEntryInDir(dir string) string {
	for _, candidate := range []string{"index.lx", "main.lx"} {
		fp := filepath.Join(dir, candidate)
		if st, err := os.Stat(fp); err == nil && !st.IsDir() {
			return fp
		}
	}
	if files, err := os.ReadDir(dir); err == nil {
		for _, f := range files {
			if !f.IsDir() && strings.HasSuffix(f.Name(), ".lx") {
				return filepath.Join(dir, f.Name())
			}
		}
	}
	return ""
}

func installGitHubSource(lib *manifest.Library, opts InstallOptions) (*Module, error) {
	owner, repo, err := parseGitHubURL(lib.URL)
	if err != nil {
		return nil, err
	}
	client := newGitHubClient()

	resolvedVersion, ref, err := resolveVersionForInstall(client, owner, repo, lib.Version)
	if err != nil {
		return nil, err
	}

	root := storeRoot(opts)
	dir := moduleDir(root, lib.Name, resolvedVersion)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("creating module directory: %w", err)
	}

	prog := newInstallProgress(fmt.Sprintf("%s@%s", lib.Name, resolvedVersion), 0)
	fetchErr := client.fetchDirRecursiveProgress(owner, repo, ref, lib.Path, dir, prog)
	if fetchErr != nil {
		resetProgress(prog)
		fetchErr = client.fetchViaTreeAPIProgress(owner, repo, ref, lib.Path, dir, prog)
		if fetchErr != nil {
			resetProgress(prog)
			fetchErr = client.fetchViaCodeload(owner, repo, ref, lib.Path, dir, prog)
			if fetchErr != nil {
				return nil, fmt.Errorf("could not install %s: %w", lib.Name, fetchErr)
			}
		}
	}
	prog.finish()

	entry := findEntryInDir(dir)
	if entry == "" {
		entry = filepath.Join(dir, "index.lx")
	}
	entryRel, _ := filepath.Rel(dir, entry)

	mod := &Module{
		Name:    lib.Name,
		Version: resolvedVersion,
		Source:  fmt.Sprintf("github:%s/%s", owner, repo),
		URL:     lib.URL,
		Path:    entry,
		Global:  opts.Global,
	}
	_ = writeModuleMeta(dir, mod, entryRel)
	if linked, linkErr := linkBinaries(dir, opts); linkErr == nil && len(linked) > 0 {
		printLinkedBins(linked, opts)
	}
	return mod, lockModule(mod, dir)
}

func installGitHubRelease(lib *manifest.Library, opts InstallOptions) (*Module, error) {
	owner, repo, err := parseGitHubURL(lib.URL)
	if err != nil {
		return nil, err
	}
	client := newGitHubClient()

	tag := lib.Release
	if tag == "" || strings.EqualFold(tag, "latest") {
		rel, err := client.LatestRelease(owner, repo)
		if err != nil {
			return nil, fmt.Errorf("fetching latest release for %s: %w", lib.Name, err)
		}
		tag = rel.TagName
	}
	rel, err := client.Release(owner, repo, tag)
	if err != nil {
		return nil, fmt.Errorf("fetching release %s for %s: %w", tag, lib.Name, err)
	}

	version := strings.TrimPrefix(tag, "v")
	root := storeRoot(opts)
	dir := moduleDir(root, lib.Name, version)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("creating module directory: %w", err)
	}

	downloadURL := rel.ZipballURL
	for _, asset := range rel.Assets {
		if strings.HasSuffix(strings.ToLower(asset.Name), ".zip") {
			downloadURL = asset.BrowserDownloadURL
			break
		}
	}
	if downloadURL == "" {
		return nil, fmt.Errorf("release %s for %s has no downloadable archive", tag, lib.Name)
	}

	prog := newInstallProgress(fmt.Sprintf("%s@%s", lib.Name, version), 0)
	if err := client.downloadZipToDir(downloadURL, lib.Path, dir, prog); err != nil {
		return nil, fmt.Errorf("could not install release %s for %s: %w", tag, lib.Name, err)
	}
	prog.finish()

	entry := findEntryInDir(dir)
	if entry == "" {
		entry = filepath.Join(dir, "index.lx")
	}
	entryRel, _ := filepath.Rel(dir, entry)

	mod := &Module{
		Name:    lib.Name,
		Version: version,
		Source:  fmt.Sprintf("github-release:%s/%s", owner, repo),
		URL:     lib.URL,
		Path:    entry,
		Global:  opts.Global,
	}
	_ = writeModuleMeta(dir, mod, entryRel)
	if linked, linkErr := linkBinaries(dir, opts); linkErr == nil && len(linked) > 0 {
		printLinkedBins(linked, opts)
	}
	return mod, lockModule(mod, dir)
}

func resolveVersionForInstall(client *githubClient, owner, repo, versionField string) (resolvedVersion, ref string, err error) {
	constraint := semver.ParseConstraint(versionField)

	if constraint.IsLatest || constraint.Exact != nil || strings.ContainsAny(versionField, "<>^*x") || strings.Contains(versionField, " ") {
		tags, tagErr := client.ListTags(owner, repo)
		if tagErr == nil && len(tags) > 0 {
			var versions []semver.Version
			byVersion := map[string]string{}
			for _, t := range tags {
				v, ok := semver.Parse(t)
				if !ok {
					continue
				}
				versions = append(versions, v)
				byVersion[fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)] = t
			}
			if best, ok := constraint.Best(versions); ok {
				key := fmt.Sprintf("%d.%d.%d", best.Major, best.Minor, best.Patch)
				tag := byVersion[key]
				return key, tag, nil
			}
		}
		if constraint.IsLatest {
			return "main", "main", nil
		}
		return "", "", fmt.Errorf("no tag in %s/%s satisfies version %q", owner, repo, versionField)
	}

	clean := strings.TrimPrefix(versionField, "v")
	if _, ok := semver.Parse(clean); ok {
		return clean, versionField, nil
	}
	if versionField == "" {
		return "main", "main", nil
	}
	return versionField, versionField, nil
}

func resetProgress(p *installProgress) {
	p.done = 0
	p.bytes = 0
	p.total = 0
}

func parseGitHubURL(url string) (owner, repo string, err error) {
	owner, repo, _, _, err = parseGitHubURLWithPath(url)
	return owner, repo, err
}

func parseGitHubURLWithPath(url string) (owner, repo, subPath, ref string, err error) {
	s := strings.TrimSpace(url)
	s = strings.TrimPrefix(s, "https://")
	s = strings.TrimPrefix(s, "http://")
	s = strings.TrimPrefix(s, "www.")
	s = strings.TrimPrefix(s, "github.com/")
	s = strings.TrimSuffix(s, ".git")
	s = strings.TrimSuffix(s, "/")

	parts := strings.Split(s, "/")
	if len(parts) < 2 || parts[0] == "" || parts[1] == "" {
		return "", "", "", "", fmt.Errorf("invalid GitHub URL %q: expected https://github.com/owner/repo", url)
	}
	owner = parts[0]
	repo = parts[1]

	if len(parts) > 3 && (parts[2] == "tree" || parts[2] == "blob") {
		ref = parts[3]
		if len(parts) > 4 {
			subPath = strings.Join(parts[4:], "/")
		}
	}

	return owner, repo, subPath, ref, nil
}

func lockModule(mod *Module, dir string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	lock, err := manifest.LoadLock(cwd)
	if err != nil {
		lock = manifest.NewLock()
	}
	hash, err := hashDir(dir)
	if err != nil {
		hash = ""
	}
	lock.Set(&manifest.LockedModule{
		Name:    mod.Name,
		Version: mod.Version,
		Hash:    hash,
		Source:  mod.Source,
		URL:     mod.URL,
	})
	return manifest.SaveLock(cwd, lock)
}

func hashDir(dir string) (string, error) {
	var paths []string
	err := filepath.WalkDir(dir, func(p string, d os.DirEntry, walkErr error) error {
		if walkErr != nil || d.IsDir() {
			return nil
		}
		if filepath.Base(p) == moduleMetaFile {
			return nil
		}
		paths = append(paths, p)
		return nil
	})
	if err != nil {
		return "", err
	}
	sort.Strings(paths)

	h := sha256.New()
	for _, p := range paths {
		rel, _ := filepath.Rel(dir, p)
		h.Write([]byte(filepath.ToSlash(rel)))
		f, err := os.Open(p)
		if err != nil {
			continue
		}
		_, _ = io.Copy(h, f)
		f.Close()
	}
	return "sha256:" + hex.EncodeToString(h.Sum(nil)), nil
}

func InstallAll(opts InstallOptions) error {
	proj, err := manifest.Load(".")
	if err != nil {
		return err
	}
	for _, lib := range proj.OrderedLibraries() {
		mod, err := InstallLibrary(lib, opts)
		if err != nil {
			return fmt.Errorf("installing %s: %w", lib.Name, err)
		}
		fmt.Printf("installed %s@%s\n", mod.Name, mod.Version)
	}
	return nil
}

func InstallFromSpec(spec string, opts InstallOptions) (*Module, *manifest.Library, error) {
	spec = strings.TrimSpace(spec)
	if spec == "" {
		return nil, nil, fmt.Errorf("empty package spec")
	}

	url := spec
	version := "latest"
	if idx := strings.LastIndex(spec, "@"); idx > strings.Index(spec, "//")+1 && idx > 0 {
		version = spec[idx+1:]
		url = spec[:idx]
	}

	owner, repo, subPath, refFromURL, err := parseGitHubURLWithPath(url)
	if err != nil {
		return nil, nil, err
	}
	if refFromURL != "" && version == "latest" {
		version = refFromURL
	}

	name := repo
	if subPath != "" {
		name = filepath.Base(subPath)
	}

	lib := &manifest.Library{
		Name:    name,
		Source:  "github",
		URL:     fmt.Sprintf("https://github.com/%s/%s", owner, repo),
		Path:    subPath,
		Version: version,
	}
	mod, err := InstallLibrary(lib, opts)
	if err != nil {
		return nil, nil, err
	}
	return mod, lib, nil
}

func List() []Module {
	var mods []Module
	mods = append(mods, listStore(globalModulesRoot(), true)...)
	mods = append(mods, listStore(localModulesRoot(), false)...)
	return mods
}

func listStore(root string, global bool) []Module {
	entries, _ := os.ReadDir(root)
	var mods []Module
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		dir := filepath.Join(root, e.Name())
		if meta, ok := readModuleMeta(dir); ok {
			mods = append(mods, Module{
				Name:    meta.Name,
				Version: meta.Version,
				Source:  meta.Source,
				URL:     meta.URL,
				Path:    filepath.Join(dir, meta.Entry),
				Global:  global,
			})
			continue
		}
		parts := strings.Split(e.Name(), "@")
		if len(parts) == 2 {
			mods = append(mods, Module{
				Name:    strings.ReplaceAll(parts[0], "__", "/"),
				Version: parts[1],
				Global:  global,
			})
		}
	}
	return mods
}

func Remove(name string) error {
	found := false
	for _, root := range []string{localModulesRoot(), globalModulesRoot()} {
		entries, _ := os.ReadDir(root)
		prefix := sanitizeName(name) + "@"
		for _, e := range entries {
			if strings.HasPrefix(e.Name(), prefix) {
				dir := filepath.Join(root, e.Name())
				if entry, ok := readModuleMeta(dir); ok && entry.Entry != "" {
					unlinkBinaries(filepath.Join(dir, entry.Entry))
				}
				if err := os.RemoveAll(dir); err != nil {
					return err
				}
				found = true
			}
		}
	}
	if !found {
		return fmt.Errorf("module %q not found", name)
	}
	return nil
}

type EnvInfo struct {
	GlobalRoot   string
	LocalRoot    string
	GlobalCount  int
	LocalCount   int
	HasManifest  bool
	ManifestPath string
	HasLock      bool
}

func Env() EnvInfo {
	info := EnvInfo{
		GlobalRoot: GlobalRoot(),
		LocalRoot:  LocalRoot(),
	}
	if entries, err := os.ReadDir(globalModulesRoot()); err == nil {
		info.GlobalCount = len(entries)
	}
	if entries, err := os.ReadDir(localModulesRoot()); err == nil {
		info.LocalCount = len(entries)
	}
	if p, ok := manifest.Find(); ok {
		info.HasManifest = true
		info.ManifestPath = p
	}
	if _, err := os.Stat(manifest.LockFile); err == nil {
		info.HasLock = true
	}
	return info
}
