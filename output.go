package glint

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"golang.org/x/mod/modfile"
)

func normalizeCachedOutput(data []byte, modulePath, moduleRoot string) ([]byte, error) {
	root := filepath.Clean(moduleRoot)
	marker := moduleMarker(modulePath)
	return rewriteJSONStream(data, func(key, value string) string {
		if key == "new" {
			return value
		}
		switch key {
		case "posn", "end":
			filename, suffix := splitPosition(value)
			return normalizeFilename(filename, root, marker) + suffix
		case "filename":
			return normalizeFilename(value, root, marker)
		case "error":
			return normalizeEmbeddedPath(value, root, marker)
		default:
			return value
		}
	})
}

func normalizeFilename(filename, moduleRoot, marker string) string {
	if filename == "" || !filepath.IsAbs(filename) {
		return filename
	}
	relative, err := filepath.Rel(moduleRoot, filepath.Clean(filename))
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return filename
	}
	return marker + filepath.ToSlash(relative)
}

func normalizeEmbeddedPath(value, moduleRoot, marker string) string {
	nativeRoot := filepath.Clean(moduleRoot) + string(filepath.Separator)
	value = strings.ReplaceAll(value, nativeRoot, marker)
	if slashRoot := filepath.ToSlash(filepath.Clean(moduleRoot)) + "/"; slashRoot != nativeRoot {
		value = strings.ReplaceAll(value, slashRoot, marker)
	}
	return value
}

func normalizeCachedText(data []byte, modulePath, moduleRoot string) []byte {
	marker := moduleMarker(modulePath)
	nativeRoot := filepath.Clean(moduleRoot) + string(filepath.Separator)
	data = bytes.ReplaceAll(data, []byte(nativeRoot), []byte(marker))
	if slashRoot := filepath.ToSlash(filepath.Clean(moduleRoot)) + "/"; slashRoot != nativeRoot {
		data = bytes.ReplaceAll(data, []byte(slashRoot), []byte(marker))
	}
	return data
}

func rebaseCachedText(data []byte) ([]byte, error) {
	if !bytes.Contains(data, []byte(normalizedModulePrefix)) {
		return data, nil
	}
	directories, err := moduleDirectories()
	if err != nil {
		return nil, err
	}
	return []byte(rebaseMarkers(string(data), directories)), nil
}

func rebaseCachedOutput(data []byte) ([]byte, error) {
	if !bytes.Contains(data, []byte(normalizedModulePrefix)) {
		return data, nil
	}
	directories, err := moduleDirectories()
	if err != nil {
		return nil, err
	}
	return rewriteJSONStream(data, func(key, value string) string {
		if key == "new" {
			return value
		}
		return rebaseMarkers(value, directories)
	})
}

func moduleDirectories() (map[string]string, error) {
	command := exec.Command("go", "env", "-json", "GOMOD", "GOWORK")
	output, err := command.Output()
	if err != nil {
		return nil, fmt.Errorf("locate module files: %w", err)
	}

	var environment struct {
		GoMod  string `json:"GOMOD"`
		GoWork string `json:"GOWORK"`
	}
	if err := json.Unmarshal(output, &environment); err != nil {
		return nil, fmt.Errorf("decode Go environment: %w", err)
	}

	directories := make(map[string]string)
	addModule := func(goModPath string) error {
		if goModPath == "" || goModPath == os.DevNull {
			return nil
		}
		data, err := os.ReadFile(goModPath)
		if err != nil {
			return err
		}
		file, err := modfile.Parse(goModPath, data, nil)
		if err != nil {
			return err
		}
		if file.Module != nil {
			directories[file.Module.Mod.Path] = filepath.Dir(goModPath)
		}
		return nil
	}
	if err := addModule(environment.GoMod); err != nil {
		return nil, fmt.Errorf("read main module: %w", err)
	}

	if environment.GoWork != "" && environment.GoWork != "off" {
		data, err := os.ReadFile(environment.GoWork)
		if err != nil {
			return nil, fmt.Errorf("read workspace: %w", err)
		}
		work, err := modfile.ParseWork(environment.GoWork, data, nil)
		if err != nil {
			return nil, fmt.Errorf("parse workspace: %w", err)
		}
		workDirectory := filepath.Dir(environment.GoWork)
		for _, use := range work.Use {
			moduleDirectory := use.Path
			if !filepath.IsAbs(moduleDirectory) {
				moduleDirectory = filepath.Join(workDirectory, moduleDirectory)
			}
			if err := addModule(filepath.Join(moduleDirectory, "go.mod")); err != nil {
				return nil, fmt.Errorf("read workspace module %s: %w", use.Path, err)
			}
		}
	}
	return directories, nil
}

func cacheWorkingDirectoryIdentity() (string, error) {
	workingDirectory, err := os.Getwd()
	if err != nil {
		return "", err
	}
	directories, err := moduleDirectories()
	if err != nil {
		return "", err
	}

	var bestDirectory string
	var identity string
	for modulePath, moduleDirectory := range directories {
		relative, err := filepath.Rel(moduleDirectory, workingDirectory)
		if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
			continue
		}
		if len(moduleDirectory) <= len(bestDirectory) {
			continue
		}
		bestDirectory = moduleDirectory
		identity = modulePath + ":" + filepath.ToSlash(relative)
	}
	if identity != "" {
		return identity, nil
	}

	// Outside a module there is no location-independent package identity to
	// use, so prefer correctness over cross-checkout reuse.
	return filepath.Clean(workingDirectory), nil
}

