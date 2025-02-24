package main

import (
	"github.com/lukasschwab/glint"
	"github.com/lukasschwab/glint/pkg/golangci"

	nilinterface "github.com/lukasschwab/nilinterface/pkg/analyzer"
)

func main() {
	glint.Main(append(
		golangci.DefaultAnalyzers(),
		nilinterface.Analyzer,
	)...)
}

// This is junk code checking nilinterface is running as expected.
func noop(a any) {}

func test() {
	// NOTE: I may want this to *not* be an error; see bc0099.
	noop(nil) // want "nil passed to interface parameter"
}
