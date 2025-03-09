package main

import (
	"github.com/lukasschwab/glint"
	"github.com/lukasschwab/glint/pkg/golangci"
)

func main() {
	glint.Main(golangci.DefaultAnalyzers()...)
}
