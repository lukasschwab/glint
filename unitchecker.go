package glint

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"golang.org/x/tools/go/analysis/unitchecker"
)

const normalizedModulePrefix = "glint-module:"

func runUnitcheckerWrapper(configIndex int, configFile string) int {
	data, err := os.ReadFile(configFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "glint: read vet config: %v\n", err)
		return 1
	}

	var config unitchecker.Config
	if err := json.Unmarshal(data, &config); err != nil {
		fmt.Fprintf(os.Stderr, "glint: decode vet config: %v\n", err)
		return 1
	}
	var configFields map[string]json.RawMessage
	if err := json.Unmarshal(data, &configFields); err != nil {
		fmt.Fprintf(os.Stderr, "glint: preserve vet config fields: %v\n", err)
		return 1
	}
	if config.Stdout == "" {
		return runInnerUnitchecker(configIndex, configFile)
	}

	rawOutput, err := os.CreateTemp(filepath.Dir(config.Stdout), "glint-raw-*.stdout")
	if err != nil {
		fmt.Fprintf(os.Stderr, "glint: create raw stdout file: %v\n", err)
		return 1
	}
	rawOutputPath := rawOutput.Name()
	if err := rawOutput.Close(); err != nil {
		fmt.Fprintf(os.Stderr, "glint: close raw stdout file: %v\n", err)
		return 1
	}
	defer removeTemporaryFile(rawOutputPath)

	originalStdout := config.Stdout
	encodedOutputPath, err := json.Marshal(rawOutputPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "glint: encode raw stdout path: %v\n", err)
		return 1
	}
	configFields["Stdout"] = encodedOutputPath
	modified, err := json.MarshalIndent(configFields, "", "\t")
	if err != nil {
		fmt.Fprintf(os.Stderr, "glint: encode vet config: %v\n", err)
		return 1
	}
	modified = append(modified, '\n')

	innerConfig, err := os.CreateTemp(filepath.Dir(configFile), "glint-*.cfg")
	if err != nil {
		fmt.Fprintf(os.Stderr, "glint: create inner vet config: %v\n", err)
		return 1
	}
	innerConfigPath := innerConfig.Name()
	if _, err := innerConfig.Write(modified); err != nil {
		_ = innerConfig.Close()
		fmt.Fprintf(os.Stderr, "glint: write inner vet config: %v\n", err)
		return 1
	}
	if err := innerConfig.Close(); err != nil {
		fmt.Fprintf(os.Stderr, "glint: close inner vet config: %v\n", err)
		return 1
	}
	defer removeTemporaryFile(innerConfigPath)

	exitCode := runInnerUnitchecker(configIndex, innerConfigPath)
	raw, err := os.ReadFile(rawOutputPath)
	if err != nil && !os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "glint: read analyzer output: %v\n", err)
		return 1
	}
	if len(raw) > 0 {
		moduleRoot := mainModuleRoot(&config)
		if moduleRoot != "" {
			if boolFlagEnabled(os.Args[1:], "json") {
				raw, err = normalizeCachedOutput(raw, config.ModulePath, moduleRoot)
				if err != nil {
					fmt.Fprintf(os.Stderr, "glint: normalize analyzer output: %v\n", err)
					return 1
				}
			} else {
				raw = normalizeCachedText(raw, config.ModulePath, moduleRoot)
			}
		}
	}
	if err := os.WriteFile(originalStdout, raw, 0o666); err != nil {
		fmt.Fprintf(os.Stderr, "glint: write normalized analyzer output: %v\n", err)
		return 1
	}
	return exitCode
}

func boolFlagEnabled(args []string, name string) bool {
	for _, arg := range args {
		if arg == "-"+name {
			return true
		}
		if strings.HasPrefix(arg, "-"+name+"=") {
			value := strings.TrimPrefix(arg, "-"+name+"=")
			return value == "true"
		}
	}
	return false
}

func removeTemporaryFile(path string) {
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "glint: remove temporary file %s: %v\n", path, err)
	}
}

func runInnerUnitchecker(configIndex int, configFile string) int {
	args := append([]string(nil), os.Args[1:]...)
	args[configIndex] = configFile
	command := exec.Command(os.Args[0], args...)
	command.Env = appendWithout(command.Environ(), unitcheckerInnerEnv)
	command.Env = append(command.Env, unitcheckerInnerEnv+"=1")
	command.Stdin = os.Stdin
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	if err := command.Run(); err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			return exitError.ExitCode()
		}
		fmt.Fprintf(os.Stderr, "glint: run unitchecker: %v\n", err)
		return 1
	}
	return 0
}

func appendWithout(environment []string, name string) []string {
	prefix := name + "="
	result := environment[:0]
	for _, item := range environment {
		if !strings.HasPrefix(item, prefix) {
			result = append(result, item)
		}
	}
	return result
}

func mainModuleRoot(config *unitchecker.Config) string {
	if config.ModulePath == "" || config.ModuleVersion != "" || config.Dir == "" {
		return ""
	}
	if config.ImportPath == config.ModulePath {
		return filepath.Clean(config.Dir)
	}
	prefix := config.ModulePath + "/"
	if !strings.HasPrefix(config.ImportPath, prefix) {
		return ""
	}
	relativeImport := strings.TrimPrefix(config.ImportPath, prefix)
	root := filepath.Clean(config.Dir)
	for range strings.SplitSeq(relativeImport, "/") {
		root = filepath.Dir(root)
	}
	return root
}

func moduleMarker(modulePath string) string {
	encoded := base64.RawURLEncoding.EncodeToString([]byte(modulePath))
	return normalizedModulePrefix + encoded + "/"
}
