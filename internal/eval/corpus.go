package eval

// Corpus is the fixed set of broken→fixed repair scenarios the harness measures
// against. Each case is one small program with one planted defect, chosen to cover
// a distinct diagnostic class and a distinct repair: an undefined name (NAM001), a
// return-type mismatch (TYP001), a missing return (TYP002), a parse error (PAR001),
// a wrong argument count (TYP003), and an undefined type (NAM002). Every Broken
// faults and every Fixed runs clean and prints Want — verified against the built
// binary, and re-asserted by TestCorpusBrokenFailsFixedSolves so the corpus can't
// rot silently.
//
// The missing-return case (TYP002) is a function with a declared result that can
// fall off the end. Both checkers reject it — `chip run` faults and `chip check`
// reports it too, since the termination rule is shared via internal/typerules — so
// it is caught under either feedback mode. (The check-silent fallback the validator
// still provides is exercised by genuinely runtime-only faults, e.g. a division by
// zero, which no static checker can catch.) The validator always decides "solved"
// by the run alone, chip's runtime being the ground truth.
//
// The undefined-type case (NAM002) is the one the per-model feedback comparison
// (§5.5) turns on. chip's types are int/float/string/bool; a model that knows other
// languages, given only "undefined type: int64", repairs by analogy — to "i64",
// still undefined — and burns another turn. The grounded "llm" feedback names chip's
// actual types, so the model reaches "int" in one. Only the first occurrence is
// reported, so a model that does not generalize the fix re-faults on the second.
func Corpus() []Case {
	return []Case{
		{
			Name: "undefined-name",
			Broken: `func main() {
    answer := 42
    print(anser)
}
`,
			Fixed: `func main() {
    answer := 42
    print(answer)
}
`,
			Want: "42\n",
		},
		{
			Name: "return-type-mismatch",
			Broken: `func nine() int {
    return "nine"
}

func main() {
    print(nine())
}
`,
			Fixed: `func nine() int {
    return 9
}

func main() {
    print(nine())
}
`,
			Want: "9\n",
		},
		{
			Name: "missing-return",
			Broken: `func pos(n int) int {
    if n > 0 {
        return 1
    }
}

func main() {
    print(pos(5))
}
`,
			Fixed: `func pos(n int) int {
    if n > 0 {
        return 1
    }
    return 0
}

func main() {
    print(pos(5))
}
`,
			Want: "1\n",
		},
		{
			Name: "parse-error",
			Broken: `func main() {
    print(1 +)
}
`,
			Fixed: `func main() {
    print(1 + 2)
}
`,
			Want: "3\n",
		},
		{
			Name: "wrong-arg-count",
			Broken: `func add(a int, b int) int {
    return a + b
}

func main() {
    print(add(1))
}
`,
			Fixed: `func add(a int, b int) int {
    return a + b
}

func main() {
    print(add(1, 2))
}
`,
			Want: "3\n",
		},
		{
			Name: "undefined-type",
			Broken: `func triple(n int64) int64 {
    return n * 3
}

func main() {
    print(triple(7))
}
`,
			Fixed: `func triple(n int) int {
    return n * 3
}

func main() {
    print(triple(7))
}
`,
			Want: "21\n",
		},
	}
}
