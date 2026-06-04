package stream

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/jackspirou/chip/internal/std"
)

// Source is one file of a package: its name (for ordering and diagnostics) and
// its raw bytes.
type Source struct {
	Name string
	Data []byte
}

// Loader resolves an import path to the source files that make up that package.
// It is the seam that decouples the engine from the filesystem so imports can
// be exercised hermetically in tests (plan §3.4).
type Loader interface {
	// Load returns the source files of the package at importPath, ordered by
	// file name, or an error (e.g. the package was not found).
	Load(importPath string) ([]Source, error)
}

// stdLoader resolves chip's built-in standard-library packages (written in chip
// and bundled into the binary) ahead of the filesystem: import "math" always
// finds the built-in package, Go-style, and is never shadowed by a local
// directory of the same import path. Any path the standard library does not
// provide falls through to the wrapped user loader. The engine wraps every
// loader with one of these (newEngine), so all run paths see the stdlib.
type stdLoader struct{ user Loader }

func (s stdLoader) Load(importPath string) ([]Source, error) {
	if src, ok := std.Package(importPath); ok {
		return []Source{{Name: path.Base(importPath) + ".chp", Data: []byte(src)}}, nil
	}
	return s.user.Load(importPath)
}

// DirLoader returns a Loader that reads packages from the filesystem rooted at
// base. A package "name" is the directory <base>/name (every .chp file in it,
// ordered by file name); a subpath like "lib/geometry" nests accordingly. As a
// fallback a single-file package <base>/name.chp is also accepted — the
// directory form wins when both exist.
func DirLoader(base string) Loader { return dirLoader{base: base} }

type dirLoader struct{ base string }

func (d dirLoader) Load(importPath string) ([]Source, error) {
	dir := filepath.Join(d.base, filepath.FromSlash(importPath))
	if info, err := os.Stat(dir); err == nil && info.IsDir() {
		return loadDir(dir)
	}
	// Fall back to a single-file package: <base>/importPath.chp.
	data, err := os.ReadFile(dir + ".chp")
	if err != nil {
		return nil, fmt.Errorf("package not found: %s", importPath)
	}
	return []Source{{Name: path.Base(importPath) + ".chp", Data: data}}, nil
}

// loadDir reads every .chp file in dir as the files of one package, ordered by
// file name (os.ReadDir sorts its entries).
func loadDir(dir string) ([]Source, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var srcs []Source
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".chp") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return nil, err
		}
		srcs = append(srcs, Source{Name: e.Name(), Data: data})
	}
	if len(srcs) == 0 {
		return nil, fmt.Errorf("package has no .chp files: %s", dir)
	}
	return srcs, nil
}

// MapLoader returns a Loader backed by an in-memory map of import path → files,
// for fast, hermetic tests with no temp files.
func MapLoader(m map[string][]Source) Loader { return mapLoader(m) }

type mapLoader map[string][]Source

func (m mapLoader) Load(importPath string) ([]Source, error) {
	srcs, ok := m[importPath]
	if !ok {
		return nil, fmt.Errorf("package not found: %s", importPath)
	}
	return srcs, nil
}

// pkgBaseName is the default reference name for an import path: its last path
// element. It is the fallback when an imported file has no package clause.
func pkgBaseName(importPath string) string { return path.Base(importPath) }

// isExported reports whether a top-level name is exported — visible across an
// import — which (Go-style, P3) holds exactly when its first rune is uppercase.
// Capitalization only matters across an import; within a package every name is
// visible.
func isExported(name string) bool {
	r, _ := utf8.DecodeRuneInString(name)
	return unicode.IsUpper(r)
}
