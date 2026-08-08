# GitHub Actions

Glint is most useful when a repository owns the small Go program that defines
its analyzer set. The action in this repository builds that program, then runs
it from the module being checked. It leaves Go installation and cache policy to
the workflow.

## Define the linter

An isolated module keeps analyzer dependencies out of the application's
`go.mod`:

```text
tools/glint/
├── go.mod
├── go.sum
└── main.go
```

For example, `tools/glint/main.go` can start from Glint's approximation of
golangci-lint's defaults and add repository-specific analyzers:

```go
package main

import (
	"github.com/lukasschwab/glint"
	"github.com/lukasschwab/glint/pkg/diagnostics"
	"github.com/lukasschwab/glint/pkg/golangci"
)

func main() {
	analyzers := golangci.DefaultAnalyzers()
	// analyzers = append(analyzers, projectlint.Analyzer)
	for _, analyzer := range analyzers {
		diagnostics.ExcludeGenerated(analyzer)
	}
	glint.Main(analyzers...)
}
```

Run `go mod tidy` in `tools/glint` after adding analyzers. Prefer their public
`analysis.Analyzer` APIs and configure them in Go; analyzers that only expose a
golangci-lint integration may need an adapter.

The generated-file filter above suppresses diagnostics without removing those
files from analyzer inputs, so facts and cross-file analysis remain intact.
`diagnostics.Filter` can express other repository-specific diagnostic policy
as a typed Go predicate. Keep such filters narrow and tested.

## Contain compatibility adapters

Prefer importing a linter's public `analysis.Analyzer` directly. When a
library exposes a checker with a different API, first classify the adapter.

Faithful, reusable GolangCI-Lint compatibility belongs in Glint when it only
recreates the upstream GolangCI integration and accepts all policy as explicit
arguments. These adapters live in linter-specific child packages so callers
who only use `golangci.DefaultAnalyzers` do not compile or link their optional
dependencies:

```go
import (
	"github.com/lukasschwab/glint/pkg/golangci/contextcheck"
	"github.com/lukasschwab/glint/pkg/golangci/forbidigo"
	"github.com/lukasschwab/glint/pkg/golangci/gochecknoinits"
	"github.com/lukasschwab/glint/pkg/golangci/unparam"
)

context := contextcheck.New("example.com/project")
forbidden, err := forbidigo.New(patterns)
noInits := gochecknoinits.New()
unusedParams := unparam.New(false)
```

These are not GolangCI defaults, so callers append them deliberately. The
forbidigo patterns, the module path passed to contextcheck, and unparam's
exported-function setting remain consumer choices.

Keep repository-specific policy and configuration in the repository-owned
linter module instead:

```text
tools/glint/
├── internal/
│   └── policy/
│       └── project_rules.go
├── go.mod
└── main.go
```

Do not move repository-specific forbidden patterns, path exceptions, analyzer
settings, diagnostic filters, or wrappers around libraries that already expose
a suitable public `analysis.Analyzer`. Keep those choices visible in the
composition and give substantial consumer integrations focused behavioral
tests.

Remember that `go vet` invokes the analysis tool in a package-oriented process
model. A wrapper that initializes a large global registry, reloads packages,
or rereads every source file can repeat expensive setup for each package. Such
tools may be better kept as explicit standalone workflow steps, just like
formatters and module-level policy checks. This is an integration-performance
decision, not a reason to make Glint own a fixed catalog of linters.

## Run it in CI

Set up Go before invoking the action. Include both the application and linter
module sums in `cache-dependency-path` so changes to either dependency graph
produce an appropriate cache entry.

```yaml
steps:
  - uses: actions/checkout@<commit-sha>
  - uses: actions/setup-go@<commit-sha>
    with:
      go-version-file: go.mod
      cache-dependency-path: |
        go.sum
        tools/glint/go.sum
  - uses: lukasschwab/glint/action@<commit-sha>
    with:
      linter-directory: tools/glint
      arguments: |
        -test
        -tags=devauth
        ./...
```

Each non-empty line in `arguments` is passed as one exact argument. The default
is `./...`. `working-directory` defaults to the repository root; set it when
the application module lives elsewhere. `linter-package` defaults to `.` and
can select a main package inside a larger tools module instead.

Pin third-party actions, including this one, to a full commit SHA in production
workflows.

## Cache behavior

The action does not create a separate Glint cache. Building the linter and
running its analyzers use Go's module and build caches, so `actions/setup-go`
can restore both. The first run after an analyzer, setting, build tag, source,
or dependency change does the work; unchanged reruns reuse the cached analyzer
results.

The linter build disables Go's automatic VCS stamping. Otherwise, an unrelated
commit in the application repository would change the linter binary's vet-tool
identity and invalidate every analyzer result. Linter source and dependency
changes still produce a new build ID and invalidate the appropriate cache
entries.

The action does not override `GOCACHE` or `GOCACHEPROG`. Glint inherits both,
which lets repositories use the same local or remote build-cache policy as
their other Go commands. Its trim-path invocation also makes cache entries safe
to share between separate worktrees.

Formatting and non-`analysis.Analyzer` checks remain separate workflow steps.
For example, keep `gofmt`, `gci`, or generated-file checks alongside Glint
rather than hiding them in the linter driver.
