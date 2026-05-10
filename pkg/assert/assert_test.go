package assert

import (
	"testing"
)

type AssertTestCase struct {
	name            string
	values          []byte
	object_path     string
	expected_input  any
	operator        Operator
	expected_output bool
}

func executeAssertTestCases(t *testing.T, tests []AssertTestCase) {
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

func TestEqualAssert(t *testing.T) {
	tests := []AssertTestCase{
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

	executeAssertTestCases(t, tests)
}

func TestExistsAssert(t *testing.T) {
	tests := []AssertTestCase{
		{
			name:            "Test existing path with Exists operator",
			values:          []byte("foo:\n  bar: 123\nbaz: qux\n"),
			object_path:     "foo.bar",
			expected_input:  nil,
			operator:        Exists,
			expected_output: true,
		},
		{
			name:            "Test non-existent path with Exists operator",
			values:          []byte("foo:\n  bar: 123\nbaz: qux\n"),
			object_path:     "foo.nonexistent",
			expected_input:  nil,
			operator:        Exists,
			expected_output: false,
		},
	}

	executeAssertTestCases(t, tests)
}

func TestContainsAssert(t *testing.T) {
	tests := []AssertTestCase{
		{
			name:            "Test string contains with Contains operator",
			values:          []byte("foo: barbaz\n"),
			object_path:     "foo",
			expected_input:  "bar",
			operator:        Contains,
			expected_output: true,
		},
		{
			name:            "Test string does not contain with Contains operator",
			values:          []byte("foo: barbaz\n"),
			object_path:     "foo",
			expected_input:  "qux",
			operator:        Contains,
			expected_output: false,
		},
	}

	executeAssertTestCases(t, tests)
}

func TestArrayContainsAssert(t *testing.T) {
	tests := []AssertTestCase{
		{
			name:            "Test array contains with Contains operator",
			values:          []byte("foo: [\"bar\", \"baz\"]\n"),
			object_path:     "foo",
			expected_input:  "bar",
			operator:        Contains,
			expected_output: true,
		},
		{
			name:            "Test array contains map with Contains operator",
			values:          []byte("foo: [{\"key\": \"value\", \"anotherKey\": \"anotherValue\"}, {\"key\": \"another value\"}]\n"),
			object_path:     "foo",
			expected_input:  map[string]string{"key": "value"},
			operator:        Contains,
			expected_output: true,
		},
	}

	executeAssertTestCases(t, tests)
}
