package glint

import (
	"slices"
	"strings"
	"testing"
)

func TestTestFlagDoesNotSelectTestsAnalyzer(t *testing.T) {
	t.Parallel()

	for _, flag := range []string{"-test", "-test=true"} {
		options, err := parseUserOptions([]string{flag, "./..."})
		if err != nil {
			t.Fatalf("parseUserOptions(%q): %v", flag, err)
		}
		if slices.Contains(options.args, "-tests=true") {
			t.Fatalf("parseUserOptions(%q) selected only the tests analyzer", flag)
		}
		if !slices.Equal(options.args, []string{"./..."}) {
			t.Fatalf("parseUserOptions(%q) args = %q", flag, options.args)
		}
	}
}

func TestTestFalseIsRejected(t *testing.T) {
	t.Parallel()

	_, err := parseUserOptions([]string{"-test=false", "./..."})
	if err == nil || !strings.Contains(err.Error(), "not supported") {
		t.Fatalf("parseUserOptions(-test=false) error = %v", err)
	}
}
