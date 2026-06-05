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

// Slices are not comparable. The batch checker must reject slice == / != just as
// the streaming checker (chip run) does: identical slice types are
// types.Identical, so before this rule the batch path accepted the comparison
// and the VM then compared the values' raw numeric words, silently returning a
// wrong boolean. The message matches the streaming checker so chip check and
// chip run agree.
func TestSliceEqualityRejected(t *testing.T) {
	for _, op := range []string{"==", "!="} {
		wantErr(t, `package main
func main() {
    a := []int{1}
    b := []int{1}
    print(a `+op+` b)
}`, "slices are not comparable")
	}
}

// A slice compared against a scalar is likewise rejected, not reported as a mere
// type mismatch — the slice operand alone makes the comparison ill-formed.
func TestSliceScalarEqualityRejected(t *testing.T) {
	wantErr(t, `package main
func main() {
    a := []int{1}
    print(a == 5)
}`, "slices are not comparable")
}

// Comparing ordinary comparable values still type-checks: the new slice guard
// must not disturb scalar equality.
func TestScalarEqualityOK(t *testing.T) {
	mustCheck(t, `package main
func main() {
    print(1 == 2)
    print("a" != "b")
}`)
}
