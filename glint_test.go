package glint_test

import (
	"testing"

	"github.com/peterldowns/testy/assert"
	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/lukasschwab/glint"
	nilinterface "github.com/lukasschwab/nilinterface/pkg/analyzer"
)

func TestAddCache(t *testing.T) {
	testdata := analysistest.TestData()

	uncached := nilinterface.Analyzer
	expected := analysistest.Run(t, testdata, uncached, "./...")

	glint.AddCache(uncached, true)
	actual := analysistest.Run(t, testdata, uncached, "./...")

	// Just confirms no infinite recursion due to stored function manipulation.
	assert.Equal(t, len(expected), len(actual))
	assert.Equal(t, 1, len(actual))
}
