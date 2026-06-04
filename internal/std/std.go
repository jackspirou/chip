// Package std provides chip's standard library: helpers written in chip itself
// and loaded into the engine before every program runs. This keeps the language
// minimal — only print and len are built in — and lets the library grow in chip
// rather than in Go.
package std

import _ "embed"

// Prelude is the chip source of the standard library. The streaming engine feeds
// it before user code, so its functions are available to every program.
//
//go:embed prelude.chp
var Prelude string

//go:embed math.chp
var mathSrc string

// packages holds chip's built-in library packages — written in chip and
// resolved by import path. Unlike the flat prelude (loaded unqualified into
// every program), these are imported explicitly (e.g. import "math") and their
// exported members are called qualified (math.Gcd). They are single-file for
// now.
var packages = map[string]string{
	"math": mathSrc,
}

// Package returns the chip source of the built-in package at importPath and
// reports whether one exists. The streaming loader consults this ahead of the
// filesystem, so a built-in package is never shadowed by a local directory of
// the same import path (Go-style).
func Package(importPath string) (string, bool) {
	src, ok := packages[importPath]
	return src, ok
}
