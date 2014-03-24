package support

// Check for errors
func Check(e error) {
	if e != nil {
		panic(e)
	}
}

//  Constant Flags & Globals
const (
	tracing = false    //  Enable or disable ENTER/EXIT.
	EOFChar = '\u0000' //  End of source sentinel.
)
