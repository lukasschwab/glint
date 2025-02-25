package glint_test

import (
	"testing"

	"github.com/peterldowns/testy/assert"
	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/lukasschwab/glint"
	"github.com/lukasschwab/glint/pkg/cache"
	nilinterface "github.com/lukasschwab/nilinterface/pkg/analyzer"
)

var (
	testdata = analysistest.TestData()
)

func TestAddCache(t *testing.T) {
	tempDir := t.TempDir()
	t.Logf("Cache directory: %s", tempDir)
	t.Setenv("GLINT_CACHE", tempDir)

	uncached := nilinterface.Analyzer
	expected := analysistest.Run(t, testdata, uncached, "./...")

	glint.AddCache(uncached, true, cache.TestLogger{t})
	actual := analysistest.Run(t, testdata, uncached, "./...")

	// Just confirms no infinite recursion due to stored function manipulation.
	assert.Equal(t, len(expected), len(actual))
	assert.Equal(t, 1, len(actual))
}

func TestRun(t *testing.T) {
	tempDir := t.TempDir()
	t.Logf("Cache directory: %s", tempDir)
	t.Setenv("GLINT_CACHE", tempDir)

	analyzer := nilinterface.Analyzer
	glint.AddCache(analyzer, true, cache.TestLogger{t})

	t.Log("First run")
	first := analysistest.Run(t, testdata, analyzer, "./...")
	assert.Equal(t, 1, len(first))

	t.Log("Second run")
	second := analysistest.Run(t, testdata, analyzer, "./...")
	assert.Equal(t, 1, len(second))
	assert.Equal(t, first[0].Diagnostics, second[0].Diagnostics)
}
