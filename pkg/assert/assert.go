// Package assert provides the core assertion evaluation logic for ya.
// It supports path-based lookups into YAML documents and comparisons using
// a set of operators (==, !=, contains, exists).
package assert

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"sigs.k8s.io/yaml"
)

// Operator is the type for assertion comparison operators.
type Operator string

const (
	Equal    Operator = "=="
	NotEqual Operator = "!="
	Contains Operator = "contains"
	Exists   Operator = "exists"

	// Future operators to implement:
	// GreaterThan        Operator = ">"
	// LessThan           Operator = "<"
	// GreaterThanOrEqual Operator = ">="
	// LessThanOrEqual    Operator = "<="
)

// AssertSpec describes a single assertion to run against a rendered YAML document.
// Kind and Name are used to select the target Kubernetes resource from a multi-doc
// YAML stream. Path, Op, and Value define what is asserted.
type AssertSpec struct {
	// Kind filters documents by their "kind" field (e.g. "Deployment").
	// Leave empty to match any kind.
	Kind string `yaml:"kind,omitempty" json:"kind,omitempty"`

	// Name filters documents by metadata.name. Leave empty to match any name.
	Name string `yaml:"name,omitempty" json:"name,omitempty"`

	// Path is a dot-and-bracket-notation path into the YAML document,
	// e.g. "spec.template.spec.containers[0].image".
	Path string `yaml:"path" json:"path"`

	// Op is the comparison operator (==, !=, contains, exists).
	// NOTE: sigs.k8s.io/yaml unmarshals via JSON, so the json tag must match
	// the YAML key "operator" — yaml tags alone are not sufficient.
	Op Operator `yaml:"operator" json:"operator"`

	// Value is the expected value for the assertion. Ignored when Op is "exists".
	Value any `yaml:"expected,omitempty" json:"expected,omitempty"`
}

// normalizeNumber converts common numeric types to float64 to allow
// consistent comparison regardless of whether the value came from YAML
// unmarshalling (always float64) or user-supplied code (may be int/uint/…).
func normalizeNumber(v any) (float64, bool) {
	switch n := v.(type) {
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	case float32:
		return float64(n), true
	case float64:
		return n, true
	case uint:
		return float64(n), true
	case uint64:
		return float64(n), true
	default:
		return 0, false
	}
}

// parsePath splits a dot-and-bracket-notation path into individual keys.
// Examples:
//
//	"foo.bar"        → ["foo", "bar"]
//	"foo[0].bar"     → ["foo", "0", "bar"]
//	"foo.bar[2].baz" → ["foo", "bar", "2", "baz"]
func parsePath(path string) []string {
	var result []string
	i := 0
	for i < len(path) {
		if path[i] == '.' {
			i++
			continue
		}
		if path[i] == '[' {
			// Bracket index: extract digits between [ and ]
			j := i + 1
			for j < len(path) && path[j] != ']' {
				j++
			}
			if j < len(path) && path[j] == ']' {
				result = append(result, path[i+1:j])
				i = j + 1
				continue
			}
		}
		// Normal map key: read until the next '.' or '['
		j := i
		for j < len(path) && path[j] != '.' && path[j] != '[' {
			j++
		}
		result = append(result, path[i:j])
		i = j
	}
	return result
}

// Evaluate walks the given YAML byte slice along object_path and compares the
// found value using the specified operator. It returns (true, nil) when the
// assertion passes, (false, nil) when it fails cleanly, and (false, err) when
// the path or values cannot be resolved.
func Evaluate(values []byte, object_path string, operator string, expected any) (bool, error) {
	var data map[string]interface{}
	if err := yaml.Unmarshal(values, &data); err != nil {
		return false, fmt.Errorf("failed to unmarshal values: %w", err)
	}

	parts := parsePath(object_path)
	var current any = data
	for i, part := range parts {
		// Try array index
		if idx, err := strconv.Atoi(part); err == nil {
			arr, ok := current.([]interface{})
			if !ok {
				if operator == "exists" {
					return false, nil
				}
				return false, fmt.Errorf("expected array at %s", part)
			}
			if idx < 0 || idx >= len(arr) {
				if operator == "exists" {
					return false, nil
				}
				return false, fmt.Errorf("invalid array index: %s", part)
			}
			current = arr[idx]
			continue
		}
		// Otherwise, treat as map key
		m, ok := current.(map[string]interface{})
		if !ok {
			if operator == "exists" {
				return false, nil
			}
			return false, fmt.Errorf("expected map at %s", part)
		}
		var exists bool
		current, exists = m[part]
		if !exists {
			if operator == "exists" {
				return false, nil
			}
			return false, fmt.Errorf("path not found: %s", part)
		}
		// If this is the last part and operator is exists, return true
		if operator == "exists" && i == len(parts)-1 {
			return true, nil
		}
	}

	// Normalize numbers for comparison
	if a, aok := normalizeNumber(current); aok {
		if b, bok := normalizeNumber(expected); bok {
			current = a
			expected = b
		}
	}

	// Compare using the operator
	switch operator {
	case "==":
		return reflect.DeepEqual(current, expected), nil
	case "!=":
		return !reflect.DeepEqual(current, expected), nil
	case "exists":
		// If we got here, the path exists
		return true, nil
	case "contains":
		// Support string contains and array contains
		switch v := current.(type) {
		case string:
			substr, ok := expected.(string)
			if !ok {
				return false, fmt.Errorf("expected value for contains must be a string when target is string")
			}
			return containsString(v, substr), nil
		case []interface{}:
			for _, item := range v {
				if reflect.DeepEqual(item, expected) {
					return true, nil
				}
			}
			return false, nil
		default:
			return false, fmt.Errorf("contains operator not supported for type %T", current)
		}
	default:
		return false, fmt.Errorf("unsupported operator: %s", operator)
	}
}

// containsString reports whether substr is contained in s.
// It delegates directly to strings.Contains, which uses an efficient algorithm.
func containsString(s, substr string) bool {
	return strings.Contains(s, substr)
}
