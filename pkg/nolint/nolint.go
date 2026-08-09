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
	file         *ast.File
	fileScoped   bool
	lines        []int
	declarations []tokenRange
}

type tokenRange struct {
	start token.Pos
	end   token.Pos
}

func newIndex(files []*ast.File, fset *token.FileSet, analyzer string) directiveIndex {
	index := directiveIndex{files: make([]fileDirectives, 0, len(files)), fset: fset}
	for _, file := range files {
		entry := fileDirectives{file: file}
		var directives []matchedDirective
		needDeclarations := false
		needCodeEnds := false
		for _, group := range file.Comments {
			for _, comment := range group.List {
				directive, matches := parseDirective(comment.Text, analyzer)
				if !matches {
					continue
				}
				directives = append(directives, matchedDirective{group, comment, directive})
				needDeclarations = needDeclarations || !directive.explicit || directive.scope == declarationScope
				needCodeEnds = needCodeEnds || !directive.explicit
			}
		}
		if len(directives) == 0 {
			continue
		}
		var declarations map[*ast.CommentGroup][]tokenRange
		if needDeclarations {
			declarations = declarationRanges(file)
		}
		var codeEnds map[int]token.Pos
		if needCodeEnds {
			codeEnds = codeEndsByLine(file, fset)
		}
		for _, directive := range directives {
			scope := directive.directive.scope
			if !directive.directive.explicit {
				if ranges := declarations[directive.group]; len(ranges) > 0 {
					scope = declarationScope
				} else if end, ok := codeEnds[fset.PositionFor(directive.comment.Pos(), false).Line]; ok && end <= directive.comment.Pos() {
					scope = lineScope
				}
			}
			switch scope {
			case fileScope:
				entry.fileScoped = true
			case lineScope:
				entry.lines = append(entry.lines, fset.PositionFor(directive.comment.Pos(), false).Line)
			case declarationScope:
				entry.declarations = append(entry.declarations, declarations[directive.group]...)
			}
		}
		if entry.fileScoped || len(entry.lines) > 0 || len(entry.declarations) > 0 {
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
		if pos < entry.file.Pos() || pos > entry.file.End() {
			continue
		}
		if entry.fileScoped {
			return true
		}
		line := index.fset.PositionFor(pos, false).Line
		if slices.Contains(entry.lines, line) {
			return true
		}
		for _, declaration := range entry.declarations {
			if declaration.start <= pos && pos <= declaration.end {
				return true
			}
		}
		return false
	}
	return false
}

type scope uint8

const (
	fileScope scope = iota
	lineScope
	declarationScope
)

type parsedDirective struct {
	scope    scope
	explicit bool
}

type matchedDirective struct {
	group     *ast.CommentGroup
	comment   *ast.Comment
	directive parsedDirective
}

func parseDirective(comment, analyzer string) (parsedDirective, bool) {
	directive, ok := strings.CutPrefix(comment, "//nolint:")
	if !ok {
		return parsedDirective{}, false
	}
	directive = strings.TrimSpace(directive)
	parsed := parsedDirective{scope: fileScope}
	for _, candidate := range []struct {
		prefix string
		scope  scope
	}{
		{"file:", fileScope},
		{"line:", lineScope},
		{"decl:", declarationScope},
	} {
		if rest, ok := strings.CutPrefix(directive, candidate.prefix); ok {
			parsed.scope, parsed.explicit, directive = candidate.scope, true, rest
			break
		}
	}
	if end := strings.Index(directive, "//"); end >= 0 {
		directive = directive[:end]
	}
	for name := range strings.SplitSeq(directive, ",") {
		name = strings.TrimSpace(name)
		if end := strings.IndexAny(name, " \t"); end >= 0 {
			name = name[:end]
		}
		if name == analyzer {
			return parsed, true
		}
	}
	return parsed, false
}

func declarationRanges(file *ast.File) map[*ast.CommentGroup][]tokenRange {
	ranges := make(map[*ast.CommentGroup][]tokenRange)
	ast.Inspect(file, func(node ast.Node) bool {
		switch node := node.(type) {
		case *ast.FuncDecl:
			if node.Doc != nil {
				ranges[node.Doc] = append(ranges[node.Doc], tokenRange{node.Pos(), node.End()})
			}
		case *ast.GenDecl:
			if node.Doc != nil {
				ranges[node.Doc] = append(ranges[node.Doc], tokenRange{node.Pos(), node.End()})
			}
		case *ast.ValueSpec:
			if node.Doc != nil {
				ranges[node.Doc] = append(ranges[node.Doc], tokenRange{node.Pos(), node.End()})
			}
		}
		return true
	})
	return ranges
}

func codeEndsByLine(file *ast.File, fset *token.FileSet) map[int]token.Pos {
	ends := make(map[int]token.Pos)
	ast.Inspect(file, func(node ast.Node) bool {
		if node == nil || node == file {
			return true
		}
		switch node.(type) {
		case *ast.CommentGroup, *ast.Comment:
			return false
		}
		end := node.End()
		line := fset.PositionFor(end, false).Line
		if end > ends[line] {
			ends[line] = end
		}
		return true
	})
	return ends
}
