package cache

import (
	"fmt"
	"go/types"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/packages"

	"github.com/golangci/golangci-lint/pkg/logutils"
	"github.com/golangci/golangci-lint/pkg/timeutils"
	golangci "github.com/lukasschwab/glint/pkg/golangci/cache"
)

const (
	// TODO: look into optimizing.
	hashMode = golangci.HashModeNeedAllDeps
)

var (
	packagesCfg = &packages.Config{
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedCompiledGoFiles | packages.NeedImports | packages.NeedDeps,
	}
)

type cache struct {
	inner  *golangci.Cache
	key    string
	logger Logger
}

func New(analyzer *analysis.Analyzer, logger Logger) cache {
	result := cache{
		key:    "lint/result:" + analyzer.Name,
		logger: logger,
	}
	{
		var err error
		log := logutils.NewStderrLog("skip")
		sw := timeutils.NewStopwatch("pkgcache", log.Child(logutils.DebugKeyStopwatch))
		if result.inner, err = golangci.NewCache(sw, log); err != nil {
			panic(err)
		}
	}
	return result
}

type Entry struct {
	Check       bool
	Diagnostics []analysis.Diagnostic
	Error       error
}

func (c cache) For(pass *analysis.Pass) (entry Entry, ok bool) {
	pkg := c.loadPackage(pass.Pkg)
	if err := c.inner.Get(pkg, hashMode, c.key, &entry); err != nil {
		c.logger.Logf("cache miss: %v", err)
		return entry, false
	} else if !entry.Check {
		c.logger.Logf("junk data: %v", entry)
		return entry, false
	}
	println(fmt.Sprintf("cache hit for %v", pass))
	c.logger.Logf("cache hit: %+v", entry)
	return entry, true
}

func (c cache) Write(pass *analysis.Pass, diagnostics []analysis.Diagnostic, err error) {
	e := Entry{
		Check: true,
		// Result:       result,
		Error: err,
	}
	pkg := c.loadPackage(pass.Pkg)
	if putErr := c.inner.Put(pkg, hashMode, c.key, e); putErr != nil {
		panic(putErr)
	}
	c.logger.Logf("wrote result to cache: %+v", e)
}

// FIXME: unsafe error handling.
// TODO: cache these results in memory?
// FIXME: I think golangci-lint just does this by constructing a struct? gosec.go
func (c cache) loadPackage(pkg *types.Package) *packages.Package {
	packages, err := packages.Load(packagesCfg, pkg.Path())
	if err != nil {
		c.logger.Logf("error loading package '%v': %v", pkg.Path(), err)
		return nil
	} else if len(packages) == 0 {
		c.logger.Logf("zero packages found at '%v': %v", pkg.Path(), err)
		return nil
	}
	c.logger.Logf("%d packages found at '%v'", len(packages), pkg.Path())
	return packages[0]
}
