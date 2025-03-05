// Copyright 2020 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package analysisinternal provides gopls' internal analyses with a
// number of helper functions that operate on typed syntax trees.
package analysisinternal

import (
	"cmp"
	"fmt"
	"go/ast"
	"go/printer"
	"go/token"
	"go/types"
	pathpkg "path"
	"slices"
	"strings"

	"golang.org/x/tools/go/analysis"
)

// A ReadFileFunc is a function that returns the
// contents of a file, such as [os.ReadFile].
type ReadFileFunc = func(filename string) ([]byte, error)

// CheckedReadFile returns a wrapper around a Pass.ReadFile
// function that performs the appropriate checks.
func CheckedReadFile(pass *analysis.Pass, readFile ReadFileFunc) ReadFileFunc {
	return func(filename string) ([]byte, error) {
		if err := CheckReadable(pass, filename); err != nil {
			return nil, err
		}
		return readFile(filename)
	}
}

// CheckReadable enforces the access policy defined by the ReadFile field of [analysis.Pass].
func CheckReadable(pass *analysis.Pass, filename string) error {
	if slices.Contains(pass.OtherFiles, filename) ||
		slices.Contains(pass.IgnoredFiles, filename) {
		return nil
	}
	for _, f := range pass.Files {
		if pass.Fset.File(f.FileStart).Name() == filename {
			return nil
		}
	}
	return fmt.Errorf("Pass.ReadFile: %s is not among OtherFiles, IgnoredFiles, or names of Files", filename)
}

// AddImport checks whether this file already imports pkgpath and
// that import is in scope at pos. If so, it returns the name under
// which it was imported and a zero edit. Otherwise, it adds a new
// import of pkgpath, using a name derived from the preferred name,
// and returns the chosen name, a prefix to be concatenated with member
// to form a qualified name, and the edit for the new import.
//
// In the special case that pkgpath is dot-imported then member, the
// identifer for which the import is being added, is consulted. If
// member is not shadowed at pos, AddImport returns (".", "", nil).
// (AddImport accepts the caller's implicit claim that the imported
// package declares member.)
//
// It does not mutate its arguments.
func AddImport(info *types.Info, file *ast.File, preferredName, pkgpath, member string, pos token.Pos) (name, prefix string, newImport []analysis.TextEdit) {
	// Find innermost enclosing lexical block.
	scope := info.Scopes[file].Innermost(pos)
	if scope == nil {
		panic("no enclosing lexical block")
	}

	// Is there an existing import of this package?
	// If so, are we in its scope? (not shadowed)
	for _, spec := range file.Imports {
		pkgname := info.PkgNameOf(spec)
		if pkgname != nil && pkgname.Imported().Path() == pkgpath {
			name = pkgname.Name()
			if name == "." {
				// The scope of ident must be the file scope.
				if s, _ := scope.LookupParent(member, pos); s == info.Scopes[file] {
					return name, "", nil
				}
			} else if _, obj := scope.LookupParent(name, pos); obj == pkgname {
				return name, name + ".", nil
			}
		}
	}

	// We must add a new import.
	// Ensure we have a fresh name.
	newName := preferredName
	for i := 0; ; i++ {
		if _, obj := scope.LookupParent(newName, pos); obj == nil {
			break // fresh
		}
		newName = fmt.Sprintf("%s%d", preferredName, i)
	}

	// Create a new import declaration either before the first existing
	// declaration (which must exist), including its comments; or
	// inside the declaration, if it is an import group.
	//
	// Use a renaming import whenever the preferred name is not
	// available, or the chosen name does not match the last
	// segment of its path.
	newText := fmt.Sprintf("%q", pkgpath)
	if newName != preferredName || newName != pathpkg.Base(pkgpath) {
		newText = fmt.Sprintf("%s %q", newName, pkgpath)
	}
	decl0 := file.Decls[0]
	var before ast.Node = decl0
	switch decl0 := decl0.(type) {
	case *ast.GenDecl:
		if decl0.Doc != nil {
			before = decl0.Doc
		}
	case *ast.FuncDecl:
		if decl0.Doc != nil {
			before = decl0.Doc
		}
	}
	// If the first decl is an import group, add this new import at the end.
	if gd, ok := before.(*ast.GenDecl); ok && gd.Tok == token.IMPORT && gd.Rparen.IsValid() {
		pos = gd.Rparen
		newText = "\t" + newText + "\n"
	} else {
		pos = before.Pos()
		newText = "import " + newText + "\n\n"
	}
	return newName, newName + ".", []analysis.TextEdit{{
		Pos:     pos,
		End:     pos,
		NewText: []byte(newText),
	}}
}

