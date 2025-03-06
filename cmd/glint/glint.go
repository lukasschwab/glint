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
