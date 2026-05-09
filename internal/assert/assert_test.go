package assert

import (
	"testing"
)

func TestAssert(t *testing.T) {
	tests := []struct {
		name            string
		values          []byte
		object_path     string
		expected_input  any
		operator        Operator
		expected_output bool
	}{
		{
			name:            "Test happy path with Equal operator",
			values:          []byte("foo: bar\nbaz: qux\n"),
			object_path:     "foo",
			expected_input:  "bar",
			operator:        Equal,
			expected_output: true,
		},
		{
			name:            "Test mismatching value with Equal operator",
			values:          []byte("foo: bar\nbaz: qux\n"),
			object_path:     "foo",
			expected_input:  "different value",
			operator:        Equal,
			expected_output: false,
		},
		{
			name:            "Test mismatching type with Equal operator",
			values:          []byte("foo: bar\nbaz: qux\n"),
			object_path:     "foo",
			expected_input:  123,
			operator:        Equal,
			expected_output: false,
		},
		{
			name:            "Test nested value with Equal operator",
			values:          []byte("foo:\n  bar: 123\nbaz: qux\n"),
			object_path:     "foo.bar",
			expected_input:  123,
			operator:        Equal,
			expected_output: true,
		},
		{
			name:            "Test nested array value with Equal operator",
			values:          []byte("foo:\n  bar: [123, 456]\nbaz: qux\n"),
			object_path:     "foo.bar[0]",
			expected_input:  123,
			operator:        Equal,
			expected_output: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Evaluate(tt.values, tt.object_path, string(tt.operator), tt.expected_input)
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if result != tt.expected_output {
				t.Errorf("Assertion failed for test: %s", tt.name)
			}
		})
	}
}
