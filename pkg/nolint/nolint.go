package nolint

import (
	"fmt"
	"go/ast"
	"slices"

	"golang.org/x/tools/go/analysis"
)

// Wrap a.Run in a function that filters out files with a nolint directive
// aimed at a.
func Wrap(a *analysis.Analyzer) {
	run := a.Run
	a.Run = func(pass *analysis.Pass) (interface{}, error) {
		retainedFiles := make([]*ast.File, 0, len(pass.Files))
		for _, file := range pass.Files {
			shouldIgnore := slices.ContainsFunc(file.Comments, func(comment *ast.CommentGroup) bool {
				return comment.List[0].Text == fmt.Sprintf("//nolint:%s", a.Name)
			})
			if shouldIgnore {
				pass.IgnoredFiles = append(pass.IgnoredFiles, file.Name.Name)
			} else {
				retainedFiles = append(retainedFiles, file)
			}
		}

		pass.Files = retainedFiles
		return run(pass)
	}
}
