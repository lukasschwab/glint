package glint

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/lukasschwab/glint/pkg/nolint"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/unitchecker"
)

const unitcheckerInnerEnv = "GLINT_UNITCHECKER_INNER"

var cacheScope string

// Main runs analyzers as a go vet compatible tool. User invocations are
// delegated to go vet so compilation, facts, and diagnostics all use Go's
// content-addressed build cache.
func Main(analyzers ...*analysis.Analyzer) {
	progname := filepath.Base(os.Args[0])
	log.SetFlags(0)
	log.SetPrefix(progname + ": ")

	for _, analyzer := range analyzers {
		nolint.Wrap(analyzer)
	}
	if err := analysis.Validate(analyzers); err != nil {
		log.Fatal(err)
	}

	// This no-op flag is included in Go's vet action key. It keeps a package's
	// dependency-only (VetxOnly) result from satisfying a later invocation in
	// which that package is an explicit diagnostic root.
	flag.StringVar(&cacheScope, "glint.scope", "", "internal cache scope")

	if os.Getenv(unitcheckerInnerEnv) == "1" || isUnitcheckerProtocolInvocation(os.Args[1:]) {
		unitchecker.Main(analyzers...)
		return
	}

	if index, configFile, ok := findConfigArgument(os.Args[1:]); ok {
		os.Exit(runUnitcheckerWrapper(index, configFile))
	}

	os.Exit(runUserCommand(os.Args[1:]))
}

func isUnitcheckerProtocolInvocation(args []string) bool {
	if len(args) == 0 {
		return false
	}
	if args[0] == "help" {
		return true
	}
	for _, arg := range args {
		if arg == "-flags" || strings.HasPrefix(arg, "-V=") {
			return true
		}
	}
	return false
}

func findConfigArgument(args []string) (int, string, bool) {
	for index, arg := range args {
		if strings.HasSuffix(arg, ".cfg") {
			return index, arg, true
		}
	}
	return 0, "", false
}

type userOptions struct {
	args      []string
	json      bool
	stdout    bool
	fix       bool
	diff      bool
	showUsage bool
}

func parseUserOptions(args []string) (userOptions, error) {
	var options userOptions
	for index := 0; index < len(args); index++ {
		arg := args[index]
		switch {
		case arg == "-stdout":
			options.stdout = true
		case strings.HasPrefix(arg, "-stdout="):
			value, err := strconv.ParseBool(strings.TrimPrefix(arg, "-stdout="))
			if err != nil {
				return options, fmt.Errorf("invalid value for -stdout: %w", err)
			}
			options.stdout = value
		case arg == "-json":
			options.json = true
		case strings.HasPrefix(arg, "-json="):
			value, err := strconv.ParseBool(strings.TrimPrefix(arg, "-json="))
			if err != nil {
				return options, fmt.Errorf("invalid value for -json: %w", err)
			}
			options.json = value
		case arg == "-alternateTool":
			// vscode-go adds this compatibility flag to alternate linters.
		case arg == "-fix":
			options.fix = true
			options.args = append(options.args, arg)
		case strings.HasPrefix(arg, "-fix="):
			value, err := strconv.ParseBool(strings.TrimPrefix(arg, "-fix="))
			if err != nil {
				return options, fmt.Errorf("invalid value for -fix: %w", err)
			}
			options.fix = value
			options.args = append(options.args, arg)
		case arg == "-diff":
			options.diff = true
			options.args = append(options.args, arg)
		case strings.HasPrefix(arg, "-diff="):
			value, err := strconv.ParseBool(strings.TrimPrefix(arg, "-diff="))
			if err != nil {
				return options, fmt.Errorf("invalid value for -diff: %w", err)
			}
			options.diff = value
			options.args = append(options.args, arg)
		case arg == "-test":
			// go vet always analyzes test files. Do not translate this
			// multichecker compatibility flag to -tests: a linter may have
			// an analyzer named "tests", and enabling any analyzer by name
			// makes unitchecker disable every other analyzer.
		case strings.HasPrefix(arg, "-test="):
			value, err := strconv.ParseBool(strings.TrimPrefix(arg, "-test="))
			if err != nil {
				return options, fmt.Errorf("invalid value for -test: %w", err)
			}
			if !value {
				return options, fmt.Errorf("-test=false is not supported by the go vet backend")
			}
		case arg == "-trimpath" || strings.HasPrefix(arg, "-trimpath="):
			// Glint always enables trimpath so action keys can be shared safely
			// between worktrees. Cached paths are normalized separately.
		case arg == "-h" || arg == "-help" || arg == "--help":
			options.showUsage = true
		default:
			options.args = append(options.args, arg)
		}
	}
	return options, nil
}

