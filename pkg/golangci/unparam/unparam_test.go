package unparam

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

func TestNew(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), New(false), "unparam")
}
