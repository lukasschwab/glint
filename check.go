package glint

import (
	"flag"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/lukasschwab/glint/internal/tools/go/analysis/analysisflags"
	"golang.org/x/tools/go/analysis/checker"
)

var (
	// Debug is a set of single-letter flags:
	//
	//	f	show [f]acts as they are created
	// 	p	disable [p]arallel execution of analyzers
	//	s	do additional [s]anity checks on fact types and serialization
	//	t	show [t]iming info (NB: use 'p' flag to avoid GC/scheduler noise)
	//	v	show [v]erbose logging
	//
	Debug = ""

	// Log files for optional performance tracing.
	CPUProfile, MemProfile, Trace string

	// IncludeTests indicates whether test files should be analyzed too.
	IncludeTests = true

	// Fix determines whether to apply (!Diff) or display (Diff) all suggested fixes.
	Fix bool

	// Diff causes the file updates to be displayed, but not applied.
	// This flag has no effect unless Fix is true.
	Diff bool
)

// RegisterFlags registers command-line flags used by the analysis driver.
func RegisterFlags() {
	// When adding flags here, remember to update
	// the list of suppressed flags in analysisflags.

	flag.StringVar(&Debug, "debug", Debug, `debug flags, any subset of "fpstv"`)

	flag.StringVar(&CPUProfile, "cpuprofile", "", "write CPU profile to this file")
	flag.StringVar(&MemProfile, "memprofile", "", "write memory profile to this file")
	flag.StringVar(&Trace, "trace", "", "write trace log to this file")
	flag.BoolVar(&IncludeTests, "test", IncludeTests, "indicates whether test files should be analyzed, too")

	flag.BoolVar(&Fix, "fix", false, "apply all suggested fixes")
	flag.BoolVar(&Diff, "diff", false, "with -fix, don't update the files, but print a unified diff")
}

// Modeled on internal/tools/go/analysis/checker
// TODO: handle command line flags
func report(graph *checker.Graph) (exitcode int) {
	return printDiagnostics(graph)
}

func mergeGraphs(graphs ...*checker.Graph) *checker.Graph {
	joined := &checker.Graph{Roots: []*checker.Action{}}
	for _, graph := range graphs {
		if graph != nil {
			joined.Roots = append(joined.Roots, graph.Roots...)
		}
	}
	return joined
}

// printDiagnostics prints diagnostics in text or JSON form
// and returns the appropriate exit code.
func printDiagnostics(graph *checker.Graph) (exitcode int) {
	// Print the results.
	// With -json, the exit code is always zero.
	if analysisflags.JSON {
		if err := graph.PrintJSON(os.Stdout); err != nil {
			return 1
		}
	} else {
		if err := graph.PrintText(os.Stderr, analysisflags.Context); err != nil {
			return 1
		}

		// Compute the exit code.
		var numErrors, rootDiags int
		// TODO(adonovan): use "for act := range graph.All() { ... }" in go1.23.
		graph.All()(func(act *checker.Action) bool {
			if act.Err != nil {
				numErrors++
			} else if act.IsRoot {
				rootDiags += len(act.Diagnostics)
			}
			return true
		})
		if numErrors > 0 {
			exitcode = 1 // analysis failed, at least partially
		} else if rootDiags > 0 {
			exitcode = 3 // successfully produced diagnostics
		}
	}

	// Print timing info.
	if dbg('t') {
		if !dbg('p') {
			log.Println("Warning: times are mostly GC/scheduler noise; use -debug=tp to disable parallelism")
		}

		var list []*checker.Action
		var total time.Duration
		// TODO(adonovan): use "for act := range graph.All() { ... }" in go1.23.
		graph.All()(func(act *checker.Action) bool {
			list = append(list, act)
			total += act.Duration
			return true
		})

		// Print actions accounting for 90% of the total.
		sort.Slice(list, func(i, j int) bool {
			return list[i].Duration > list[j].Duration
		})
		var sum time.Duration
		for _, act := range list {
			fmt.Fprintf(os.Stderr, "%s\t%s\n", act.Duration, act)
			sum += act.Duration
			if sum >= total*9/10 {
				break
			}
		}
		if total > sum {
			fmt.Fprintf(os.Stderr, "%s\tall others\n", total-sum)
		}
	}

	return exitcode
}

func dbg(b byte) bool { return strings.IndexByte(Debug, b) >= 0 }
