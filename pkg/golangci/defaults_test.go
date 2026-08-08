package golangci

import (
	"fmt"
	"testing"
)

func TestDefaultAnalyzersCoverGolangCIDefaultFamilies(t *testing.T) {
	want := map[string]bool{
		"hostport":  true,
		"waitgroup": true,
	}
	for i := 1; i <= 12; i++ {
		want[fmt.Sprintf("QF%04d", 1000+i)] = true
	}
	for _, name := range []string{
		"ST1001", "ST1005", "ST1006", "ST1008", "ST1011", "ST1012",
		"ST1013", "ST1015", "ST1017", "ST1018", "ST1019", "ST1023",
	} {
		want[name] = true
	}

	seen := make(map[string]bool)
	for _, analyzer := range DefaultAnalyzers() {
		if seen[analyzer.Name] {
			t.Errorf("duplicate analyzer %q", analyzer.Name)
		}
		seen[analyzer.Name] = true
	}

	for name := range want {
		if !seen[name] {
			t.Errorf("default analyzers do not include %q", name)
		}
	}
	for name := range golangciDisabledStyleAnalyzers {
		if seen[name] {
			t.Errorf("default analyzers unexpectedly include %q", name)
		}
	}
	if seen["loopclosure"] {
		t.Error("default analyzers unexpectedly include loopclosure")
	}

	if got, want := len(seen), 191; got != want {
		t.Errorf("got %d default analyzers, want %d", got, want)
	}
}
