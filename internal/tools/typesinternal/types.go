// Copyright 2020 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package typesinternal provides access to internal go/types APIs that are not
// yet exported.
package typesinternal

import (
	"go/types"
)

// A NamedOrAlias is a [types.Type] that is named (as
// defined by the spec) and capable of bearing type parameters: it
// abstracts aliases ([types.Alias]) and defined types
// ([types.Named]).
//
// Every type declared by an explicit "type" declaration is a
// NamedOrAlias. (Built-in type symbols may additionally
// have type [types.Basic], which is not a NamedOrAlias,
// though the spec regards them as "named".)
//
// NamedOrAlias cannot expose the Origin method, because
// [types.Alias.Origin] and [types.Named.Origin] have different
// (covariant) result types; use [Origin] instead.
type NamedOrAlias interface {
	types.Type
	Obj() *types.TypeName
	// TODO(hxjiang): add method TypeArgs() *types.TypeList after stop supporting go1.22.
}
