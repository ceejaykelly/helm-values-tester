package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/ceejaykelly/helm-values-tester/internal/assert"
	"github.com/ceejaykelly/helm-values-tester/internal/merge"
	"sigs.k8s.io/yaml"
)

type AssertSpec struct {
	Path     string      `yaml:"path"`
	Operator string      `yaml:"operator"`
	Expected interface{} `yaml:"expected"`
}

func parseInlineAssert(arg string) (AssertSpec, error) {
	// Format: path[.sub][[idx]]<op>value, e.g. foo.bar[0]==123
	ops := []string{"==", "!=", ">=", "<=", ">", "<"}
	for _, op := range ops {
		if idx := strings.Index(arg, op); idx != -1 {
			path := arg[:idx]
			val := arg[idx+len(op):]
			// Try to parse as int, float, or string
			var expected interface{} = val
			if i, err := parseNumber(val); err == nil {
				expected = i
			}
			return AssertSpec{Path: path, Operator: op, Expected: expected}, nil
		}
	}
	return AssertSpec{}, fmt.Errorf("invalid assert format: %s", arg)
}

func parseNumber(s string) (interface{}, error) {
	if i, err := fmt.Sscanf(s, "%d", new(int)); err == nil && i == 1 {
		var n int
		fmt.Sscanf(s, "%d", &n)
		return n, nil
	}
	if f, err := fmt.Sscanf(s, "%f", new(float64)); err == nil && f == 1 {
		var n float64
		fmt.Sscanf(s, "%f", &n)
		return n, nil
	}
	return s, fmt.Errorf("not a number")
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: hvt <file1.yaml> [file2.yaml] ... [fileN.yaml] [--assert key.path==value] [--assert-file asserts.yaml]")
		os.Exit(1)
	}

	var filePaths []string
	var asserts []AssertSpec

	// Parse CLI args
	i := 1
	for i < len(os.Args) {
		arg := os.Args[i]
		if arg == "--assert" && i+1 < len(os.Args) {
			spec, err := parseInlineAssert(os.Args[i+1])
			if err != nil {
				log.Fatalf("invalid --assert: %v", err)
			}
			asserts = append(asserts, spec)
			i += 2
			continue
		}
		if arg == "--assert-file" && i+1 < len(os.Args) {
			data, err := os.ReadFile(os.Args[i+1])
			if err != nil {
				log.Fatalf("failed to read assert file: %v", err)
			}
			var specs []AssertSpec
			if err := yaml.Unmarshal(data, &specs); err != nil {
				log.Fatalf("failed to parse assert file: %v", err)
			}
			asserts = append(asserts, specs...)
			i += 2
			continue
		}
		if strings.HasPrefix(arg, "--") {
			log.Fatalf("unknown flag: %s", arg)
		}
		filePaths = append(filePaths, arg)
		i++
	}

	var payloads [][]byte
	if len(filePaths) == 0 || (len(filePaths) == 1 && filePaths[0] == "-") {
		// Read from stdin
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			log.Fatalf("failed to read from stdin: %v", err)
		}
		payloads = append(payloads, data)
	} else {
		for _, path := range filePaths {
			data, err := os.ReadFile(path)
			if err != nil {
				log.Fatalf("failed to read %s: %v", path, err)
			}
			payloads = append(payloads, data)
		}
	}

	result, err := merge.Values(payloads...)
	if err != nil {
		log.Fatalf("merge failed: %v", err)
	}

	fmt.Printf("---\n%s\n", string(result))

	// Run asserts if any
	if len(asserts) > 0 {
		fmt.Println("\nAssertions:")
		for _, a := range asserts {
			pass, err := assert.Evaluate(result, a.Path, a.Operator, a.Expected)
			if err != nil {
				fmt.Printf("FAIL: %s %s %v (error: %v)\n", a.Path, a.Operator, a.Expected, err)
			} else if pass {
				fmt.Printf("PASS: %s %s %v\n", a.Path, a.Operator, a.Expected)
			} else {
				fmt.Printf("FAIL: %s %s %v\n", a.Path, a.Operator, a.Expected)
			}
		}
	}
}
