package glint

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
)

func TestCachedBackendCorrectness(t *testing.T) {
	if testing.Short() {
		t.Skip("builds and runs an integration-test vettool")
	}

	repository := repositoryRoot(t)
	binary := filepath.Join(t.TempDir(), "cacheprobe")
	buildCache := t.TempDir()
	command := exec.Command("go", "build", "-o", binary, "./internal/testcmd/cacheprobe")
	command.Dir = repository
	command.Env = environmentWith(os.Environ(), "GOCACHE", buildCache)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("build cacheprobe: %v\n%s", err, output)
	}

	t.Run("dependency result cannot poison later root", func(t *testing.T) {
		module := newProbeModule(t, filepath.Join(t.TempDir(), "target"), true)
		cache := t.TempDir()

		rootOutput := runProbe(t, binary, module, cache, "./root")
		if !strings.Contains(rootOutput, "cache probe for example.com/target/root") {
			t.Fatalf("root output missing diagnostic:\n%s", rootOutput)
		}
		depOutput := runProbe(t, binary, module, cache, "./dep")
		if !strings.Contains(depOutput, "cache probe for example.com/target/dep") {
			t.Fatalf("dependency output was poisoned by VetxOnly cache entry:\n%s", depOutput)
		}
		warmOutput := runProbe(t, binary, module, cache, "./dep")
		if warmOutput != depOutput {
			t.Fatalf("warm diagnostics changed:\nfirst:\n%s\nwarm:\n%s", depOutput, warmOutput)
		}
	})

	t.Run("relative targets include their semantic working directory", func(t *testing.T) {
		module := newProbeModule(t, filepath.Join(t.TempDir(), "target"), true)
		cache := t.TempDir()

		rootOutput := runProbe(t, binary, filepath.Join(module, "root"), cache, "./...")
		if !strings.Contains(rootOutput, "cache probe for example.com/target/root") {
			t.Fatalf("subdirectory output missing diagnostic:\n%s", rootOutput)
		}
		moduleOutput := runProbe(t, binary, module, cache, "./...")
		if !strings.Contains(moduleOutput, "cache probe for example.com/target/dep") {
			t.Fatalf("module-root output reused subdirectory VetxOnly entry:\n%s", moduleOutput)
		}
	})

	t.Run("shared cache rebases paths between worktrees", func(t *testing.T) {
		base := t.TempDir()
		first := newProbeModule(t, filepath.Join(base, "first"), false)
		second := newProbeModule(t, filepath.Join(base, "second"), false)
		cache := t.TempDir()

		firstOutput := runProbe(t, binary, first, cache, "./...")
		secondOutput := runProbe(t, binary, second, cache, "./...")
		if !strings.Contains(firstOutput, filepath.Join(first, "dep", "dep.go")) {
			t.Fatalf("first output has wrong path:\n%s", firstOutput)
		}
		if !strings.Contains(secondOutput, filepath.Join(second, "dep", "dep.go")) {
			t.Fatalf("second output was not rebased:\n%s", secondOutput)
		}
		if strings.Contains(secondOutput, first) {
			t.Fatalf("second output leaked first worktree path:\n%s", secondOutput)
		}
	})

	t.Run("concurrent worktrees can publish to one cache", func(t *testing.T) {
		base := t.TempDir()
		first := newProbeModule(t, filepath.Join(base, "first"), false)
		second := newProbeModule(t, filepath.Join(base, "second"), false)
		cache := t.TempDir()

		type result struct {
			directory string
			output    string
			err       error
		}
		results := make(chan result, 2)
		var wait sync.WaitGroup
		for _, directory := range []string{first, second} {
			wait.Add(1)
			go func() {
				defer wait.Done()
				output, err := executeProbe(binary, directory, cache, "./...")
				results <- result{directory: directory, output: output, err: err}
			}()
		}
		wait.Wait()
		close(results)

		for result := range results {
			if !isDiagnosticExit(result.err) {
				t.Fatalf("probe in %s: %v\n%s", result.directory, result.err, result.output)
			}
			if !strings.Contains(result.output, filepath.Join(result.directory, "dep", "dep.go")) {
				t.Fatalf("concurrent output has wrong path for %s:\n%s", result.directory, result.output)
			}
		}
	})

	t.Run("applying fixes is scoped to each worktree", func(t *testing.T) {
		base := t.TempDir()
		first := newProbeModule(t, filepath.Join(base, "first"), false)
		second := newProbeModule(t, filepath.Join(base, "second"), false)
		cache := t.TempDir()

		for _, directory := range []string{first, second} {
			output, err := executeProbe(binary, directory, cache, "-fix", "./dep")
			if err != nil {
				t.Fatalf("apply fix in %s: %v\n%s", directory, err, output)
			}
			content, err := os.ReadFile(filepath.Join(directory, "dep", "dep.go"))
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(content), "const Value = 2") {
				t.Fatalf("fix was applied to the wrong worktree; %s contains:\n%s", directory, content)
			}
		}
	})
}

func repositoryRoot(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("locate integration test")
	}
	return filepath.Dir(filename)
}

func newProbeModule(t *testing.T, directory string, withRoot bool) string {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(directory, "dep"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, filepath.Join(directory, "go.mod"), "module example.com/target\n\ngo 1.25.0\n")
	writeTestFile(t, filepath.Join(directory, "dep", "dep.go"), "package dep\n\nconst Value = 1\n")
	if withRoot {
		if err := os.MkdirAll(filepath.Join(directory, "root"), 0o755); err != nil {
			t.Fatal(err)
		}
		writeTestFile(t, filepath.Join(directory, "root", "root.go"), "package root\n\nimport \"example.com/target/dep\"\n\nconst Value = dep.Value\n")
	}
	return directory
}

func writeTestFile(t *testing.T, filename, content string) {
	t.Helper()
	if err := os.WriteFile(filename, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func runProbe(t *testing.T, binary, directory, cache string, patterns ...string) string {
	t.Helper()
	output, err := executeProbe(binary, directory, cache, patterns...)
	if !isDiagnosticExit(err) {
		t.Fatalf("run cacheprobe: %v\n%s", err, output)
	}
	return output
}

func executeProbe(binary, directory, cache string, patterns ...string) (string, error) {
	args := append([]string{"-stdout"}, patterns...)
	command := exec.Command(binary, args...)
	command.Dir = directory
	command.Env = environmentWith(os.Environ(), "GOCACHE", cache)
	var output bytes.Buffer
	command.Stdout = &output
	command.Stderr = &output
	err := command.Run()
	return output.String(), err
}

func isDiagnosticExit(err error) bool {
	var exitError *exec.ExitError
	return errors.As(err, &exitError) && exitError.ExitCode() == 3
}

func environmentWith(environment []string, name, value string) []string {
	prefix := name + "="
	result := make([]string, 0, len(environment)+1)
	for _, item := range environment {
		if !strings.HasPrefix(item, prefix) {
			result = append(result, item)
		}
	}
	return append(result, prefix+value)
}
