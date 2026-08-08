// package golangci emulates behaviors from golangci-lint.
package golangci

import (
	"golang.org/x/tools/go/analysis"
	vet "golang.org/x/tools/go/analysis/suite/vet"

	"github.com/gordonklaus/ineffassign/pkg/ineffassign"
	"github.com/kisielk/errcheck/errcheck"
	honnef "honnef.co/go/tools/analysis/lint"
	"honnef.co/go/tools/quickfix"
	gosimple "honnef.co/go/tools/simple"
	"honnef.co/go/tools/staticcheck"
	"honnef.co/go/tools/stylecheck"
	"honnef.co/go/tools/unused"
)

// DefaultAnalyzers run by golangci-lint when no config is provided; see
// https://golangci-lint.run/usage/linters/#enabled-by-default
func DefaultAnalyzers() (result []*analysis.Analyzer) {
	// errcheck.Analyzer applies its version's default exclusions itself.
	result = append(result, errcheck.Analyzer)

	result = append(result, extractHonnefAnalyzers(gosimple.Analyzers)...)
	result = append(result, DefaultVetAnalyzers...)
	result = append(result, ineffassign.Analyzer)
	result = append(result, extractHonnefAnalyzers(staticcheck.Analyzers)...)
	result = append(result, defaultStyleAnalyzers()...)
	result = append(result, extractHonnefAnalyzers(quickfix.Analyzers)...)
	result = append(result, unused.Analyzer.Analyzer)

	return result
}

// DefaultVetAnalyzers matches golangci-lint's modern default vet suite.
// loopclosure remains available upstream for callers analyzing pre-Go 1.22
// modules, but golangci-lint excludes it for current Go versions.
var DefaultVetAnalyzers = analyzersExcept(vet.Suite, "loopclosure")

var golangciDisabledStyleAnalyzers = map[string]struct{}{
	"ST1000": {},
	"ST1003": {},
	"ST1016": {},
	"ST1020": {},
	"ST1021": {},
	"ST1022": {},
}

func defaultStyleAnalyzers() []*analysis.Analyzer {
	var result []*analysis.Analyzer
	for _, analyzer := range stylecheck.Analyzers {
		if _, disabled := golangciDisabledStyleAnalyzers[analyzer.Analyzer.Name]; !disabled {
			result = append(result, analyzer.Analyzer)
		}
	}
	return result
}

func analyzersExcept(analyzers []*analysis.Analyzer, excludedName string) []*analysis.Analyzer {
	var result []*analysis.Analyzer
	for _, analyzer := range analyzers {
		if analyzer.Name != excludedName {
			result = append(result, analyzer)
		}
	}
	return result
}

func extractHonnefAnalyzers(pre []*honnef.Analyzer) []*analysis.Analyzer {
	extracted := make([]*analysis.Analyzer, len(pre))
	for i := range extracted {
		extracted[i] = pre[i].Analyzer
	}
	return extracted
}
