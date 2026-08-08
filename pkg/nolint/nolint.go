package nolint

import (
	"go/ast"
	"slices"
	"strings"

	"golang.org/x/tools/go/analysis"
)

// Wrap a.Run in a function that filters diagnostics from files with a nolint
// directive aimed at a.
func Wrap(a *analysis.Analyzer) {
	run := a.Run
	a.Run = func(pass *analysis.Pass) (interface{}, error) {
		var ignoredFiles []*ast.File
		for _, file := range pass.Files {
			shouldIgnore := slices.ContainsFunc(file.Comments, func(comment *ast.CommentGroup) bool {
				return slices.ContainsFunc(comment.List, func(comment *ast.Comment) bool {
					return suppressesAnalyzer(comment.Text, a.Name)
				})
			})
			if shouldIgnore {
				ignoredFiles = append(ignoredFiles, file)
			}
		}

		report := pass.Report
		pass.Report = func(diagnostic analysis.Diagnostic) {
			ignored := slices.ContainsFunc(ignoredFiles, func(file *ast.File) bool {
				return file.Pos() <= diagnostic.Pos && diagnostic.Pos <= file.End()
			})
			if !ignored {
				report(diagnostic)
			}
		}
		defer func() {
			pass.Report = report
		}()
		return run(pass)
	}
}

func suppressesAnalyzer(comment, analyzer string) bool {
	directive, ok := strings.CutPrefix(comment, "//nolint:")
	if !ok {
		return false
	}
	directive = strings.TrimSpace(directive)
	if end := strings.Index(directive, "//"); end >= 0 {
		directive = directive[:end]
	}
	for name := range strings.SplitSeq(directive, ",") {
		name = strings.TrimSpace(name)
		if end := strings.IndexAny(name, " \t"); end >= 0 {
			name = name[:end]
		}
		if name == analyzer {
			return true
		}
	}
	return false
}
