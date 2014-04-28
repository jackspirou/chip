package helper

import "fmt"

// Check for errors
func Check(e error) {
	if e != nil {
		panic(e)
	}
}

//  WRITE BLANKS. Write COUNT blanks to PRINTF, without a newline.
func WriteBlanks(count int) {
	for count > 0 {
		fmt.Printf(" ")
		count -= 1
	}
}
