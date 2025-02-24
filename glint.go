package glint

import (
	"github.com/lukasschwab/glint/pkg/cache"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/multichecker"
)

func Main(analyzers ...*analysis.Analyzer) {
	// TODO: allow disabling the cache this really just requires forcing a miss.
	for i := range analyzers {
		AddCache(analyzers[i], false)
	}

	multichecker.Main(analyzers...)
}

func AddCache(analyzer *analysis.Analyzer, forceMiss bool) {
	initialRun := analyzer.Run
	cachedRun := func(pass *analysis.Pass) (any, error) {
		if result, err, ok := cache.For(pass); ok && !forceMiss {
			return result, err
		}
		// TODO: check not infinite recursion.
		result, err := initialRun(pass)
		cache.Write(result, err)
		return result, err
	}

	analyzer.Run = cachedRun
}
