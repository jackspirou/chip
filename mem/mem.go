// Package mem is a singleton packge that manages memory.
package mem

var ints []int
var strings []string

// SetInt sets the value of integer register in a slice of memory.
func SetInt(reg, i int) {
	ints[reg] = i
}

// SetString sets the value of a string register in a slice of memory.
func SetString(reg int, s string) {
	strings[reg] = s
}

// Int retrieves the value of an integer register in a slice of memory.
func Int(reg int) int {
	return ints[reg]
}

// String retrieves the value of an string register in a slice of memory.
func String(reg int) string {
	return strings[reg]
}
