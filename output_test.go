package glint

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/tools/go/analysis/unitchecker"
)

func TestMainModuleRoot(t *testing.T) {
	t.Parallel()

	root := filepath.Join(string(filepath.Separator), "work", "module")
	config := &unitchecker.Config{
		Dir:           filepath.Join(root, "internal", "thing"),
		ImportPath:    "example.com/project/internal/thing",
		ModulePath:    "example.com/project",
		ModuleVersion: "",
	}
	if got := mainModuleRoot(config); got != root {
		t.Fatalf("mainModuleRoot() = %q, want %q", got, root)
	}

	config.ModuleVersion = "v1.2.3"
	if got := mainModuleRoot(config); got != "" {
		t.Fatalf("dependency mainModuleRoot() = %q, want empty", got)
	}
}

func TestNormalizeAndRebaseCachedOutput(t *testing.T) {
	t.Parallel()

	rootA := filepath.Join(string(filepath.Separator), "work", "a")
	rootB := filepath.Join(string(filepath.Separator), "work", "b")
	filenameA := filepath.Join(rootA, "pkg", "file.go")
	input := jsonTree{
		"example.com/project/pkg": {
			"probe": json.RawMessage(`[{` +
				`"posn":` + quoted(filenameA+":3:4") + `,` +
				`"end":` + quoted(filenameA+":3:7") + `,` +
				`"message":"problem",` +
				`"related":[{"posn":` + quoted(filenameA+":1:1") + `,"end":` + quoted(filenameA+":1:2") + `,"message":"context"}],` +
				`"suggested_fixes":[{"message":"fix","edits":[{"filename":` + quoted(filenameA) + `,"start":0,"end":1,"new":` + quoted(filenameA) + `}]}]` +
				`}]`),
		},
	}
	raw, err := json.Marshal(input)
	if err != nil {
		t.Fatal(err)
	}

	normalized, err := normalizeCachedOutput(raw, "example.com/project", rootA)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(normalized), `"posn": "`+rootA) ||
		strings.Contains(string(normalized), `"filename": "`+rootA) {
		t.Fatalf("normalized output retained module root in a path field:\n%s", normalized)
	}
	if !strings.Contains(string(normalized), normalizedModulePrefix) {
		t.Fatalf("normalized output has no module marker:\n%s", normalized)
	}
	// Suggested replacement text is source content, not a path-bearing field.
	if !strings.Contains(string(normalized), quoted(filenameA)) {
		t.Fatalf("normalization changed suggested replacement text:\n%s", normalized)
	}

	rebased, err := rewriteJSONStream(normalized, func(key, value string) string {
		if key == "new" {
			return value
		}
		return rebaseMarkers(value, map[string]string{"example.com/project": rootB})
	})
	if err != nil {
		t.Fatal(err)
	}
	filenameB := filepath.Join(rootB, "pkg", "file.go")
	if !strings.Contains(string(rebased), filenameB) {
		t.Fatalf("rebased output does not contain current path %q:\n%s", filenameB, rebased)
	}
	if strings.Contains(string(rebased), normalizedModulePrefix) {
		t.Fatalf("rebased output retained module marker:\n%s", rebased)
	}
}

func TestSplitPositionWithWindowsDrive(t *testing.T) {
	t.Parallel()

	filename, suffix := splitPosition(`C:\work\project\file.go:12:34`)
	if filename != `C:\work\project\file.go` || suffix != ":12:34" {
		t.Fatalf("splitPosition() = (%q, %q)", filename, suffix)
	}
}

func TestInvocationScope(t *testing.T) {
	t.Parallel()

	first := invocationScope([]string{"-unused=false", "./..."})
	if second := invocationScope([]string{"-unused=false", "./..."}); first != second {
		t.Fatalf("same arguments produced different scopes: %q != %q", first, second)
	}
	if other := invocationScope([]string{"-unused=false", "./pkg"}); first == other {
		t.Fatalf("different target sets produced the same scope: %q", first)
	}
}

func quoted(value string) string {
	encoded, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return string(encoded)
}
