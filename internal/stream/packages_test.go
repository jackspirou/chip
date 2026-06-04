package stream

import (
	"bytes"
	"strings"
	"testing"
)

// runPkg streams main through the engine with an in-memory loader holding the
// given packages (import path → source), capturing stdout and the run error.
func runPkg(t *testing.T, main string, pkgs map[string]string) (string, error) {
	t.Helper()
	m := make(map[string][]Source, len(pkgs))
	for path, src := range pkgs {
		m[path] = []Source{{Name: pkgBaseName(path) + ".chp", Data: []byte(src)}}
	}
	var out bytes.Buffer
	err := RunWithLoader(strings.NewReader(main), &out, MapLoader(m))
	return out.String(), err
}

const geometrySrc = "package geometry\nfunc Area(w int, h int) int { return w * h }\n"

// PE1 — import a package and call a qualified function: geometry.Area(3, 4) is 12.
func TestPE1ImportAndQualifiedCall(t *testing.T) {
	out, err := runPkg(t,
		"import \"geometry\"\nfunc main() { print(geometry.Area(3, 4)) }\n",
		map[string]string{"geometry": geometrySrc})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "12\n" {
		t.Fatalf("stdout = %q, want %q", out, "12\n")
	}
}

// An import alias rebinds the reference name: g.Area works when imported as g.
func TestImportAlias(t *testing.T) {
	out, err := runPkg(t,
		"import g \"geometry\"\nfunc main() { print(g.Area(3, 4)) }\n",
		map[string]string{"geometry": geometrySrc})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "12\n" {
		t.Fatalf("stdout = %q, want %q", out, "12\n")
	}
}

// PE2 — using a package that was never imported is an error naming the package.
func TestPE2PackageNotImported(t *testing.T) {
	_, err := runPkg(t,
		"func main() { print(geometry.Area(3, 4)) }\n",
		map[string]string{"geometry": geometrySrc})
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	if got := err.Error(); !strings.Contains(got, "undefined: geometry") {
		t.Fatalf("error = %q, want it to contain %q", got, "undefined: geometry")
	}
}

// PE3 — calling a member the package does not define is an error naming it.
func TestPE3UndefinedMember(t *testing.T) {
	_, err := runPkg(t,
		"import \"geometry\"\nfunc main() { print(geometry.Nope()) }\n",
		map[string]string{"geometry": geometrySrc})
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	if got := err.Error(); !strings.Contains(got, "undefined: geometry.Nope") {
		t.Fatalf("error = %q, want it to contain %q", got, "undefined: geometry.Nope")
	}
}

// An imported package is loaded as a unit (collect-then-check), so a function in
// it may reference another defined later in the same file.
func TestImportedPackageForwardRef(t *testing.T) {
	pkg := "package mathx\nfunc Twice(n int) int { return Plus(n, n) }\nfunc Plus(a int, b int) int { return a + b }\n"
	out, err := runPkg(t,
		"import \"mathx\"\nfunc main() { print(mathx.Twice(21)) }\n",
		map[string]string{"mathx": pkg})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "42\n" {
		t.Fatalf("stdout = %q, want %q", out, "42\n")
	}
}

// An imported package may itself use the flat prelude (unqualified resolution
// falls through to it).
func TestImportedPackageUsesPrelude(t *testing.T) {
	pkg := "package box\nfunc Width(a int, b int) int { return max(a, b) }\n"
	out, err := runPkg(t,
		"import \"box\"\nfunc main() { print(box.Width(3, 7)) }\n",
		map[string]string{"box": pkg})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "7\n" {
		t.Fatalf("stdout = %q, want %q", out, "7\n")
	}
}

// PE7 — an import cycle is detected during eager loading and reported, naming
// the path back to the offending package.
func TestPE7ImportCycle(t *testing.T) {
	_, err := runPkg(t,
		"import \"a\"\nfunc main() { print(a.A()) }\n",
		map[string]string{
			"a": "package a\nimport \"b\"\nfunc A() int { return b.B() }\n",
			"b": "package b\nimport \"a\"\nfunc B() int { return a.A() }\n",
		})
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	if got := err.Error(); !strings.Contains(got, "import cycle: a -> b -> a") {
		t.Fatalf("error = %q, want it to contain %q", got, "import cycle: a -> b -> a")
	}
}

