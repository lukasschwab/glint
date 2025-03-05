// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package myers implements the Myers diff algorithm.
package myers

// opKind is used to denote the type of operation a line represents.
type opKind int

const (
	opDelete opKind = iota // line deleted from input (-)
	opInsert               // line inserted into output (+)
	opEqual                // line present in input and output
)

type operation struct {
	Kind    opKind
	Content []string // content from b
	I1, I2  int      // indices of the line in a
	J1      int      // indices of the line in b, J2 implied by len(Content)
}
