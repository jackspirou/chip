chip [![GoDoc](http://godoc.org/github.com/jackspirou/chip?status.png)](http://godoc.org/github.com/jackspirou/chip) [![Build Status](https://travis-ci.org/jackspirou/chip.svg?branch=master)](https://travis-ci.org/jackspirou/chip) [![Go Report Card](http://goreportcard.com/badge/jackspirou/chip)](http://goreportcard.com/report/jackspirou/chip)
====
Chip is a toy systems scripting language.

Motivation
----------
Long ago I wrote a compiler in college in Java. It was for a two series
compilers course. The language was known as SNARL and the compiler output was
MIPS assembly code. The MIPS asm then was manually ran on a virtual machine.
While the SNARL compiler was a simple toy for academic purposes, I noticed the
simplicity of it's design provided powerful foundations to explore.  

A couple years later, I stumbled upon Golang and I was reminded of the same
simplicity of SNARL. It was refreshing having dealt with so much Java, and C++.

It is important to note that while Golang is lexically simple, as a compiled
language it  holds tremendous complexity around it's runtime
(GC, green threads), CSP, optimizations, and available target architectures.

Excited by Golang's spirit of simplicity, the same lexically as SNARL, I was
inspired to try writing a lexically simple toy scripting language. It seemed
obvious to leverage Golang to implement this experimental language.

With Go as a guide, I want to make a lexically simple and beautify toy scripting
language that has extreme minimal syntax. I also want the implementation to be
as idiomatic as possible.

This project is not a race, it is experimental and a labor of love.
