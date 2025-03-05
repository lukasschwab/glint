// Copyright 2020 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package analysisinternal provides gopls' internal analyses with a
// number of helper functions that operate on typed syntax trees.
package analysisinternal

// A ReadFileFunc is a function that returns the
// contents of a file, such as [os.ReadFile].
type ReadFileFunc = func(filename string) ([]byte, error)
