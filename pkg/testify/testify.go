package testify

import (
	"go/ast"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
)

var Analyzer = &analysis.Analyzer{
	Name: "testify",
	Doc:  "find and replace github.com/stretchr/testify with github.com/peterldowns/testy",
	Run:  run,
	Requires: []*analysis.Analyzer{
		inspect.Analyzer,
	},
}

func run(pass *analysis.Pass) (interface{}, error) {
	for _, file := range pass.Files {
		ast.Inspect(file, func(n ast.Node) bool {
			importSpec, ok := n.(*ast.ImportSpec)
			if !ok {
				return true
			}

			if strings.HasPrefix(importSpec.Path.Value, `"github.com/stretchr/testify`) {
				newText := strings.ReplaceAll(importSpec.Path.Value, `"github.com/stretchr/testify`, `"github.com/peterldowns/testy`)

				pass.Report(analysis.Diagnostic{
					Pos:     importSpec.Pos(),
					End:     importSpec.End(),
					Message: "replace with peterldowns/testy",
					SuggestedFixes: []analysis.SuggestedFix{
						{
							Message: "Replace with github.com/peterldowns/testy",
							TextEdits: []analysis.TextEdit{
								{
									Pos:     importSpec.Path.Pos(),
									End:     importSpec.Path.End(),
									NewText: []byte(newText),
								},
							},
						},
					},
				})
			}
			return true
		})
	}
	return nil, nil
}
