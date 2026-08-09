// Package forbidigo adapts forbidigo to go/analysis.
package forbidigo

import (
	"fmt"

	upstream "github.com/ashanbrown/forbidigo/v2/forbidigo"
	"golang.org/x/tools/go/analysis"
)

// New returns a forbidigo analyzer configured with patterns in forbidigo's
// YAML representation. The caller owns the forbidden patterns and any
// repository-specific diagnostic filtering.
func New(patterns []string) (*analysis.Analyzer, error) {
	linter, err := upstream.NewLinter(patterns,
		upstream.OptionExcludeGodocExamples(true),
		upstream.OptionIgnorePermitDirectives(true),
		upstream.OptionAnalyzeTypes(true),
	)
	if err != nil {
		return nil, fmt.Errorf("configure forbidigo: %w", err)
	}

	return &analysis.Analyzer{
		Name: "forbidigo",
		Doc:  "forbids configured identifiers",
		Run: func(pass *analysis.Pass) (any, error) {
			for _, file := range pass.Files {
				issues, err := linter.RunWithConfig(upstream.RunConfig{
					Fset:      pass.Fset,
					TypesInfo: pass.TypesInfo,
				}, file)
				if err != nil {
					return nil, fmt.Errorf("check %s: %w", file.Name.Name, err)
				}
				for _, issue := range issues {
					pass.Report(analysis.Diagnostic{Pos: issue.Pos(), Message: issue.Details()})
				}
			}
			return nil, nil
		},
	}, nil
}
