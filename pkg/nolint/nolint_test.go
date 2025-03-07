package nolint_test

import (
	"testing"

	"github.com/lukasschwab/glint/pkg/nolint"
	nilinterface "github.com/lukasschwab/nilinterface/pkg/analyzer"
	"golang.org/x/tools/go/analysis/analysistest"
)

func TestAnalyzer(t *testing.T) {
	testdata := analysistest.TestData()
	nolint.Wrap(nilinterface.Analyzer)
	analysistest.Run(t, testdata, nilinterface.Analyzer, "./...")
}
