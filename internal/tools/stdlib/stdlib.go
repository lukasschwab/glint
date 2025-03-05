// Copyright 2022 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:generate go run generate.go

// Package stdlib provides a table of all exported symbols in the
// standard library, along with the version at which they first
// appeared. It also provides the import graph of std packages.
package stdlib

type Symbol struct {
	Name    string
	Kind    Kind
	Version Version // Go version that first included the symbol
}

// A Kind indicates the kind of a symbol:
// function, variable, constant, type, and so on.
type Kind int8

const (
	Invalid Kind = iota // Example name:
	Type                // "Buffer"
	Func                // "Println"
	Var                 // "EOF"
	Const               // "Pi"
	Field               // "Point.X"
	Method              // "(*Buffer).Grow"
)

// A Version represents a version of Go of the form "go1.%d".
type Version int8

var versions [30]string // (increase constant as needed)
