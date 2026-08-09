package nolint

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"

	"golang.org/x/tools/go/analysis"
)

func TestParseDirective(t *testing.T) {
	for _, test := range []struct {
		comment  string
		analyzer string
		want     scope
		explicit bool
		matches  bool
	}{
		{"//nolint:probe", "probe", fileScope, false, true},
		{"//nolint:file: probe, other // reason", "probe", fileScope, true, true},
		{"//nolint:line: probe, other", "probe", lineScope, true, true},
		{"//nolint:decl: probe, other", "probe", declarationScope, true, true},
		{"//nolint:line: other", "probe", lineScope, true, false},
		{"//nolint:unknown:probe", "probe", fileScope, false, false},
		{"//nolint:decl:", "probe", declarationScope, true, false},
		{"//nolint:prober", "probe", fileScope, false, false},
	} {
		got, matches := parseDirective(test.comment, test.analyzer)
		if got.scope != test.want || got.explicit != test.explicit || matches != test.matches {
			t.Errorf("parseDirective(%q, %q) = (%v, %v, %v), want (%v, %v, %v)", test.comment, test.analyzer, got.scope, got.explicit, matches, test.want, test.explicit, test.matches)
		}
	}
}

func TestWrapRestoresReporterAndDoesNotSuppressInvalidPosition(t *testing.T) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", "package p\n//nolint:file:probe\nvar Value = 1\n", parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	var reports []analysis.Diagnostic
	pass := &analysis.Pass{
		Fset:  fset,
		Files: []*ast.File{file},
		Report: func(diagnostic analysis.Diagnostic) {
			reports = append(reports, diagnostic)
		},
	}
	analyzer := &analysis.Analyzer{
		Name: "probe",
		Run: func(pass *analysis.Pass) (any, error) {
			pass.Report(analysis.Diagnostic{Message: "no position"})
			pass.Report(analysis.Diagnostic{Pos: file.Pos(), Message: "file position"})
			return nil, nil
		},
	}
	Wrap(analyzer)
	for range 2 {
		if _, err := analyzer.Run(pass); err != nil {
			t.Fatal(err)
		}
	}
	pass.Report(analysis.Diagnostic{Message: "after run"})
	if len(reports) != 3 {
		t.Fatalf("reports = %#v, want two invalid-position reports and restored reporter", reports)
	}
}
