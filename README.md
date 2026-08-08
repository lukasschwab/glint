# glint

Experimental Go-defined metalinter.

Glint requires Go 1.26 or newer. Its analysis backend is a `go vet`
compatible unit checker, so compilation, analyzer facts, fixes, and diagnostic
output use Go's content-addressed build cache.

## Prospectus

- [x] VS Code integration
- [x] GitHub Action example
    - [ ] Action using a version in a separate repo
- [x] `nolint` directives
- [x] Clear demo of `-fix` working
- [x] Safe cached diagnostics across concurrent worktrees
- [ ] Deep-dive the lint scope: compare result sets on go-fiber.

## Caching and concurrency

Glint does not maintain a separate result database. It asks `go vet` to run
each package as an independent analysis unit and emit JSON into the output
file introduced by Go 1.26. Go caches that output alongside the package's
`.vetx` facts and compilation artifacts.

Two details make that safe:

- An invocation-scope digest, including the module-relative working
  directory, is included in every vet action key. A package analyzed only to
  produce dependency facts therefore cannot suppress its diagnostics when a
  later command selects it as a root.
- Cached filenames are stored as slash-normalized module-relative markers.
  The outer process rebases them onto the current module directory. Together
  with `-trimpath`, identical worktrees can share entries without leaking
  absolute paths from one checkout to another.

Glint relies on the standard Go cache's atomic publication. Multiple Glint
processes doing read-only analysis may safely share `GOCACHE`; duplicate cold
work is harmless and no global Glint lock is used. Applying `-fix` runs are
scoped to the current checkout because Go's fix archives contain destination
paths. (`-fix -diff` remains safely shareable because it does not mutate
files.)

On Prometheus commit `e75af38` on 2026-08-07, Glint and golangci-lint v2.11.4
were each run with the same 191 named analyzers: `errcheck`, `ineffassign`,
`unused`, the 34 modern default vet analyzers, and the 154 checks selected by
golangci-lint's default `staticcheck` configuration. Dependencies were already
downloaded, but each cold sample used otherwise empty caches.

| Runner | Cold | Warm | One-file edit |
| --- | ---: | ---: | ---: |
| Glint defaults | 98.31s (94.68-100.18) | 1.45s (1.38-1.52) | 6.93s (6.66-6.98) |
| golangci-lint v2.11.4 | 110.84s (108.15-119.02) | 2.00s (1.91-2.34) | 12.27s (12.27-12.96) |

Values are wall-clock medians with observed ranges: three independent cold
caches, five warm runs, and three independent caches warmed before adding a
comment to the widely imported `model/timestamp/timestamp.go`. On this machine,
Glint's medians were 11% lower cold, 28% lower warm, and 44% lower after the
edit. These are experimental results from one repository and machine, not a
general performance guarantee.

The selected analyzer names match, but their implementations are not identical.
Glint uses x/tools v0.48 and errcheck v1.20, while golangci-lint v2.11.4 embeds
x/tools v0.43 and errcheck v1.10. Both use Staticcheck v0.7 and ineffassign
v0.2. errcheck v1.20 adds three newer `crypto/sha3` default exclusions.

### `-fix`

`glint` can apply analyzer-produced autofixes.

```console
$ go run ./cmd/glint -fix -diff ./pkg/nolint/testdata # Show changes
--- ./pkg/nolint/testdata/c.go (old)
+++ ./pkg/nolint/testdata/c.go (new)
@@ -5,8 +5,7 @@
 // Should trigger some gosimple fix.
 func example() {
        // Simplifiable code
-       var x int
-       x = 0
+       var x int = 0
        fmt.Println(x)
 
        // Simplifiable loop
$ go run ./cmd/glint -fix ./pkg/nolint/testdata # Apply changes
```

### `nolint` directives

`nolint` directives are blunt instruments for `glint`: adding `//nolint:analyzername` to a file *completely removes* that file from that anlyzer's run. There's no per-line or per-block `nolint`ing here.

You shouldn't be using them anyway.

For example, exempt a file from `nilinterface` linting:

```go
//nolint:nilinterface
package main

// ...
```

### `vscode-go`

VS Code's [`vscode-go` extension](https://github.com/golang/vscode-go) only explicitly supports three lint tools: `golangci-lint`, `revive`, and `staticcheck`. 

You can use a built `glint` binary instead:

```json
{
    "go.lintTool": "glint",
    "go.lintFlags": ["-stdout", "./..."]
}
```

You need the `glint` binary available on your path, e.g. by running `go install ./cmd/glint` in this repo. VS Code may warn you your custom glint-based linter is unsupported. Who cares what they think?

If you open the Go VS Code output, saving a file should trigger a `glint` run and print the findings. For example:

```log
2025-03-05 19:26:58.484 [info] Running checks...
2025-03-05 19:26:58.485 [info] Starting linting the current package at /Users/lukas/Programming/temp/fiber/middleware/etag
2025-03-05 19:27:04.989 [info] /Users/lukas/Programming/temp/fiber/middleware/etag>Finished running tool: /Users/lukas/Programming/glint/glint -alternateTool
2025-03-05 19:27:04.990 [info] /Users/lukas/Programming/temp/fiber/middleware/etag/etag.go:57:12 unchecked error
2025-03-05 19:27:04.990 [info] /Users/lukas/Programming/temp/fiber/middleware/etag/etag.go:61:15 unchecked error
2025-03-05 19:27:04.990 [info] /Users/lukas/Programming/temp/fiber/middleware/etag/etag.go:69:15 unchecked error
2025-03-05 19:27:04.990 [info] /Users/lukas/Programming/temp/fiber/middleware/etag/etag.go:71:15 unchecked error
2025-03-05 19:27:04.990 [info] /Users/lukas/Programming/temp/fiber/middleware/etag/etag_test.go:24:66 nil passed to interface parameter
2025-03-05 19:27:04.990 [info] /Users/lukas/Programming/temp/fiber/middleware/etag/etag_test.go:40:66 nil passed to interface parameter
2025-03-05 19:27:04.990 [info] /Users/lukas/Programming/temp/fiber/middleware/etag/etag_test.go:56:66 nil passed to interface parameter
2025-03-05 19:27:04.990 [info] /Users/lukas/Programming/temp/fiber/middleware/etag/etag_test.go:72:66 nil passed to interface parameter
2025-03-05 19:27:04.990 [info] /Users/lukas/Programming/temp/fiber/middleware/etag/etag_test.go:105:51 nil passed to interface parameter
2025-03-05 19:27:04.990 [info] /Users/lukas/Programming/temp/fiber/middleware/etag/etag_test.go:159:51 nil passed to interface parameter
2025-03-05 19:27:04.990 [info] /Users/lukas/Programming/temp/fiber/middleware/etag/etag_test.go:217:51 nil passed to interface parameter
2025-03-05 19:27:04.990 [info] /Users/lukas/Programming/temp/fiber/middleware/etag/etag_test.go:258:51 nil passed to interface parameter
```
