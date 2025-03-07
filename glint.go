package glint

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/lukasschwab/glint/internal/tools/go/analysis/analysisflags"
	"github.com/lukasschwab/glint/pkg/checkrunner"
	"github.com/lukasschwab/glint/pkg/nolint"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/checker"
	"golang.org/x/tools/go/packages"
)

var (
	// Should control file set and destination.
	UseStdout = false
)

type Logger interface {
	Log(string)
}

// Main modeled on [multichecker.Main](https://cs.opensource.google/go/x/tools/+/refs/tags/v0.29.0:go/analysis/multichecker/multichecker.go).
func Main(analyzers ...*analysis.Analyzer) {
	progname := filepath.Base(os.Args[0])
	log.SetFlags(0)
	log.SetPrefix(progname + ": ")

	for _, a := range analyzers {
		nolint.Wrap(a)
	}

	if err := analysis.Validate(analyzers); err != nil {
		log.Fatal(err)
	}

	checkrunner.RegisterFlags()
	flag.BoolVar(&UseStdout, "stdout", false, "write linter findings to stdout instead of stderr (default false)")

	// NOTE: could use this to list and filter analyzers.
	analysisflags.Parse([]*analysis.Analyzer{}, true)

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

	runnable := buildRunner(args, analyzers...)

	sink := os.Stderr
	if UseStdout {
		sink = os.Stdout
	}

	os.Exit(checkrunner.Run(args, runnable, sink))
}

func buildRunner(args []string, analyzers ...*analysis.Analyzer) checkrunner.Runnable {
	grouped := groupByLoadMode(analyzers)

	return func(opts *checker.Options) (*checker.Graph, error) {
		var totalGraph *checker.Graph

		for loadMode, analyzers := range grouped {
			pkgs, err := packages.Load(&packages.Config{
				Mode:  loadMode,
				Tests: true,
			}, args...)
			if err != nil {
				panic(err)
			}
			if graph, err := checker.Analyze(analyzers, pkgs, opts); err != nil {
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
