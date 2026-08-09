# glint

Experimental Go-defined metalinter.

> [!NOTE]
> Glint requires Go 1.26 or newer.

Glint runs ordinary [`analysis.Analyzer`](https://pkg.go.dev/golang.org/x/tools/go/analysis)
values through `go vet`, so analyzer results participate in the Go build cache.
Add a small Go program to your project:

```go
package main

import (
	"github.com/lukasschwab/glint"
	"github.com/lukasschwab/glint/pkg/golangci"
)

func main() {
	glint.Main(golangci.DefaultAnalyzers()...)
}
```

Add, remove, and configure analyzers in Go. See [GitHub Actions](docs/github-actions.md)
for an isolated tools module and cache-aware CI setup.

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

`nolint` directives suppress diagnostics from named analyzers. They filter
reported diagnostics (and therefore their suggested fixes) after the analyzer
has run: files remain analyzer inputs, and results and facts are preserved.
They do not avoid analyzer work or fact production.

The grammar is `//nolint:analyzer[, analyzer...] [// explanation]`.
Whitespace around analyzer names is accepted. A name must exactly match an
analyzer; unknown or malformed names are silent no-ops for now.

Ordinary GolangCI-compatible bare directives infer scope from their placement:

- A directive in the leading comment group attached to a `func`, `type`,
  `const`, `var`, or parenthesized declaration value spec covers that
  declaration's token range.
- A directive after source code on the same physical line covers diagnostics
  on that line.
- An unattached directive, including one at the top of the file, covers the
  whole file.

This deliberately changes one legacy behavior: a bare directive directly
attached to a declaration or after code now scopes narrowly. Move it to an
unattached comment, such as the top of the file, to retain file-wide
suppression.

File scope takes precedence; otherwise any matching line or enclosing
declaration scope suppresses the diagnostic. Directives never apply to another
analyzer. Diagnostics without a valid primary position are never suppressed,
because Glint cannot reliably associate them with source text.

Arbitrary statement/block attachment is deliberately not supported. A
declaration directive must be in the leading comment group ending immediately
before the eligible declaration; comments within a function do not reach
backward or attach to the next statement.

You shouldn't be using them anyway.

For example, exempt a file from `nilinterface` linting:

```go
//nolint:nilinterface
package main

func example() {
	ignoreErr() //nolint:errcheck
}

//nolint:nilinterface // third-party interface contract
func compatible(value any) {
	_ = consume(value)
}
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
