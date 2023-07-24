// SPDX-FileCopyrightText: 2023 Christoph Mewes
// SPDX-License-Identifier: MIT

package diff

import (
	"errors"
	"fmt"

	"go.xrstf.de/stalk/pkg/maputil"

	"github.com/gookit/color"
	"github.com/shibukawa/cdiff"
	"k8s.io/client-go/util/jsonpath"
)

type Options struct {
	ContextLines    int
	HideEmptyDiffs  bool
	DisableWordDiff bool

	JSONPath         string
	compiledJSONPath *jsonpath.JSONPath

	IncludePaths       []string
	parsedIncludePaths []maputil.Path

	ExcludePaths       []string
	parsedExcludePaths []maputil.Path

	CreateColorTheme map[cdiff.Tag]color.Style
	UpdateColorTheme map[cdiff.Tag]color.Style
	DeleteColorTheme map[cdiff.Tag]color.Style
}

func (o *Options) Validate() error {
	if o.ContextLines < 0 {
		return errors.New("context lines cannot be negative")
	}

	if o.JSONPath != "" {
		path := jsonpath.New("mypath")
		if err := path.Parse(o.JSONPath); err != nil {
			return fmt.Errorf("invalid JSON path: %w", err)
		}

		path.EnableJSONOutput(true)
		path.AllowMissingKeys(true)

		o.compiledJSONPath = path
	}

	if len(o.IncludePaths) > 0 {
		o.parsedIncludePaths = []maputil.Path{}

		for _, path := range o.IncludePaths {
			parsed, err := maputil.ParsePath(path)
			if err != nil {
				return fmt.Errorf("invalid include expression %q: %w", path, err)
			}

			o.parsedIncludePaths = append(o.parsedIncludePaths, parsed)
		}
	}

	if len(o.ExcludePaths) > 0 {
		o.parsedExcludePaths = []maputil.Path{}

		for _, path := range o.ExcludePaths {
			parsed, err := maputil.ParsePath(path)
			if err != nil {
				return fmt.Errorf("invalid exclude expression %q: %w", path, err)
			}

			o.parsedExcludePaths = append(o.parsedExcludePaths, parsed)
		}
	}

	return nil
}
