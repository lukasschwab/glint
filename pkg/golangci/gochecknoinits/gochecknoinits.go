// Package gochecknoinits recreates GolangCI-Lint's gochecknoinits behavior.
package gochecknoinits

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// New returns an analyzer that reports package init functions.
func New() *analysis.Analyzer {
	return &analysis.Analyzer{
		Name:     "gochecknoinits",
		Doc:      "checks that no init functions are present",
		Requires: []*analysis.Analyzer{inspect.Analyzer},
		Run: func(pass *analysis.Pass) (any, error) {
			inspect := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
			for node := range inspect.PreorderSeq((*ast.FuncDecl)(nil)) {
				function := node.(*ast.FuncDecl)
				if function.Recv == nil && function.Name.Name == "init" {
					pass.Reportf(function.Pos(), "don't use `init` function")
				}
			}
			return nil, nil
		},
	}
}
