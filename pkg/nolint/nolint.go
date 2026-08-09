// Package nolint filters diagnostics selected by source comments.
package nolint

import (
	"go/ast"
	"go/token"
	"slices"
	"strings"

	"golang.org/x/tools/go/analysis"
)

// Wrap a.Run so directives filter only diagnostics reported by a. The analyzer
// still receives the original Pass, including every file, result, and fact.
func Wrap(a *analysis.Analyzer) {
	run := a.Run
	a.Run = func(pass *analysis.Pass) (any, error) {
		index := newIndex(pass.Files, pass.Fset, a.Name)
		report := pass.Report
		pass.Report = func(diagnostic analysis.Diagnostic) {
			if !index.suppresses(diagnostic.Pos) {
				report(diagnostic)
			}
		}
		defer func() { pass.Report = report }()
		return run(pass)
	}
}

type directiveIndex struct {
	files []fileDirectives
	fset  *token.FileSet
}

type fileDirectives struct {
	tokenFile *token.File
	lines     []lineRange
	scopes    []tokenRange
}

type lineRange struct {
	start int
	end   int
}

type tokenRange struct {
	start token.Pos
	end   token.Pos
}

func newIndex(files []*ast.File, fset *token.FileSet, analyzer string) directiveIndex {
	index := directiveIndex{files: make([]fileDirectives, 0, len(files)), fset: fset}
	for _, file := range files {
		entry := fileDirectives{tokenFile: fset.File(file.Pos())}
		var directives []matchedDirective
		for _, group := range file.Comments {
			for _, comment := range group.List {
				if !suppressesAnalyzer(comment.Text, analyzer) {
					continue
				}
				directives = append(directives, matchedDirective{group, comment})
			}
		}
		if len(directives) == 0 {
			continue
		}
		codePositions, attached := indexNodes(file, fset, directives)
		for _, directive := range directives {
			groupStart := fset.PositionFor(directive.group.Pos(), false).Line
			groupEnd := fset.PositionFor(directive.group.End(), false).Line
			entry.lines = append(entry.lines, lineRange{groupStart, groupEnd})

			commentLine := fset.PositionFor(directive.comment.Pos(), false).Line
			if position, ok := codePositions[commentLine]; ok && position <= directive.comment.Pos() {
				continue
			}
			entry.scopes = append(entry.scopes, attached[directive.group]...)
		}
		if len(entry.lines) > 0 || len(entry.scopes) > 0 {
			index.files = append(index.files, entry)
		}
	}
	return index
}

func (index directiveIndex) suppresses(pos token.Pos) bool {
	if !pos.IsValid() {
		return false
	}
	for _, entry := range index.files {
		if pos < token.Pos(entry.tokenFile.Base()) || pos > token.Pos(entry.tokenFile.Base()+entry.tokenFile.Size()) {
			continue
		}
		line := index.fset.PositionFor(pos, false).Line
		if slices.ContainsFunc(entry.lines, func(scope lineRange) bool {
			return scope.start <= line && line <= scope.end
		}) {
			return true
		}
		for _, scope := range entry.scopes {
			if scope.start <= pos && pos <= scope.end {
				return true
			}
		}
		return false
	}
	return false
}

type matchedDirective struct {
	group   *ast.CommentGroup
	comment *ast.Comment
}

func suppressesAnalyzer(comment, analyzer string) bool {
	directive, ok := strings.CutPrefix(comment, "//nolint:")
	if !ok {
		return false
	}
	directive = strings.TrimSpace(directive)
	if end := strings.Index(directive, "//"); end >= 0 {
		directive = directive[:end]
	}
	matches := false
	for name := range strings.SplitSeq(directive, ",") {
		name = strings.TrimSpace(name)
		if name == "" || strings.ContainsAny(name, " \t") {
			return false
		}
		if name == analyzer {
			matches = true
		}
	}
	return matches
}

type attachmentPoint struct {
	line   int
	column int
}

func indexNodes(file *ast.File, fset *token.FileSet, directives []matchedDirective) (map[int]token.Pos, map[*ast.CommentGroup][]tokenRange) {
	groups := make(map[attachmentPoint][]*ast.CommentGroup)
	seen := make(map[*ast.CommentGroup]bool)
	for _, directive := range directives {
		if seen[directive.group] {
			continue
		}
		seen[directive.group] = true
		start := fset.PositionFor(directive.group.Pos(), false)
		end := fset.PositionFor(directive.group.End(), false)
		point := attachmentPoint{line: end.Line + 1, column: start.Column}
		groups[point] = append(groups[point], directive.group)
	}

	positions := make(map[int]token.Pos)
	tokenFile := fset.File(file.Pos())
	attached := make(map[*ast.CommentGroup][]tokenRange)
	ast.Inspect(file, func(node ast.Node) bool {
		if node == nil {
			return true
		}
		switch node.(type) {
		case *ast.CommentGroup, *ast.Comment:
			return false
		}

		start := fset.PositionFor(node.Pos(), false)
		for _, group := range groups[attachmentPoint{line: start.Line, column: start.Column}] {
			if node == file {
				attached[group] = append(attached[group], tokenRange{
					start: token.Pos(tokenFile.Base()),
					end:   token.Pos(tokenFile.Base() + tokenFile.Size()),
				})
			} else {
				attached[group] = append(attached[group], tokenRange{node.Pos(), node.End()})
			}
		}
		if node == file {
			return true
		}

		if node.Pos() > positions[start.Line] {
			positions[start.Line] = node.Pos()
		}
		end := node.End()
		line := fset.PositionFor(end, false).Line
		if end > positions[line] {
			positions[line] = end
		}
		return true
	})
	return positions, attached
}
