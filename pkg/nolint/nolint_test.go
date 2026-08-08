package nolint_test

import (
	"go/ast"
	"testing"

	"github.com/lukasschwab/glint/pkg/nolint"
	nilinterface "github.com/lukasschwab/nilinterface/pkg/analyzer"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/analysistest"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

func TestAnalyzer(t *testing.T) {
	testdata := analysistest.TestData()
	nolint.Wrap(nilinterface.Analyzer)
	analysistest.Run(t, testdata, nilinterface.Analyzer, "./...")
}

func TestAnalyzerUsingInspector(t *testing.T) {
	testdata := analysistest.TestData()
	analyzer := &analysis.Analyzer{
		Name:     "inspectprobe",
		Doc:      "reports main calls through a required inspector",
		Requires: []*analysis.Analyzer{inspect.Analyzer},
		Run: func(pass *analysis.Pass) (any, error) {
			inspect := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
			inspect.Preorder([]ast.Node{(*ast.CallExpr)(nil)}, func(node ast.Node) {
				call := node.(*ast.CallExpr)
				name, ok := call.Fun.(*ast.Ident)
				if ok && name.Name == "main" {
					pass.Reportf(call.Pos(), "nil passed to interface parameter")
				}
			})
			return nil, nil
		},
	}
	nolint.Wrap(analyzer)
	analysistest.Run(t, testdata, analyzer, "./...")
}
