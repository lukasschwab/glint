// Package diagnostics provides composable filters for analyzer diagnostics.
package diagnostics

import (
	"go/ast"
	"go/token"

	"golang.org/x/tools/go/analysis"
)

// Predicate reports whether a diagnostic should be retained.
type Predicate func(*analysis.Pass, analysis.Diagnostic) bool

// Filter wraps analyzer so it only reports diagnostics accepted by keep.
// Analyzer results and facts are unchanged.
func Filter(analyzer *analysis.Analyzer, keep Predicate) {
	run := analyzer.Run
	analyzer.Run = func(pass *analysis.Pass) (any, error) {
		report := pass.Report
		pass.Report = func(diagnostic analysis.Diagnostic) {
			if keep(pass, diagnostic) {
				report(diagnostic)
			}
		}
		defer func() {
			pass.Report = report
		}()
		return run(pass)
	}
}

// ExcludeGenerated wraps analyzer so diagnostics whose primary position is in
// a generated Go file are suppressed. Generated files remain analyzer inputs.
func ExcludeGenerated(analyzer *analysis.Analyzer) {
	Filter(analyzer, func(pass *analysis.Pass, diagnostic analysis.Diagnostic) bool {
		return !generatedPosition(pass.Files, diagnostic.Pos)
	})
}

func generatedPosition(files []*ast.File, position token.Pos) bool {
	if !position.IsValid() {
		return false
	}
	for _, file := range files {
		if file.Pos() <= position && position <= file.End() {
			return ast.IsGenerated(file)
		}
	}
	return false
}
