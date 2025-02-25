package cache

import (
	"log"
	"testing"
)

type Logger interface {
	Log(string)
	Logf(string, ...interface{})
}

type TestLogger struct{ *testing.T }

func (tl TestLogger) Log(s string) {
	tl.T.Helper()
	tl.T.Log(s)
}

func (tl TestLogger) Logf(s string, args ...interface{}) {
	tl.T.Helper()
	tl.T.Logf(s, args...)
}

type NoopLogger struct{}

func (nl NoopLogger) Log(s string) {}

func (nl NoopLogger) Logf(s string, args ...interface{}) {}

type LogLogger struct{}

func (ll LogLogger) Log(s string) {
	log.Println(s)
}

func (ll LogLogger) Logf(s string, args ...interface{}) {
	log.Printf(s, args...)
}
