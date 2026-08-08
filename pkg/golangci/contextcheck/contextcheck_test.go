package contextcheck

import (
	"go/types"
	"testing"

	"golang.org/x/tools/go/analysis"
)

func TestNewMatchesGolangCIPrerequisites(t *testing.T) {
	analyzer := New("github.com/example/project")
	wantRequires := map[string]bool{
		"buildssa": false,
		"ctrlflow": false,
		"inspect":  false,
	}
	for _, required := range analyzer.Requires {
		if _, ok := wantRequires[required.Name]; ok {
			wantRequires[required.Name] = true
		}
	}
	for name, seen := range wantRequires {
		if !seen {
			t.Errorf("contextcheck does not require %s", name)
		}
	}
	if len(analyzer.FactTypes) == 0 {
		t.Error("contextcheck does not export facts")
	}
	if _, err := analyzer.Run(&analysis.Pass{
		Analyzer: analyzer,
		Pkg:      types.NewPackage("example.com/external", "external"),
	}); err != nil {
		t.Fatalf("skip external package: %v", err)
	}
}
