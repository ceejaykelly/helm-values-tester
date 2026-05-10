// Command ya is a CLI tool for merging Helm values files and asserting on
// rendered YAML output. It provides two subcommands:
//
//	ya merge file1.yaml file2.yaml ...
//	ya assert [--assert path==value] [--assert-file asserts.yaml] [file.yaml ...]
package main

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/ceejaykelly/yaml-assertions/pkg/assert"
	"github.com/ceejaykelly/yaml-assertions/pkg/document"
	"github.com/ceejaykelly/yaml-assertions/pkg/logger"
	"github.com/ceejaykelly/yaml-assertions/pkg/merge"
	"sigs.k8s.io/yaml"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: ya <merge|assert> [args...]")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "merge":
		runMerge(os.Args[2:])
	case "assert":
		os.Exit(runAssert(os.Args[2:]))
	default:
		fmt.Fprintf(os.Stderr, "Unknown subcommand %q. Use 'merge' or 'assert'.\n", os.Args[1])
		os.Exit(1)
	}
}

// runMerge handles "ya merge file1.yaml file2.yaml ..."
// It merges the files in order using Helm-aware coalescing and prints the result.
func runMerge(args []string) {
	if len(args) == 0 {
		logger.Fatal("Usage: ya merge <file1.yaml> [file2.yaml] ...")
	}

	var payloads [][]byte
	for _, path := range args {
		data, err := os.ReadFile(path)
		if err != nil {
			logger.Fatal("failed to read %s: %v", path, err)
		}
		payloads = append(payloads, data)
	}

	merged, err := merge.Values(payloads...)
	if err != nil {
		logger.Fatal("merge failed: %v", err)
	}
	fmt.Printf("%s\n", string(merged))
}

