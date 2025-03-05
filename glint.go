package glint

import (
	"flag"
	"fmt"
	"os"

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
		slow, _ := grouped[SlowLoadMode]
		fast, _ := grouped[FastLoadMode]
		logger.Log(fmt.Sprintf("slow: %v\tfast: %v", len(slow), len(fast)))
	}

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

	os.Exit(report(totalGraph))
}

func groupByLoadMode(analyzers []*analysis.Analyzer) map[packages.LoadMode][]*analysis.Analyzer {
	groups := make(map[packages.LoadMode][]*analysis.Analyzer)
	for _, a := range analyzers {
		mode := LoadMode(a)
		groups[mode] = append(groups[mode], a)
	}
	return groups
}
