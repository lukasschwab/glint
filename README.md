# glint

Experimental Go-defined metalinter.

## Prospectus

- [x] VS Code integration
- [x] GitHub Action example
    - [ ] Action using a version in a separate repo
- [ ] `nolint` directives
- [ ] Deep-dive the lint scope: compare result sets on go-fiber.

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

You can use a built `glint` binary instead like so:

```json
{
    "go.lintTool": "glint",
    "go.lintFlags": ["-alternateTool"]
}
```

You need the `glint` binary available on your path, e.g. by running `go install ./cmd/glint` in this repo. VS Code may warn you your custom glint-based linter is unsupported. Who cares what they think?

You *must* provide the `-alternateTool` lint flag; this redirects the linter output to `stdout` and injects the `./...` wildcard package selector.

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
