// SPDX-FileCopyrightText: 2023 Christoph Mewes
// SPDX-License-Identifier: MIT

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

func TestPruneObject(t *testing.T) {
	testcases := []struct {
		input    string
		paths    []string
		expected string
	}{
		{
			input:    `{"foo":"bar"}`,
			paths:    []string{`foo`},
			expected: `{"foo":"bar"}`,
		},
		{
			input:    `{"foo":"bar","bar":"exlude"}`,
			paths:    []string{`foo`},
			expected: `{"foo":"bar"}`,
		},
		{
			input:    `{"foo":"bar","bar":"exlude"}`,
			paths:    []string{`foo.muh`},
			expected: `{"foo":"bar"}`,
		},
		{
			input:    `{"metadata":{"name":"name","namespace":"ns"}}`,
			paths:    []string{`metadata`},
			expected: `{"metadata":{"name":"name","namespace":"ns"}}`,
		},
		{
			input:    `{"metadata":{"name":"name","namespace":"ns"}}`,
			paths:    []string{`metadata.name`},
			expected: `{"metadata":{"name":"name"}}`,
		},
		{
			input:    `{"metadata":{"name":"name","namespace":"ns"}}`,
			paths:    []string{`metadata.name`, `metadata`},
			expected: `{"metadata":{"name":"name","namespace":"ns"}}`,
		},
		{
			input:    `{"metadata":{"name":"name","namespace":"ns"}}`,
			paths:    []string{`metadata.name`, `metadata.namespace`},
			expected: `{"metadata":{"name":"name","namespace":"ns"}}`,
		},
		{
			input:    `{"metadata":{"name":"name","namespace":"ns"},"spec":{"labels":["myvalue"],"replicas":1}}`,
			paths:    []string{`metadata.name`, `spec`},
			expected: `{"metadata":{"name":"name"},"spec":{"labels":["myvalue"],"replicas":1}}`,
		},
	}

	for _, testcase := range testcases {
		t.Run(testcase.input, func(t *testing.T) {
			var input map[string]interface{}
			if err := json.Unmarshal([]byte(testcase.input), &input); err != nil {
				t.Fatalf("invalid testcase: %v", err)
			}

			paths := []Path{}
			for _, path := range testcase.paths {
				p, err := ParsePath(path)
				if err != nil {
					t.Fatalf("invalid path: %v", err)
				}

				paths = append(paths, p)
			}

			output, err := PruneObject(input, paths)
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
