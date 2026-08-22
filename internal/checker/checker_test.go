package checker

import (
	"os"
	"path/filepath"
	"testing"
)

func TestProductionChecker(t *testing.T) {
	dir := t.TempDir()
	mod := filepath.Join(dir, "math.lx")
	if err := os.WriteFile(mod, []byte("fn add(a, b) { a + b }\n"), 0644); err != nil {
		t.Fatal(err)
	}
	loader := func(path, from string) (string, string, bool) {
		p := path
		if !filepath.IsAbs(p) {
			p = filepath.Join(filepath.Dir(from), p)
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return "", p, false
		}
		return string(b), p, true
	}
	good := `val math = @fimport("./math.lx")
fn main() {
  val x = math.add(1, 2)
  log(x)
}`
	r := New(Options{Loader: loader}).Check(good, filepath.Join(dir, "main.lx"))
	if len(r.Diagnostics) != 0 {
		t.Fatalf("good source diagnostics: %+v", r.Diagnostics)
	}

	bad := `val math = @fimport("./math.lx")
fn main() {
  val x = math.missing(1, 2)
  nope(x)
}`
	r = New(Options{Loader: loader}).Check(bad, filepath.Join(dir, "main2.lx"))
	if len(r.Diagnostics) < 2 {
		t.Fatalf("expected import member + unresolved symbol errors, got %+v", r.Diagnostics)
	}
}

func TestProductionCheckerRules(t *testing.T) {
	dir := t.TempDir()
	loader := func(path, from string) (string, string, bool) {
		p := path
		if !filepath.IsAbs(p) {
			p = filepath.Join(filepath.Dir(from), p)
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return "", p, false
		}
		return string(b), p, true
	}
	cases := []struct{ name, src, code string }{
		{"missing main", `val x = 1`, "E0601"},
		{"top level call", `log("x")
fn main() {}`, "E0071"},
		{"const assignment", `val x = 1
fn main() { x = 2 }`, "E0594"},
		{"arity", `fn add(a, b) { a + b }
fn main() { add(1) }`, "E0061"},
		{"duplicate", `val x = 1
val x = 2
fn main() {}`, "E0428"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := New(Options{Loader: loader}).Check(tc.src, filepath.Join(dir, tc.name+".lx"))
			found := false
			for _, d := range r.Diagnostics {
				if d.Code == tc.code {
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("expected %s, got %+v", tc.code, r.Diagnostics)
			}
		})
	}
	mathPath := filepath.Join(dir, "math.lx")
	if err := os.WriteFile(mathPath, []byte("fn add(a, b) { a + b }\n"), 0644); err != nil {
		t.Fatal(err)
	}
	r := New(Options{Loader: loader}).Check(`import { add } from "./math.lx"
fn main() { add(1, 2) }`, filepath.Join(dir, "import.lx"))
	if len(r.Diagnostics) != 0 {
		t.Fatalf("import syntax diagnostics: %+v", r.Diagnostics)
	}
}

func TestKnownStdMembers(t *testing.T) {
	std := map[string]map[string]struct{}{"io": {"log": {}, "read": {}}}
	good := `val io = @import("std.io")
fn main() { io.log("ok") }`
	if r := New(Options{KnownModules: std, CheckStdMembers: true}).Check(good, "main.lx"); len(r.Diagnostics) != 0 {
		t.Fatalf("std member false positive: %+v", r.Diagnostics)
	}
	bad := `val io = @import("std.io")
fn main() { io.missing("ok") }`
	r := New(Options{KnownModules: std, CheckStdMembers: true}).Check(bad, "main.lx")
	found := false
	for _, d := range r.Diagnostics {
		if d.Code == "E0599" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected std member error, got %+v", r.Diagnostics)
	}
}
