package a

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert" // want "replace with peterldowns/testy"
)

func main() {
	fmt.Println("Hello, world!")
	assert.NoError(new(testing.T), nil)
}
