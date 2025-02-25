package glint

import (
	"flag"
	"fmt"

	"github.com/lukasschwab/glint/pkg/cache"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/checker"
	"golang.org/x/tools/go/packages"
)

type Logger interface {
	Log(string)
}

const (
	SlowLoadMode = packages.LoadAllSyntax
	FastLoadMode = packages.LoadSyntax
)

func Main(logger Logger, forceMiss bool, analyzers ...*analysis.Analyzer) {
	for i := range analyzers {
		// TODO: real logger.
		AddCache(analyzers[i], forceMiss, cache.NoopLogger{})
	}

	// NOTE: maybe I segment the analyzers by whether they need facts, then use
	// different load modes for each run?

	// // LoadSyntax loads typed syntax for the initial packages.
	// LoadSyntax = LoadTypes | NeedSyntax | NeedTypesInfo
	//
	// // LoadAllSyntax loads typed syntax for the initial packages and all dependencies.
	// LoadAllSyntax = LoadSyntax | NeedDeps

	slow, fast := segmentByNeedFacts(analyzers)
	println("slow:", len(slow), "	fast:", len(fast))

	// Fast run
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

	pkgs, err = packages.Load(&packages.Config{
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

func AddCache(
	analyzer *analysis.Analyzer,
	forceMiss bool,
	logger cache.Logger,
) {
	runAnalyzer := analyzer.Run

	acache := cache.New(analyzer, logger)
	analyzer.Run = func(pass *analysis.Pass) (any, error) {
		println(fmt.Sprintf("%v analyzing package '%v'", analyzer.Name, pass.Pkg.Path()))
		logger.Logf("%v analyzing package '%v'", analyzer.Name, pass.Pkg.Path())

		if entry, ok := acache.For(pass); ok && !forceMiss {
			for _, diagnostic := range entry.Diagnostics {
				pass.Report(diagnostic)
			}
			// TODO: figure out how to serialize, deserialize the results:
			// analyzer.ResultType.
			return nil, entry.Error
		}

		diagnostics := []analysis.Diagnostic{}
		report := pass.Report
		pass.Report = func(d analysis.Diagnostic) {
			diagnostics = append(diagnostics, d)
			report(d)
		}

		result, err := runAnalyzer(pass)

		acache.Write(pass, diagnostics, err)

		return result, err
	}
}

func segmentByNeedFacts(analyzers []*analysis.Analyzer) (yes, no []*analysis.Analyzer) {
	seen := make(map[*analysis.Analyzer]bool)
	// needFacts reports whether any analysis required by the specified set
	// needs facts.  If so, we must load the entire program from source.
	// https://cs.opensource.google/go/x/tools/+/refs/tags/v0.29.0:go/analysis/internal/checker/checker.go;l=451-469
	needFacts := func(as *analysis.Analyzer) bool {
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

	for _, a := range analyzers {
		if needFacts(a) {
			yes = append(yes, a)
		} else {
			no = append(no, a)
		}
	}
	return yes, no
}
