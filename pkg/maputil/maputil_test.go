package maputil

import (
	"fmt"
	"testing"

	"k8s.io/apimachinery/pkg/util/json"
)

func TestRemovePath(t *testing.T) {
	testcases := []struct {
		input    string
		path     string
		expected string
	}{
		{
			input:    `{"foo":"bar"}`,
			path:     `foo`,
			expected: `{}`,
		},
		{
			input:    `{"foo":"bar","bar":"foo"}`,
			path:     `foo`,
			expected: `{"bar":"foo"}`,
		},
		{
			input:    `{"foo":12}`,
			path:     `foo`,
			expected: `{}`,
		},
		{
			input:    `{"foo":{"bar":12}}`,
			path:     `foo`,
			expected: `{}`,
		},
		{
			input:    `{"foo":{"bar":12}}`,
			path:     `foo.bar`,
			expected: `{}`, // specifically, we do not want `"foo":{}`!
		},
		{
			input:    `{"foo":{"bar":12,"extra":"yes"}}`,
			path:     `foo.bar`,
			expected: `{"foo":{"extra":"yes"}}`,
		},
		{
			input:    `{"foo":[1,2,3]}`,
			path:     `foo.bar`,
			expected: `{"foo":[1,2,3]}`,
		},
	}

	for _, testcase := range testcases {
		t.Run(fmt.Sprintf("%s against %s", testcase.path, testcase.input), func(t *testing.T) {
			var input map[string]interface{}
			if err := json.Unmarshal([]byte(testcase.input), &input); err != nil {
				t.Fatalf("invalid testcase: %v", err)
			}

			p, err := ParsePath(testcase.path)
			if err != nil {
				t.Fatalf("invalid path: %v", err)
			}

			output, err := RemovePath(input, p)
			if err != nil {
				t.Fatalf("failed to remove path: %v", err)
			}

			outputEncoded, _ := json.Marshal(output)

			if string(outputEncoded) != testcase.expected {
				t.Errorf("Expected %q, but got %q.", testcase.expected, string(outputEncoded))
			}
		})
	}
}
