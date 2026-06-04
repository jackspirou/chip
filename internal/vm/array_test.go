package vm_test

import (
	"strings"
	"testing"
)

func TestArrayLiteralAndIndex(t *testing.T) {
	got := mustRun(t, `package main
func main() {
    xs := []int{3, 1, 4}
    print(xs)
    print(xs[0])
    print(xs[2])
    print(len(xs))
}`)
	want := "[3 1 4]\n3\n4\n3\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestArrayIndexAssign(t *testing.T) {
	got := mustRun(t, `package main
func main() {
    xs := []int{1, 2, 3}
    xs[1] = 99
    print(xs[1])
    print(xs)
}`)
	want := "99\n[1 99 3]\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestArraySumLoop(t *testing.T) {
	got := mustRun(t, `package main
func main() { print(sum([]int{3, 1, 4, 1, 5})) }
func sum(xs []int) int {
    total := 0
    i := 0
    for i < len(xs) {
        total = total + xs[i]
        i = i + 1
    }
    return total
}`)
	if got != "14\n" {
		t.Errorf("got %q, want %q", got, "14\n")
	}
}

func TestStringConcatAndLen(t *testing.T) {
	got := mustRun(t, `package main
func main() {
    s := "abc" + "de"
    print(s)
    print(len(s))
}`)
	want := "abcde\n5\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestIndexOutOfRange(t *testing.T) {
	_, err := runProgram(`package main
func main() { print([]int{1, 2}[5]) }`)
	if err == nil || !strings.Contains(err.Error(), "index out of range") {
		t.Fatalf("expected index out of range error, got %v", err)
	}
}
