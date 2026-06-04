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
