// Package mock_test demonstrates how to integrate ya's merge and assert
// packages directly into Go tests. This lets you validate YAML values files
// programmatically as part of your standard test suite.
package mock_test

import (
	"os"
	"testing"

	"github.com/ceejaykelly/yaml-assertions/pkg/assert"
	"github.com/ceejaykelly/yaml-assertions/pkg/merge"
)

// assertion groups a path, operator, and expected value for a single check.
type assertion struct {
	path     string
	operator string
	expected any
}

func TestMergedValuesAssertions(t *testing.T) {
	// Load values files from disk.
	base, err := os.ReadFile("testdata/base.yaml")
	if err != nil {
		t.Fatalf("failed to read base.yaml: %v", err)
	}
	override, err := os.ReadFile("testdata/override.yaml")
	if err != nil {
		t.Fatalf("failed to read override.yaml: %v", err)
	}

	// Merge in order: base is overridden by override.
	// Additional files can be appended to the variadic argument.
	merged, err := merge.Values(base, override)
	if err != nil {
		t.Fatalf("failed to merge values: %v", err)
	}

	// Define assertions against the merged result.
	// Operators: == (equal), != (not equal), contains, exists
	assertions := []assertion{
		// Override takes precedence over base for replicaCount.
		{path: "replicaCount", operator: "==", expected: 3},
		// Image tag is overridden by the override file.
		{path: "image.tag", operator: "==", expected: "v1.2.3"},
		// pullPolicy comes from the base (not overridden).
		{path: "image.pullPolicy", operator: "==", expected: "IfNotPresent"},
		// Secret password is overridden.
		{path: "secret.password", operator: "==", expected: "prod-secret"},
		// app.name is present (existence check).
		{path: "app.name", operator: "exists", expected: nil},
	}

	for _, a := range assertions {
		t.Run(a.path, func(t *testing.T) {
			passed, err := assert.Evaluate(merged, a.path, a.operator, a.expected)
			if err != nil {
				t.Errorf("assertion error: %v", err)
			} else if !passed {
				t.Errorf("assertion failed: %s %s %v", a.path, a.operator, a.expected)
			}
		})
	}
}