// runAssert handles "ya assert [flags] [files...]"
// It returns 0 if all assertions pass, 1 if any fail or an error occurs.
//
// Flags:
//
//	--assert path<op>value   inline assertion (may be repeated)
//	--assert-file path.yaml  YAML map of named AssertSpec objects (may be repeated; later files override earlier ones)
//
// Input files (or stdin if omitted or "-") are parsed as multi-document YAML.
// Each assertion is evaluated against the matching Kubernetes resource.
func runAssert(args []string) int {
	var filePaths []string
	// assertMap holds named assertions. Using a map allows multiple --assert-file
	// inputs to be merged: a key in a later file overrides the same key in an
	// earlier file, enabling layered assertion suites (e.g. base + environment-specific).
	assertMap := map[string]assert.AssertSpec{}
	// inlineOrder tracks the insertion order of --assert flags so they run first
	// and in the order they were specified.
	var inlineKeys []string

	i := 0
	for i < len(args) {
		arg := args[i]
		switch {
		case arg == "--assert" && i+1 < len(args):
			spec, err := parseInlineAssert(args[i+1])
			if err != nil {
				logger.Error("invalid --assert %q: %v", args[i+1], err)
				return 1
			}
			// Use the raw expression as the name for inline assertions.
			key := args[i+1]
			assertMap[key] = spec
			inlineKeys = append(inlineKeys, key)
			i += 2

		case arg == "--assert-file" && i+1 < len(args):
			data, err := os.ReadFile(args[i+1])
			if err != nil {
				logger.Error("failed to read assert file %q: %v", args[i+1], err)
				return 1
			}
			// Assert files are maps of name → AssertSpec. Merging multiple files
			// is as simple as overlaying one map on top of another.
			var fileSpecs map[string]assert.AssertSpec
			if err := yaml.Unmarshal(data, &fileSpecs); err != nil {
				logger.Error("failed to parse assert file %q: %v", args[i+1], err)
				return 1
			}
			for k, v := range fileSpecs {
				assertMap[k] = v
			}
			i += 2

		case strings.HasPrefix(arg, "--"):
			logger.Error("unknown flag: %s", arg)
			return 1

		default:
			filePaths = append(filePaths, arg)
			i++
		}
	}

	// Read YAML input from files or stdin.
	var rawInput []byte
	if len(filePaths) == 0 || (len(filePaths) == 1 && filePaths[0] == "-") {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			logger.Error("failed to read from stdin: %v", err)
			return 1
		}
		rawInput = data
	} else {
		var parts [][]byte
		for _, path := range filePaths {
			data, err := os.ReadFile(path)
			if err != nil {
				logger.Error("failed to read %s: %v", path, err)
				return 1
			}
			parts = append(parts, data)
		}
		// Join multiple files with a document separator so Split handles them uniformly.
		rawInput = []byte(strings.Join(func() []string {
			ss := make([]string, len(parts))
			for i, p := range parts {
				ss[i] = string(p)
			}
			return ss
		}(), "\n---\n"))
	}

	// Parse the multi-document YAML stream into individual resource maps.
	docs, err := document.Split(rawInput)
	if err != nil {
		logger.Error("failed to parse YAML input: %v", err)
		return 1
	}

	if len(assertMap) == 0 {
		// No assertions — just pretty-print what we parsed and exit 0.
		for _, doc := range docs {
			out, _ := yaml.Marshal(doc)
			fmt.Printf("---\n%s\n", string(out))
		}
		return 0
	}

	// Build an ordered list of assertion names. Inline --assert flags come first
	// (in the order specified), then file-based assertions in sorted order.
	fileKeys := make([]string, 0, len(assertMap))
	inlineSet := make(map[string]bool, len(inlineKeys))
	for _, k := range inlineKeys {
		inlineSet[k] = true
	}
	for k := range assertMap {
		if !inlineSet[k] {
			fileKeys = append(fileKeys, k)
		}
	}
	sort.Strings(fileKeys)
	orderedKeys := append(inlineKeys, fileKeys...)

	// Evaluate each assertion. Track overall pass/fail.
	exitCode := 0
	for _, name := range orderedKeys {
		a := assertMap[name]
		// Find resources that match the assertion's kind/name selector.
		matched := document.Match(docs, a.Kind, a.Name)
		if len(matched) == 0 {
			logger.Fail("%s [no match] (no resource matched selector kind=%q name=%q)",
				name, a.Kind, a.Name)
			exitCode = 1
			continue
		}

		for _, doc := range matched {
			resource := fmt.Sprintf("[%s/%s]", document.Kind(doc), document.Name(doc))

			// Marshal the document back to YAML bytes so Evaluate can unmarshal it.
			docBytes, err := yaml.Marshal(doc)
			if err != nil {
				logger.Error("%s %s failed to re-marshal document: %v", name, resource, err)
				exitCode = 1
				continue
			}

			pass, err := assert.Evaluate(docBytes, a.Path, string(a.Op), a.Value)
			if err != nil {
				logger.Fail("%s %s %s %s %v (error: %v)", name, resource, a.Path, a.Op, a.Value, err)
				exitCode = 1
			} else if pass {
				logger.Pass("%s %s %s %s %v", name, resource, a.Path, a.Op, a.Value)
			} else {
				logger.Fail("%s %s %s %s %v", name, resource, a.Path, a.Op, a.Value)
				exitCode = 1
			}
		}
	}

	return exitCode
}

// parseInlineAssert parses a single inline assertion of the form:
//
//	path<op>value
//
// where op is one of: ==, !=, >=, <=, >, <, contains, exists
// The value is parsed as an integer or float if possible, otherwise kept as a string.
func parseInlineAssert(arg string) (assert.AssertSpec, error) {
	// Check longer multi-character operators first to avoid partial matches.
	ops := []string{"contains", "exists", "==", "!=", ">=", "<=", ">", "<"}
	for _, op := range ops {
		idx := strings.Index(arg, op)
		if idx == -1 {
			continue
		}
		path := arg[:idx]
		rawVal := arg[idx+len(op):]

		// "exists" takes no value; ignore whatever follows.
		if op == "exists" {
			return assert.AssertSpec{Path: path, Op: assert.Exists}, nil
		}

		var expected interface{} = rawVal
		if v, err := parseValue(rawVal); err == nil {
			expected = v
		}
		return assert.AssertSpec{Path: path, Op: assert.Operator(op), Value: expected}, nil
	}
	return assert.AssertSpec{}, fmt.Errorf("no valid operator found in %q", arg)
}

// parseValue attempts to parse s as an integer, then a float64.
// Returns the raw string if neither succeeds.
func parseValue(s string) (interface{}, error) {
	if n, err := strconv.Atoi(s); err == nil {
		return n, nil
	}
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return f, nil
	}
	return s, fmt.Errorf("not a number")
}
