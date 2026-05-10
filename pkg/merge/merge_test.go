package merge

import (
	"fmt"
	"testing"

	"sigs.k8s.io/yaml"
)

func TestValues(t *testing.T) {
	tests := []struct {
		name      string
		base      string
		overrides [][]byte
		expected  string
	}{
		{
			name:      "simple override",
			base:      "foo: bar\nbaz: qux\n",
			overrides: [][]byte{[]byte("foo: newbar\n")},
			expected:  "baz: qux\nfoo: newbar\n",
		},
		{
			name: "nested multiple override",
			base: "parent:\n  child: value\n",
			overrides: [][]byte{
				[]byte("parent:\n  child: newvalue\n"),
				[]byte("parent:\n  child: newervalue\n"),
				[]byte("parent:\n  child: evennewervalue\n"),
			},
			expected: "parent:\n  child: evennewervalue\n",
		},
		{
			name:      "add new key",
			base:      "foo: bar\n",
			overrides: [][]byte{[]byte("baz: qux\n")},
			expected:  "baz: qux\nfoo: bar\n",
		},
		{
			name:      "failed unmarshal",
			base:      "foo: bar\n",
			overrides: [][]byte{[]byte("invalid: [unclosed\n")},
			expected:  "error converting YAML to JSON: yaml: line 1: did not find expected ',' or ']'",
		},
		{
			name:      "null value detected",
			base:      "foo: bar\n",
			overrides: [][]byte{[]byte("test:\n")},
			expected:  "null value at path: test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Values(append([][]byte{[]byte(tt.base)}, tt.overrides...)...)
			if err != nil {
				if err.Error() != tt.expected {
					t.Errorf("Expected error:\n%s\nGot error:\n%s", tt.expected, err.Error())
				}
				return
			}
			// For YAML output, compare as maps to ignore key order
			var expectedMap, resultMap map[string]interface{}
			expectedErr := false
			if err := yaml.Unmarshal([]byte(tt.expected), &expectedMap); err != nil {
				expectedErr = true
			}
			if err := yaml.Unmarshal(result, &resultMap); err != nil {
				expectedErr = true
			}
			if !expectedErr {
				if !mapsEqual(expectedMap, resultMap) {
					t.Errorf("Expected (YAML as map):\n%v\nGot:\n%v", expectedMap, resultMap)
				}
			} else {
				// Fallback to string compare for error or non-YAML cases
				if string(result) != tt.expected {
					t.Errorf("Expected:\n%s\nGot:\n%s", tt.expected, string(result))
				}
			}
		})
	}
}

// mapsEqual compares two maps recursively for equality
func mapsEqual(a, b map[string]interface{}) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		bv, ok := b[k]
		if !ok {
			return false
		}
		switch va := v.(type) {
		case map[string]interface{}:
			vb, ok := bv.(map[string]interface{})
			if !ok || !mapsEqual(va, vb) {
				return false
			}
		default:
			if fmt.Sprintf("%v", va) != fmt.Sprintf("%v", bv) {
				return false
			}
		}
	}
	return true
}
