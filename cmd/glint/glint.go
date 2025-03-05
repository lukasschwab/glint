package main

import (
	"log"

	"github.com/lukasschwab/glint"
	"github.com/lukasschwab/glint/pkg/golangci"
	nilinterface "github.com/lukasschwab/nilinterface/pkg/analyzer"
)

func main() {
	glint.Main(LogLogger{}, append(
		golangci.DefaultAnalyzers(),
		nilinterface.Analyzer,
	)...)
}

type LogLogger struct{}

func (ll LogLogger) Log(s string) {
	log.Println(s)
}
