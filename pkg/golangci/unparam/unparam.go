// Package unparam adapts unparam to go/analysis.
package unparam

import (
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/buildssa"
	"golang.org/x/tools/go/packages"
	"mvdan.cc/unparam/check"
)

// New returns an analyzer that reports unused function parameters. The
// checkExported argument matches unparam's CheckExportedFuncs setting.
func New(checkExported bool) *analysis.Analyzer {
	return &analysis.Analyzer{
		Name:     "unparam",
		Doc:      "reports unused function parameters",
		Requires: []*analysis.Analyzer{buildssa.Analyzer},
		Run: func(pass *analysis.Pass) (any, error) {
			ssa := pass.ResultOf[buildssa.Analyzer].(*buildssa.SSA)
			pkg := &packages.Package{
				Fset:      pass.Fset,
				Syntax:    pass.Files,
				Types:     pass.Pkg,
				TypesInfo: pass.TypesInfo,
			}

			checker := &check.Checker{}
			checker.CheckExportedFuncs(checkExported)
			checker.Packages([]*packages.Package{pkg})
			checker.ProgramSSA(ssa.Pkg.Prog)
			issues, err := checker.Check()
			if err != nil {
				return nil, err
			}
			for _, issue := range issues {
				pass.Report(analysis.Diagnostic{Pos: issue.Pos(), Message: issue.Message()})
			}
			return nil, nil
		},
	}
}