func rebaseMarkers(value string, directories map[string]string) string {
	for {
		start := strings.Index(value, normalizedModulePrefix)
		if start < 0 {
			return value
		}
		encodedStart := start + len(normalizedModulePrefix)
		slash := strings.IndexByte(value[encodedStart:], '/')
		if slash < 0 {
			return value
		}
		slash += encodedStart
		modulePathBytes, err := base64.RawURLEncoding.DecodeString(value[encodedStart:slash])
		if err != nil {
			return value
		}
		moduleDirectory, ok := directories[string(modulePathBytes)]
		if !ok {
			return value
		}

		end := slash + 1
		for end < len(value) && !isPathTerminator(value[end]) {
			end++
		}
		relative := value[slash+1 : end]
		rebased := filepath.Join(moduleDirectory, filepath.FromSlash(relative))
		value = value[:start] + rebased + value[end:]
	}
}

func isPathTerminator(character byte) bool {
	switch character {
	case ':', ' ', '\t', '\n', '\r', '"', '\'', ')', ']', '}':
		return true
	default:
		return false
	}
}

func splitPosition(value string) (filename, suffix string) {
	lastColon := strings.LastIndexByte(value, ':')
	if lastColon < 0 {
		return value, ""
	}
	if _, err := strconv.Atoi(value[lastColon+1:]); err != nil {
		return value, ""
	}
	secondColon := strings.LastIndexByte(value[:lastColon], ':')
	if secondColon < 0 {
		return value, ""
	}
	if _, err := strconv.Atoi(value[secondColon+1 : lastColon]); err != nil {
		return value, ""
	}
	return value[:secondColon], value[secondColon:]
}

func rewriteJSONStream(data []byte, rewrite func(key, value string) string) ([]byte, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	var output bytes.Buffer
	encoder := json.NewEncoder(&output)
	encoder.SetIndent("", "\t")
	for {
		var value any
		if err := decoder.Decode(&value); err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		rewriteJSONValue(value, "", rewrite)
		if err := encoder.Encode(value); err != nil {
			return nil, err
		}
	}
	return output.Bytes(), nil
}

func rewriteJSONValue(value any, key string, rewrite func(key, value string) string) {
	switch value := value.(type) {
	case map[string]any:
		for childKey, child := range value {
			if text, ok := child.(string); ok {
				value[childKey] = rewrite(childKey, text)
				continue
			}
			rewriteJSONValue(child, childKey, rewrite)
		}
	case []any:
		for _, child := range value {
			rewriteJSONValue(child, key, rewrite)
		}
	}
}

type diagnostic struct {
	Category string `json:"category,omitempty"`
	Posn     string `json:"posn"`
	End      string `json:"end"`
	Message  string `json:"message"`
	Related  []struct {
		Posn    string `json:"posn"`
		Message string `json:"message"`
	} `json:"related,omitempty"`
}

type jsonTree map[string]map[string]json.RawMessage

func decodeJSONTrees(data []byte) (jsonTree, error) {
	merged := make(jsonTree)
	decoder := json.NewDecoder(bytes.NewReader(data))
	for {
		var tree jsonTree
		if err := decoder.Decode(&tree); err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		for packageID, analyses := range tree {
			if merged[packageID] == nil {
				merged[packageID] = make(map[string]json.RawMessage)
			}
			for analyzer, result := range analyses {
				merged[packageID][analyzer] = result
			}
		}
	}
	return merged, nil
}

func mergeAndFormatJSON(data []byte) ([]byte, error) {
	tree, err := decodeJSONTrees(data)
	if err != nil {
		return nil, err
	}
	formatted, err := json.MarshalIndent(tree, "", "\t")
	if err != nil {
		return nil, err
	}
	return append(formatted, '\n'), nil
}

func printTextOutput(writer io.Writer, data []byte) (diagnostics, analysisErrors int, _ error) {
	tree, err := decodeJSONTrees(data)
	if err != nil {
		return 0, 0, err
	}

	packages := sortedKeys(tree)
	for _, packageID := range packages {
		analyses := tree[packageID]
		analyzers := sortedKeys(analyses)
		for _, analyzer := range analyzers {
			result := analyses[analyzer]
			var failure struct {
				Error string `json:"error"`
			}
			if err := json.Unmarshal(result, &failure); err == nil && failure.Error != "" {
				_, _ = fmt.Fprintf(writer, "%s: %s: %s\n", packageID, analyzer, failure.Error)
				analysisErrors++
				continue
			}

			var findings []diagnostic
			if err := json.Unmarshal(result, &findings); err != nil {
				return diagnostics, analysisErrors, fmt.Errorf("decode %s/%s: %w", packageID, analyzer, err)
			}
			sort.SliceStable(findings, func(i, j int) bool {
				if findings[i].Posn != findings[j].Posn {
					return findings[i].Posn < findings[j].Posn
				}
				return findings[i].Message < findings[j].Message
			})
			for _, finding := range findings {
				_, _ = fmt.Fprintf(writer, "%s: %s\n", finding.Posn, finding.Message)
				for _, related := range finding.Related {
					_, _ = fmt.Fprintf(writer, "%s: \t%s\n", related.Posn, related.Message)
				}
				diagnostics++
			}
		}
	}
	return diagnostics, analysisErrors, nil
}

func sortedKeys[V any](values map[string]V) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
