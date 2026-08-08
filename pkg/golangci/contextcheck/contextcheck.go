// Package contextcheck adapts GolangCI-Lint's contextcheck integration.
package contextcheck

import (
	upstream "github.com/kkHAIKE/contextcheck"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/ctrlflow"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/packages"
)

// New returns contextcheck with the prerequisites and package filtering used
// by GolangCI-Lint v2.10.1. packagePath should be the module path being
// analyzed.
func New(packagePath string) *analysis.Analyzer {
	analyzer := upstream.NewAnalyzer(upstream.Configuration{})
	analyzer.Requires = append(analyzer.Requires, ctrlflow.Analyzer, inspect.Analyzer)
	analyzer.Run = upstream.NewRun([]*packages.Package{{PkgPath: packagePath}}, false)
	return analyzer
}
