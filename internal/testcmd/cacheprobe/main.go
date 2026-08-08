// cacheprobe is an integration-test vettool for Glint's cache protocol.
package main

import (
	"go/ast"
	"go/token"

	"github.com/lukasschwab/glint"
	"golang.org/x/tools/go/analysis"
)

type packageFact struct {
	Seen bool
}

func (*packageFact) AFact() {}

var analyzer = &analysis.Analyzer{
	Name:      "cacheprobe",
	Doc:       "reports each diagnostic root and exports a dependency fact",
	FactTypes: []analysis.Fact{new(packageFact)},
	Run: func(pass *analysis.Pass) (any, error) {
		pass.ExportPackageFact(&packageFact{Seen: true})
		if len(pass.Files) > 0 {
			diagnostic := analysis.Diagnostic{
				Pos:     pass.Files[0].Pos(),
				Message: "cache probe for " + pass.Pkg.Path(),
			}
			ast.Inspect(pass.Files[0], func(node ast.Node) bool {
				literal, ok := node.(*ast.BasicLit)
				if !ok || literal.Kind != token.INT || literal.Value != "1" {
					return true
				}
				diagnostic.SuggestedFixes = []analysis.SuggestedFix{{
					Message: "replace benchmark literal",
					TextEdits: []analysis.TextEdit{{
						Pos:     literal.Pos(),
						End:     literal.End(),
						NewText: []byte("2"),
					}},
				}}
				return false
			})
			pass.Report(diagnostic)
		}
		return nil, nil
	},
}

func main() {
	glint.Main(analyzer)
}
