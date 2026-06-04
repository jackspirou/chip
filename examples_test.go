package chip_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/jackspirou/chip"
)

// TestExamples runs every program in examples/ and checks its exact output, so
// the examples stay correct and verifiable. Every .chp file must have an
// expected-output entry here, or the test fails — no example goes unchecked.
func TestExamples(t *testing.T) {
	want := map[string]string{
		"hello.chp":             "Hello, chip\n",
		"gcd.chp":               "21\n",
		"fibonacci.chp":         "55\n",
		"mutual_recursion.chp":  "1\n",
		"forward_reference.chp": "streamed!\n",
		"arrays.chp":            "4\n20\n",
	}

	files, err := filepath.Glob("examples/*.chp")
	if err != nil {
		t.Fatal(err)
	}
	if len(files) == 0 {
		t.Fatal("no example programs found")
	}

	for _, file := range files {
		name := filepath.Base(file)
		t.Run(name, func(t *testing.T) {
			expected, ok := want[name]
			if !ok {
				t.Fatalf("example %s has no expected output in examples_test.go", name)
			}
			src, err := os.ReadFile(file)
			if err != nil {
				t.Fatal(err)
			}
			var out bytes.Buffer
			if err := chip.Run(src, &out); err != nil {
				t.Fatalf("chip.Run(%s): %v", name, err)
			}
			if got := out.String(); got != expected {
				t.Fatalf("%s output = %q, want %q", name, got, expected)
			}
		})
	}
}
