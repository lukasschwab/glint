package testdata

import "fmt"

// Should trigger some gosimple fix.
func example() {
	// Simplifiable code
	var x int
	x = 0
	fmt.Println(x)

	// Simplifiable loop
	for i := 0; i < len([]int{1, 2, 3}); i++ {
		fmt.Println(i)
	}

	// Simplifiable if statement
	if true {
		fmt.Println("This is always true")
	}
}