// Format returns a string representation of the expression e.
func Format(fset *token.FileSet, e ast.Expr) string {
	var buf strings.Builder
	printer.Fprint(&buf, fset, e) // ignore errors
	return buf.String()
}

// Imports returns true if path is imported by pkg.
func Imports(pkg *types.Package, path string) bool {
	for _, imp := range pkg.Imports() {
		if imp.Path() == path {
			return true
		}
	}
	return false
}

// ValidateFixes validates the set of fixes for a single diagnostic.
// Any error indicates a bug in the originating analyzer.
//
// It updates fixes so that fixes[*].End.IsValid().
//
// It may be used as part of an analysis driver implementation.
func ValidateFixes(fset *token.FileSet, a *analysis.Analyzer, fixes []analysis.SuggestedFix) error {
	fixMessages := make(map[string]bool)
	for i := range fixes {
		fix := &fixes[i]
		if fixMessages[fix.Message] {
			return fmt.Errorf("analyzer %q suggests two fixes with same Message (%s)", a.Name, fix.Message)
		}
		fixMessages[fix.Message] = true
		if err := validateFix(fset, fix); err != nil {
			return fmt.Errorf("analyzer %q suggests invalid fix (%s): %v", a.Name, fix.Message, err)
		}
	}
	return nil
}

// validateFix validates a single fix.
// Any error indicates a bug in the originating analyzer.
//
// It updates fix so that fix.End.IsValid().
func validateFix(fset *token.FileSet, fix *analysis.SuggestedFix) error {

	// Stably sort edits by Pos. This ordering puts insertions
	// (end = start) before deletions (end > start) at the same
	// point, but uses a stable sort to preserve the order of
	// multiple insertions at the same point.
	slices.SortStableFunc(fix.TextEdits, func(x, y analysis.TextEdit) int {
		if sign := cmp.Compare(x.Pos, y.Pos); sign != 0 {
			return sign
		}
		return cmp.Compare(x.End, y.End)
	})

	var prev *analysis.TextEdit
	for i := range fix.TextEdits {
		edit := &fix.TextEdits[i]

		// Validate edit individually.
		start := edit.Pos
		file := fset.File(start)
		if file == nil {
			return fmt.Errorf("no token.File for TextEdit.Pos (%v)", edit.Pos)
		}
		if end := edit.End; end.IsValid() {
			if end < start {
				return fmt.Errorf("TextEdit.Pos (%v) > TextEdit.End (%v)", edit.Pos, edit.End)
			}
			endFile := fset.File(end)
			if endFile == nil {
				return fmt.Errorf("no token.File for TextEdit.End (%v; File(start).FileEnd is %d)", end, file.Base()+file.Size())
			}
			if endFile != file {
				return fmt.Errorf("edit #%d spans files (%v and %v)",
					i, file.Position(edit.Pos), endFile.Position(edit.End))
			}
		} else {
			edit.End = start // update the SuggestedFix
		}
		if eof := token.Pos(file.Base() + file.Size()); edit.End > eof {
			return fmt.Errorf("end is (%v) beyond end of file (%v)", edit.End, eof)
		}

		// Validate the sequence of edits:
		// properly ordered, no overlapping deletions
		if prev != nil && edit.Pos < prev.End {
			xpos := fset.Position(prev.Pos)
			xend := fset.Position(prev.End)
			ypos := fset.Position(edit.Pos)
			yend := fset.Position(edit.End)
			return fmt.Errorf("overlapping edits to %s (%d:%d-%d:%d and %d:%d-%d:%d)",
				xpos.Filename,
				xpos.Line, xpos.Column,
				xend.Line, xend.Column,
				ypos.Line, ypos.Column,
				yend.Line, yend.Column,
			)
		}
		prev = edit
	}

	return nil
}

// CanImport reports whether one package is allowed to import another.
//
// TODO(adonovan): allow customization of the accessibility relation
// (e.g. for Bazel).
func CanImport(from, to string) bool {
	// TODO(adonovan): better segment hygiene.
	if to == "internal" || strings.HasPrefix(to, "internal/") {
		// Special case: only std packages may import internal/...
		// We can't reliably know whether we're in std, so we
		// use a heuristic on the first segment.
		first, _, _ := strings.Cut(from, "/")
		if strings.Contains(first, ".") {
			return false // example.com/foo ∉ std
		}
		if first == "testdata" {
			return false // testdata/foo ∉ std
		}
	}
	if strings.HasSuffix(to, "/internal") {
		return strings.HasPrefix(from, to[:len(to)-len("/internal")])
	}
	if i := strings.LastIndex(to, "/internal/"); i >= 0 {
		return strings.HasPrefix(from, to[:i])
	}
	return true
}
