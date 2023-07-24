// SPDX-FileCopyrightText: 2023 Christoph Mewes
// SPDX-License-Identifier: MIT

package diff

import (
	"github.com/gookit/color"
	"github.com/shibukawa/cdiff"
)

var (
	CreateColorTheme map[cdiff.Tag]color.Style
	UpdateColorTheme map[cdiff.Tag]color.Style
	DeleteColorTheme map[cdiff.Tag]color.Style
)

func init() {
	UpdateColorTheme = cloneColorTheme(cdiff.GooKitColorTheme)
	UpdateColorTheme[cdiff.OpenHeader] = color.New(color.Yellow)

	CreateColorTheme = cloneColorTheme(UpdateColorTheme)
	CreateColorTheme[cdiff.OpenInsertedModified] = nil

	DeleteColorTheme = cloneColorTheme(UpdateColorTheme)
	DeleteColorTheme[cdiff.OpenDeletedModified] = nil
}

func cloneColorTheme(theme map[cdiff.Tag]color.Style) map[cdiff.Tag]color.Style {
	result := map[cdiff.Tag]color.Style{}

	for k, v := range theme {
		result[k] = v
	}

	return result
}

func disableWordDiff(theme map[cdiff.Tag]color.Style) map[cdiff.Tag]color.Style {
	theme[cdiff.OpenDeletedModified] = theme[cdiff.OpenDeletedNotModified]
	theme[cdiff.OpenInsertedModified] = theme[cdiff.OpenInsertedNotModified]

	return theme
}