func runUserCommand(args []string) int {
	options, err := parseUserOptions(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "glint: %v\n", err)
		return 2
	}
	if options.showUsage || len(options.args) == 0 {
		printUsage(os.Stderr)
		if options.showUsage {
			return 0
		}
		return 2
	}

	executable, err := os.Executable()
	if err != nil {
		fmt.Fprintf(os.Stderr, "glint: locate executable: %v\n", err)
		return 1
	}
	if evaluated, err := filepath.EvalSymlinks(executable); err == nil {
		executable = evaluated
	}

	workingDirectoryIdentity, err := cacheWorkingDirectoryIdentity()
	if err != nil {
		fmt.Fprintf(os.Stderr, "glint: identify working directory for cache: %v\n", err)
		return 1
	}
	scopeArguments := append(append([]string(nil), options.args...), "glint-working-directory="+workingDirectoryIdentity)
	if options.fix && !options.diff {
		workingDirectory, err := os.Getwd()
		if err != nil {
			fmt.Fprintf(os.Stderr, "glint: locate working directory for fixes: %v\n", err)
			return 1
		}
		scopeArguments = append(scopeArguments, "glint-fix-directory="+workingDirectory)
	}
	scope := invocationScope(scopeArguments)
	goArgs := []string{
		"vet",
		"-trimpath",
		"-vettool=" + executable,
		"-glint.scope=" + scope,
	}
	structuredOutput := !options.fix && !options.diff
	if structuredOutput {
		goArgs = append(goArgs, "-json")
	}
	goArgs = append(goArgs, options.args...)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	command := exec.Command("go", goArgs...)
	command.Stdin = os.Stdin
	command.Stdout = &stdout
	command.Stderr = &stderr
	runErr := command.Run()

	if stderr.Len() > 0 {
		_, _ = io.Copy(os.Stderr, &stderr)
	}

	output := stdout.Bytes()
	if len(output) > 0 && structuredOutput {
		output, err = rebaseCachedOutput(output)
		if err != nil {
			fmt.Fprintf(os.Stderr, "glint: rebase cached diagnostics: %v\n", err)
			return 1
		}
	}

	if !structuredOutput {
		if len(output) > 0 {
			output, err = rebaseCachedText(output)
			if err != nil {
				fmt.Fprintf(os.Stderr, "glint: rebase cached fix output: %v\n", err)
				return 1
			}
			_, _ = os.Stdout.Write(output)
		}
	} else if options.json {
		if len(output) > 0 {
			formatted, err := mergeAndFormatJSON(output)
			if err != nil {
				fmt.Fprintf(os.Stderr, "glint: parse analyzer JSON: %v\n", err)
				return 1
			}
			_, _ = os.Stdout.Write(formatted)
		}
	} else if len(output) > 0 {
		sink := io.Writer(os.Stderr)
		if options.stdout {
			sink = os.Stdout
		}
		diagnostics, analysisErrors, err := printTextOutput(sink, output)
		if err != nil {
			fmt.Fprintf(os.Stderr, "glint: parse analyzer JSON: %v\n", err)
			return 1
		}
		if analysisErrors > 0 {
			return 1
		}
		if diagnostics > 0 && runErr == nil {
			return 3
		}
	}

	if runErr != nil {
		var exitError *exec.ExitError
		if errors.As(runErr, &exitError) {
			return exitError.ExitCode()
		}
		fmt.Fprintf(os.Stderr, "glint: run go vet: %v\n", runErr)
		return 1
	}
	return 0
}

func invocationScope(args []string) string {
	hash := sha256.New()
	for _, arg := range args {
		_, _ = io.WriteString(hash, arg)
		_, _ = hash.Write([]byte{0})
	}
	return fmt.Sprintf("%x", hash.Sum(nil))
}

func printUsage(writer io.Writer) {
	_, _ = fmt.Fprintf(writer, `glint is a cached Go analysis driver.

Usage: glint [-flag] [package]

Run 'glint help' for analyzer details and flags.
`)
}
