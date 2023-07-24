// SPDX-FileCopyrightText: 2023 Christoph Mewes
// SPDX-License-Identifier: MIT

package maputil

import (
	"errors"
	"strings"
)

type Path []string

func ParsePath(path string) (Path, error) {
	if path == "" {
		return nil, errors.New("path cannot be empty")
	}

	parts := strings.Split(path, ".")
	validParts := []string{}

	for _, part := range parts {
		if part == "" {
			continue
		}

		validParts = append(validParts, part)
	}

	if len(validParts) == 0 {
		return nil, errors.New("path does not contain a single path element")
	}

	return Path(validParts), nil
}

func (p Path) Head() string {
	if len(p) == 0 {
		return ""
	}

	return p[0]
}

func (p Path) Tail() Path {
	if len(p) == 0 {
		return nil
	}

	return p[1:]
}
