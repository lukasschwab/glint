// nolintprobe is an integration-test vettool for diagnostic suppression.
package main

import (
	"go/ast"
	"go/token"

	"github.com/lukasschwab/glint"
	"golang.org/x/tools/go/analysis"
)

var analyzer = &analysis.Analyzer{
	Name: "nolintprobe",
	Doc:  "reports integer literals with a suggested fix",
	Run: func(pass *analysis.Pass) (any, error) {
		for _, file := range pass.Files {
			ast.Inspect(file, func(node ast.Node) bool {
				literal, ok := node.(*ast.BasicLit)
				if !ok || literal.Kind != token.INT || literal.Value != "1" {
					return true
				}
				pass.Report(analysis.Diagnostic{
					Pos:     literal.Pos(),
					Message: "replace literal",
					SuggestedFixes: []analysis.SuggestedFix{{
						Message:   "replace literal",
						TextEdits: []analysis.TextEdit{{Pos: literal.Pos(), End: literal.End(), NewText: []byte("2")}},
					}},
				})
				return true
			})
		}
		return nil, nil
	},
}

func main() { glint.Main(analyzer) }
