chip [![GoDoc](http://godoc.org/github.com/jackspirou/chip?status.png)](http://godoc.org/github.com/jackspirou/chip) [![Build Status](https://travis-ci.org/jackspirou/chip.svg?branch=master)](https://travis-ci.org/jackspirou/chip) [![Go Report Card](http://goreportcard.com/badge/jackspirou/chip)](http://goreportcard.com/report/jackspirou/chip)
====
Chip is a toy systems scripting language.

Motivation
----------
Long ago I wrote a compiler in Java. It was for a two-part college compilers
course series. The language was known as SNARL and the compiler output was MIPS
assembly (asm) code. Since we had no MIPS machines readily available, the
asm was then ported to a MIPS emulator.

While the SNARL compiler was a simple toy for academic purposes, I noticed that
the simplicity of it's design provided powerful foundations to explore further.
A couple years later, I stumbled upon Golang and it reminded me of the same
simplicity of SNARL. It was refreshing after writing lots of Java and C++.

It is important to note that while Golang is lexically simple, it's runtime
(GC, green threads), CSP design, optimizations, and available target hardware
architecture implementations are not trivial. Simple is not trivial.

Anyway, excited that Golang's spirit of lexical simplicity seemed to support
the value I saw in SNARL, I was inspired to try writing a toy scripting
language that was equal with, or exceeded the lexically simplicity of Go. It
seemed obvious to leverage Golang for the implementation of this idea.

With Go as a guide, I want to produce a toy scripting language that has extreme
minimal syntax. I also want the Go implementation to be idiomatic.

This project is not a race, but a labor of love.
