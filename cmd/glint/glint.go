package main

import (
	"github.com/lukasschwab/glint"
	"github.com/lukasschwab/glint/pkg/golangci"
	"github.com/lukasschwab/glint/pkg/testify"
	nilinterface "github.com/lukasschwab/nilinterface/pkg/analyzer"
)

func main() {
	glint.Main(append(
		golangci.DefaultAnalyzers(),
		nilinterface.Analyzer,
		testify.Analyzer,
	)...)
}
