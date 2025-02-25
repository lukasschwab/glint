# glint

*Your* Go metalinter.

## Prospectus

You're getting sidetracked by overfocusing on golangci-lint's cache implementation.

1. Focus on just caching diagnostics, not facts or return values.

2. Consider using the Go cache directly. 

    + Consider using the published Go cache package rather than the golangci-lint modified one: only difference is the env variable defining cache location.

3. To get diagnostics, provide a custom analysis.Pass; this might effectively mean implementing a new lightweight runner. Look at the implementation of multichecker.

I think I need to limit the search depth; seems to run away and lint internal Go stuff on the first pass. Achieved by changing the package load mode and segmenting the analyzers.

Next up: cache the package hashes. This gave me some tricky memory errors last time, but I can definitely solve those.
