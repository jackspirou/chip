package check_test

import "testing"

func TestArrayOK(t *testing.T) {
	mustCheck(t, `package main
func main() {
    xs := []int{1, 2, 3}
    xs[0] = xs[1] + xs[2]
    print(len(xs))
}`)
}

func TestArrayElemTypeMismatch(t *testing.T) {
	wantErr(t, `package main
func main() { xs := []int{1, "two"} }`, "cannot use string as int in array literal")
}

func TestIndexNonSlice(t *testing.T) {
	wantErr(t, `package main
func main() {
    x := 5
    print(x[0])
}`, "cannot index int")
}

func TestLenWrongType(t *testing.T) {
	wantErr(t, `package main
func main() { print(len(5)) }`, "len expects a slice or string")
}

func TestFixedArrayUnsupported(t *testing.T) {
	wantErr(t, `package main
func main() { xs := [3]int{1, 2, 3} }`, "fixed-size arrays are not supported")
}
