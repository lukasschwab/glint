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
		matches  bool
	}{
		{"//nolint:probe", "probe", true},
		{"//nolint: probe, other // reason", "probe", true},
		{"//nolint:other", "probe", false},
		{"//nolint:file:probe", "probe", false},
		{"//nolint:prober", "probe", false},
	} {
		if got := suppressesAnalyzer(test.comment, test.analyzer); got != test.matches {
			t.Errorf("suppressesAnalyzer(%q, %q) = %v, want %v", test.comment, test.analyzer, got, test.matches)
		}
	}
}

func TestWrapRestoresReporterAndDoesNotSuppressInvalidPosition(t *testing.T) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", "//nolint:probe\npackage p\nvar Value = 1\n", parser.ParseComments)
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

func TestWrapFiltersSuggestedFixesWithTheirDiagnostics(t *testing.T) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", "package p\nvar suppressed = 1 //nolint:probe\nvar unsuppressed = 2\n", parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	var literals []*ast.BasicLit
	ast.Inspect(file, func(node ast.Node) bool {
		if literal, ok := node.(*ast.BasicLit); ok {
			literals = append(literals, literal)
		}
		return true
	})
	if len(literals) != 2 {
		t.Fatalf("literals = %d, want 2", len(literals))
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
			for _, literal := range literals {
				pass.Report(analysis.Diagnostic{
					Pos:     literal.Pos(),
					Message: "replace " + literal.Value,
					SuggestedFixes: []analysis.SuggestedFix{{
						Message:   "replace " + literal.Value,
						TextEdits: []analysis.TextEdit{{Pos: literal.Pos(), End: literal.End(), NewText: []byte("replace-" + literal.Value)}},
					}},
				})
			}
			return nil, nil
		},
	}
	Wrap(analyzer)
	if _, err := analyzer.Run(pass); err != nil {
		t.Fatal(err)
	}
	if len(reports) != 1 || reports[0].Message != "replace 2" {
		t.Fatalf("reports = %#v, want only the unsuppressed diagnostic", reports)
	}
	if len(reports[0].SuggestedFixes) != 1 || string(reports[0].SuggestedFixes[0].TextEdits[0].NewText) != "replace-2" {
		t.Fatalf("unsuppressed suggested fix was not preserved: %#v", reports[0].SuggestedFixes)
	}
}
