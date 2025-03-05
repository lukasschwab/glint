package glint

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/lukasschwab/glint/internal/tools/go/analysis/analysisflags"
	checkrunner "github.com/lukasschwab/glint/pkg/checker"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/checker"
	"golang.org/x/tools/go/analysis/unitchecker"
	"golang.org/x/tools/go/packages"
)

type Logger interface {
	Log(string)
}

// Main modeled on [multichecker.Main](https://cs.opensource.google/go/x/tools/+/refs/tags/v0.29.0:go/analysis/multichecker/multichecker.go).
func Main(logger Logger, analyzers ...*analysis.Analyzer) {
	progname := filepath.Base(os.Args[0])
	log.SetFlags(0)
	log.SetPrefix(progname + ": ") // e.g. "vet: "

	if err := analysis.Validate(analyzers); err != nil {
		log.Fatal(err)
	}

	checkrunner.RegisterFlags()

	analyzers = analysisflags.Parse(analyzers, true)

	args := flag.Args()
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, `%[1]s is a tool for static analysis of Go programs.

Usage: %[1]s [-flag] [package]

Run '%[1]s help' for more detail,
 or '%[1]s help name' for details and flags of a specific analyzer.
`, progname)
		os.Exit(1)
	}

	if args[0] == "help" {
		analysisflags.Help(progname, analyzers, args[1:])
		os.Exit(0)
	}

	if len(args) == 1 && strings.HasSuffix(args[0], ".cfg") {
		unitchecker.Run(args[0], analyzers)
		panic("unreachable")
	}

	runnable := buildRunner(logger, analyzers...)

	os.Exit(checkrunner.Run([]string{}, runnable))
}

func buildRunner(logger Logger, analyzers ...*analysis.Analyzer) checkrunner.Runnable {
	grouped := groupByLoadMode(analyzers)
	{
		slow := grouped[SlowLoadMode]
		fast := grouped[FastLoadMode]
		logger.Log(fmt.Sprintf("slow: %v\tfast: %v", len(slow), len(fast)))
	}

	return func(opts *checker.Options) (*checker.Graph, error) {
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
