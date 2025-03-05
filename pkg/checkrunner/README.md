# checkrunner

Partial fork of [golang.org/x/tools/go/analysis/internal/checker](https://pkg.go.dev/golang.org/x/tools/go/analysis/internal/checker) with modification for use by [glint](../../glint.go).

Like the upstream package, this handles command-line flags and output for compatibility with `go vet -vettool`.

Unlike the upstream package, it does not control an `analyzer.Analyzer` set and delegates analyzer execution to a received `Runnable` function:

```go
type Runnable func(opts *checker.Options) (*checker.Graph, error)
```

This allows split `checker.Run` calls for different sets of analyzers.
