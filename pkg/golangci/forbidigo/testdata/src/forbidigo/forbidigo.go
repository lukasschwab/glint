package forbidigo

import "fmt"

func printMessage() {
	fmt.Println("message") // want "use of `fmt.Println` forbidden because .use structured logging."
}
