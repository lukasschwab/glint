package glint

import (
	"flag"
	"fmt"
	"os"

	checkrunner "github.com/lukasschwab/glint/pkg/checker"
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
	{
		slow := grouped[SlowLoadMode]
		fast := grouped[FastLoadMode]
		logger.Log(fmt.Sprintf("slow: %v\tfast: %v", len(slow), len(fast)))
	}

	runner := func(opts *checker.Options) (*checker.Graph, error) {
		var totalGraph *checker.Graph
		for loadMode, analyzers := range grouped {
			logger.Log(fmt.Sprintf("Loading packages for %v run", modeName[loadMode]))
			pkgs, err := packages.Load(&packages.Config{
				Mode:  loadMode,
				Tests: true,
			}, flag.Args()...)
			if err != nil {
				panic(err)
			}
			logger.Log(fmt.Sprintf("Starting %v run", modeName[loadMode]))
			if graph, err := checker.Analyze(analyzers, pkgs, nil); err != nil {
				panic(err)
			} else {
				totalGraph = mergeGraphs(totalGraph, graph)
			}
		}

		return totalGraph, nil
	}

	// TODO: figure out what else multichecker runs before calling checker.Run
	os.Exit(checkrunner.Run([]string{}, runner))
}

func mergeGraphs(graphs ...*checker.Graph) *checker.Graph {
	final := &checker.Graph{}
	for _, graph := range graphs {
		if graph != nil {
			final.Roots = append(final.Roots, graph.Roots...)
		}
	}
	return final
}

func groupByLoadMode(analyzers []*analysis.Analyzer) map[packages.LoadMode][]*analysis.Analyzer {
	groups := make(map[packages.LoadMode][]*analysis.Analyzer)
	for _, a := range analyzers {
		mode := LoadMode(a)
		groups[mode] = append(groups[mode], a)
	}
	return groups
}
