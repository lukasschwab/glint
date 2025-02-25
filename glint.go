package glint

import (
	"flag"
	"fmt"
	"log"

	"github.com/lukasschwab/glint/pkg/cache"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/checker"
	"golang.org/x/tools/go/packages"
)

type Logger interface {
	Log(string)
}

func Main(logger Logger, forceMiss bool, analyzers ...*analysis.Analyzer) {
	for i := range analyzers {
		// TODO: real logger.
		AddCache(analyzers[i], forceMiss, cache.NoopLogger{})
	}

	// TODO: can optimize, e.g. by checking for whether needs facts.
	// SEE https://cs.opensource.google/go/x/tools/+/refs/tags/v0.29.0:go/analysis/internal/checker/checker.go
	// FIXME: arg parsing here is dangerous.
	packages, err := packages.Load(&packages.Config{Mode: packages.LoadAllSyntax | packages.NeedModule, Tests: true}, flag.Args()[0:]...)
	if err != nil {
		panic(err)
	}

	graph, err := checker.Analyze(analyzers, packages, nil)
	if err != nil {
		panic(err)
	}

	for _, a := range graph.Roots {
		// This would be a fine place to write to the cache...
		// Or we can return the roots!
		log.Printf("Root: %v", a)
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
