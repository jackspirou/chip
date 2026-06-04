package stream

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
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

// DirLoader returns a Loader that reads packages from the filesystem rooted at
// base. For Slice 1 a package "name" is the single file <base>/name.chp; Slice 3
// generalizes this to a directory of files.
func DirLoader(base string) Loader { return dirLoader{base: base} }

type dirLoader struct{ base string }

func (d dirLoader) Load(importPath string) ([]Source, error) {
	file := filepath.Join(d.base, filepath.FromSlash(importPath)+".chp")
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	return []Source{{Name: path.Base(importPath) + ".chp", Data: data}}, nil
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
