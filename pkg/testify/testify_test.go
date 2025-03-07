package testify_test

import (
	"testing"

	"github.com/lukasschwab/glint/pkg/testify"
	"golang.org/x/tools/go/analysis/analysistest"
)

func TestAnalyzer(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, testify.Analyzer, "./...")
}
