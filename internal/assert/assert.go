package assert

import (
	"fmt"
	"reflect"
	"regexp"
	"strconv"

	"sigs.k8s.io/yaml"
)

type Operator string

const (
	Equal    Operator = "=="
	NotEqual Operator = "!="

	// Future operators to implement:
	// GreaterThan        Operator = ">"
	// LessThan           Operator = "<"
	// GreaterThanOrEqual Operator = ">="
	// LessThanOrEqual    Operator = "<="
)

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

// parsePath splits a path like foo.bar[0].baz into ["foo", "bar", "0", "baz"]
func parsePath(path string) []string {
	var result []string
	_ = regexp.MustCompile(`[^.\[\]]+|\[\d+\]`)
	i := 0
	for i < len(path) {
		if path[i] == '.' {
			i++
			continue
		}
		if path[i] == '[' {
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
		// normal key
		j := i
		for j < len(path) && path[j] != '.' && path[j] != '[' {
			j++
		}
		result = append(result, path[i:j])
		i = j
	}
	return result
}

func Evaluate(values []byte, object_path string, operator string, expected any) (bool, error) {
	var data map[string]interface{}
	if err := yaml.Unmarshal(values, &data); err != nil {
		return false, fmt.Errorf("failed to unmarshal values: %w", err)
	}

	parts := parsePath(object_path)
	var current any = data
	for _, part := range parts {
		// Try array index
		if idx, err := strconv.Atoi(part); err == nil {
			arr, ok := current.([]interface{})
			if !ok {
				return false, fmt.Errorf("expected array at %s", part)
			}
			if idx < 0 || idx >= len(arr) {
				return false, fmt.Errorf("invalid array index: %s", part)
			}
			current = arr[idx]
			continue
		}
		// Otherwise, treat as map key
		m, ok := current.(map[string]interface{})
		if !ok {
			return false, fmt.Errorf("expected map at %s", part)
		}
		var exists bool
		current, exists = m[part]
		if !exists {
			return false, fmt.Errorf("path not found: %s", part)
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
	// Add more operators as needed, with type assertions for numbers
	default:
		return false, fmt.Errorf("unsupported operator: %s", operator)
	}
}
