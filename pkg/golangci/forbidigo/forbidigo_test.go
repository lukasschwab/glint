package forbidigo

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

func TestNew(t *testing.T) {
	analyzer, err := New([]string{`p: '^fmt\.Println$'
msg: use structured logging`})
	if err != nil {
		t.Fatal(err)
	}
	analysistest.Run(t, analysistest.TestData(), analyzer, "forbidigo")
}
