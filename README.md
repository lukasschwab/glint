# glint

Experimental Go-defined metalinter.

## Prospectus

- [ ] `nolint` directives
- [ ] VS Code integration
- [ ] GitHub Action example
- [ ] Deep-dive the lint scope: compare result sets on go-fiber.

### VS Code integration

Have to sub in for `golangci-lint`, `staticcheck`, or `revive`.

```jsonc
{
    "go.lintTool": "revive",
    "go.alternateTools": {
        "revive": "/Users/lukas/Programming/glint/glint",
    },
    "go.lintFlags": ["-stdout", "./..."]
}
```

- `staticcheck` is invoked without arguments.
- `revive` is invoked without arguments.
- `golangci-lint` is invoked with `run --print-issued-lines=false --out-format=colored-line-number --issues-exit-code=0`.
- Additional arguments can be provided with the linter flags option.

I think the best approach is to add some `-pose` flag that makes it act like `staticcheck`: use `./...`, return exit code 0 or 1.
