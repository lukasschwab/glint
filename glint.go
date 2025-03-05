package glint

import (
	"flag"
	"fmt"
	"time"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/checker"
	"golang.org/x/tools/go/packages"
)

type Logger interface {
	Log(string)
}

// Main modeled on [multichecker.Main](https://cs.opensource.google/go/x/tools/+/refs/tags/v0.29.0:go/analysis/multichecker/multichecker.go).
func Main(logger Logger, forceMiss bool, analyzers ...*analysis.Analyzer) {
	grouped := groupByLoadMode(analyzers)
	slow, _ := grouped[SlowLoadMode]
	fast, _ := grouped[FastLoadMode]
	logger.Log(fmt.Sprintf("slow: %v\tfast: %v", len(slow), len(fast)))

	// Fast run
	start := time.Now()
	{
		pkgs, err := packages.Load(&packages.Config{
			Mode:  FastLoadMode | packages.NeedModule,
			Tests: true,
		}, flag.Args()...)
		if err != nil {
			panic(err)
		}
		if _, err := checker.Analyze(fast, pkgs, nil); err != nil {
			panic(err)
		}
	}
	fastDone := time.Since(start)
	logger.Log(fmt.Sprintf("fast run: %v", fastDone))

	// Slow run
	{
		pkgs, err := packages.Load(&packages.Config{
			Mode:  SlowLoadMode | packages.NeedModule,
			Tests: true,
		}, flag.Args()...)
		if err != nil {
			panic(err)
		}
		if _, err := checker.Analyze(slow, pkgs, nil); err != nil {
			panic(err)
		}
	}
	logger.Log(fmt.Sprintf("slow run: %v", time.Since(start)-fastDone))

	// TODO: figure out how to report facts for use with vet.
}

func groupByLoadMode(analyzers []*analysis.Analyzer) map[packages.LoadMode][]*analysis.Analyzer {
	groups := make(map[packages.LoadMode][]*analysis.Analyzer)
	for _, a := range analyzers {
		mode := LoadMode(a)
		groups[mode] = append(groups[mode], a)
	}
	return groups
}

// func segmentByNeedFacts(analyzers []*analysis.Analyzer) (yes, no []*analysis.Analyzer) {
// 	seen := make(map[*analysis.Analyzer]bool)
// 	// needFacts reports whether any analysis required by the specified set
// 	// needs facts. If so, we must load the entire program from source.
// 	// https://cs.opensource.google/go/x/tools/+/refs/tags/v0.29.0:go/analysis/internal/checker/checker.go;l=451-469
// 	needFacts := func(as *analysis.Analyzer) bool {
// 		q := []*analysis.Analyzer{as} // for BFS
// 		for len(q) > 0 {
// 			a := q[0]
// 			q = q[1:]
// 			if !seen[a] {
// 				seen[a] = true
// 				if len(a.FactTypes) > 0 {
// 					return true
// 				}
// 				q = append(q, a.Requires...)
// 			}
// 		}
// 		return false
// 	}

// 	for _, a := range analyzers {
// 		if needFacts(a) {
// 			yes = append(yes, a)
// 		} else {
// 			no = append(no, a)
// 		}
// 	}
// 	return yes, no
// }
