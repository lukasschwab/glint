package glint

import (
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/packages"
)

const (
	SlowLoadMode = packages.LoadAllSyntax | packages.NeedModule
	FastLoadMode = packages.LoadSyntax | packages.NeedModule
)

var modeName = map[packages.LoadMode]string{
	SlowLoadMode: "SlowLoadMode",
	FastLoadMode: "FastLoadMode",
}

func LoadMode(a *analysis.Analyzer) packages.LoadMode {
	if !needFacts(a) {
		return FastLoadMode
	}
	return SlowLoadMode
}

var seen = map[*analysis.Analyzer]bool{}

// needFacts reports whether any analysis required by the specified set
// needs facts. If so, we must load the entire program from source.
// https://cs.opensource.google/go/x/tools/+/refs/tags/v0.29.0:go/analysis/internal/checker/checker.go;l=451-469
func needFacts(as *analysis.Analyzer) bool {
	q := []*analysis.Analyzer{as} // for BFS
	for len(q) > 0 {
		a := q[0]
		q = q[1:]
		if !seen[a] {
			seen[a] = true
			if len(a.FactTypes) > 0 {
				return true
			}
			q = append(q, a.Requires...)
		}
	}
	return false
}