// PE6 — a package spread over two files loads as a unit: a function in one file
// calls a function in the other (collect-then-check resolves the cross-file
// reference), and the importer runs it.
func TestPE6MultiFilePackage(t *testing.T) {
	m := map[string][]Source{
		"geometry": {
			{Name: "area.chp", Data: []byte("package geometry\nfunc Area(w int, h int) int { return scale(w * h) }\n")},
			{Name: "scale.chp", Data: []byte("package geometry\nfunc scale(n int) int { return n }\n")},
		},
	}
	var out bytes.Buffer
	src := "import \"geometry\"\nfunc main() { print(geometry.Area(3, 4)) }\n"
	if err := RunWithLoader(strings.NewReader(src), &out, MapLoader(m)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := out.String(); got != "12\n" {
		t.Fatalf("stdout = %q, want %q", got, "12\n")
	}
}

// Files of one package must agree on the package name.
func TestPackageNameMismatch(t *testing.T) {
	m := map[string][]Source{
		"geometry": {
			{Name: "a.chp", Data: []byte("package geometry\nfunc A() int { return 1 }\n")},
			{Name: "b.chp", Data: []byte("package geom\nfunc B() int { return 2 }\n")},
		},
	}
	var out bytes.Buffer
	src := "import \"geometry\"\nfunc main() { print(geometry.A()) }\n"
	err := RunWithLoader(strings.NewReader(src), &out, MapLoader(m))
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	if got := err.Error(); !strings.Contains(got, "package name mismatch") {
		t.Fatalf("error = %q, want it to contain %q", got, "package name mismatch")
	}
}

// A subpath import refers by the declared package name: import "lib/geometry"
// (declaring package geometry) is referred to as geometry.
func TestSubpathImport(t *testing.T) {
	out, err := runPkg(t,
		"import \"lib/geometry\"\nfunc main() { print(geometry.Area(3, 4)) }\n",
		map[string]string{"lib/geometry": geometrySrc})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "12\n" {
		t.Fatalf("stdout = %q, want %q", out, "12\n")
	}
}

// PE5 — accessing a package-private (lowercase) member across an import is an
// error naming the unexported member and its package.
func TestPE5UnexportedAccess(t *testing.T) {
	_, err := runPkg(t,
		"import \"geometry\"\nfunc main() { print(geometry.area(3, 4)) }\n",
		map[string]string{"geometry": "package geometry\nfunc area(w int, h int) int { return w * h }\n"})
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	if got := err.Error(); !strings.Contains(got, "area is not exported by geometry") {
		t.Fatalf("error = %q, want it to contain %q", got, "area is not exported by geometry")
	}
}

// Within a package every name is visible: an exported function may call an
// unexported one in the same package, and the importer reaches it through the
// exported wrapper.
func TestUnexportedCallWithinPackage(t *testing.T) {
	pkg := "package geometry\nfunc Area(w int, h int) int { return area(w, h) }\nfunc area(w int, h int) int { return w * h }\n"
	out, err := runPkg(t,
		"import \"geometry\"\nfunc main() { print(geometry.Area(3, 4)) }\n",
		map[string]string{"geometry": pkg})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "12\n" {
		t.Fatalf("stdout = %q, want %q", out, "12\n")
	}
}

// A bare package name used as a value (not as a qualified call) is rejected,
// distinctly from an undefined variable.
func TestBarePackageNameAsValue(t *testing.T) {
	_, err := runPkg(t,
		"import \"geometry\"\nfunc main() { x := geometry\nprint(x) }\n",
		map[string]string{"geometry": geometrySrc})
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	if got := err.Error(); !strings.Contains(got, "geometry is a package, not a value") {
		t.Fatalf("error = %q, want it to contain %q", got, "geometry is a package, not a value")
	}
}

// Importing a package that does not exist reports a clear error.
func TestImportNotFound(t *testing.T) {
	_, err := runPkg(t,
		"import \"missing\"\nfunc main() {}\n",
		map[string]string{})
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	if got := err.Error(); !strings.Contains(got, "cannot import") {
		t.Fatalf("error = %q, want it to contain %q", got, "cannot import")
	}
}
