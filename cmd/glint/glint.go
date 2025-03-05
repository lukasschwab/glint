package main

import (
	"log"

	"github.com/lukasschwab/glint"
	"github.com/lukasschwab/glint/internal/golangci"

	nilinterface "github.com/lukasschwab/nilinterface/pkg/analyzer"
)

func main() {
	// TODO: parameterize some of these?
	glint.Main(LogLogger{}, false, append(
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

type LogLogger struct{}

func (ll LogLogger) Log(s string) {
	log.Println(s)
}
