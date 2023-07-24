// SPDX-FileCopyrightText: 2023 Christoph Mewes
// SPDX-License-Identifier: MIT

package diff

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"
	"time"

	"go.xrstf.de/stalk/pkg/maputil"

	"github.com/gookit/color"
	"github.com/shibukawa/cdiff"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/json"
	"sigs.k8s.io/yaml"
)

type Differ struct {
	opt *Options
	log logrus.FieldLogger
}

func NewDiffer(opt *Options, log logrus.FieldLogger) (*Differ, error) {
	if err := opt.Validate(); err != nil {
		return nil, fmt.Errorf("invalid options: %w", err)
	}

	if opt.DisableWordDiff {
		opt.CreateColorTheme = disableWordDiff(cloneColorTheme(opt.CreateColorTheme))
		opt.UpdateColorTheme = disableWordDiff(cloneColorTheme(opt.UpdateColorTheme))
		opt.DeleteColorTheme = disableWordDiff(cloneColorTheme(opt.DeleteColorTheme))
	}

	return &Differ{
		opt: opt,
		log: log,
	}, nil
}

func (d *Differ) PrintDiff(oldObj, newObj *unstructured.Unstructured, lastSeen time.Time) error {
	oldString, err := d.preprocess(oldObj)
	if err != nil {
		return fmt.Errorf("failed to process previous object: %w", err)
	}

	newString, err := d.preprocess(newObj)
	if err != nil {
		return fmt.Errorf("failed to process current object: %w", err)
	}

	// this can happen if the spec changes, but `--show metadata` was given by the user
	if oldString == newString && d.opt.HideEmptyDiffs {
		return nil
	}

	titleA := diffTitle(oldObj, lastSeen)
	titleB := diffTitle(newObj, time.Now())

	colorTheme := d.opt.UpdateColorTheme
	if oldObj == nil {
		colorTheme = d.opt.CreateColorTheme
	}
	if newObj == nil {
		colorTheme = d.opt.DeleteColorTheme
	}

	diff := cdiff.Diff(oldString, newString, cdiff.WordByWord)

	var buf bytes.Buffer
	color.Fprint(&buf, diff.UnifiedWithGooKitColor(titleA, titleB, d.opt.ContextLines, colorTheme))

	fmt.Println(fixBadSection(buf.String(), colorTheme))

	return nil
}

func (d *Differ) preprocess(obj *unstructured.Unstructured) (string, error) {
	if obj == nil {
		return "", nil
	}

	generic, err := json.Marshal(obj)
	if err != nil {
		return "", fmt.Errorf("failed to encode object as JSON: %w", err)
	}

	var genericObj map[string]interface{}
	if err := json.Unmarshal(generic, &genericObj); err != nil {
		return "", fmt.Errorf("failed to re-decode object from JSON: %w", err)
	}

	if d.opt.compiledJSONPath != nil {
		results, err := d.opt.compiledJSONPath.FindResults(genericObj)
		if err != nil {
			d.log.Warnf("Failed to apply JSON path: %w", err)
		} else if len(results) > 0 && len(results[0]) > 0 {
			generic, err = json.Marshal(results[0][0].Interface())
			if err != nil {
				return "", fmt.Errorf("failed to encode JSON path result as JSON: %w", err)
			}

			// reset the map
			genericObj = map[string]interface{}{}

			if err := json.Unmarshal(generic, &genericObj); err != nil {
				// the JSONPath might have resulted in a scalar value
				var testValue interface{}
				if innerErr := json.Unmarshal(generic, &testValue); innerErr != nil {
					return "", fmt.Errorf("failed to re-decode JSON path result from JSON: %w", err)
				}

				final, err := yaml.JSONToYAML(generic)
				if err != nil {
					return "", fmt.Errorf("failed to encode object as YAML: %w", err)
				}

				return string(final), nil
			}
		}
	}

	if len(d.opt.parsedIncludePaths) > 0 {
		genericObj, err = maputil.PruneObject(genericObj, d.opt.parsedIncludePaths)
		if err != nil {
			return "", fmt.Errorf("failed to apply include inpressions: %w", err)
		}

		generic, err = json.Marshal(genericObj)
		if err != nil {
			return "", fmt.Errorf("failed to encode include inpression result as JSON: %w", err)
		}
	}

	if len(d.opt.parsedExcludePaths) > 0 {
		for _, excludePath := range d.opt.parsedExcludePaths {
			genericObj, err = maputil.RemovePath(genericObj, excludePath)
			if err != nil {
				return "", fmt.Errorf("failed to apply exclude expression %v: %w", excludePath, err)
			}
		}

		generic, err = json.Marshal(genericObj)
		if err != nil {
			return "", fmt.Errorf("failed to encode exclude expression result as JSON: %w", err)
		}
	}

	final, err := yaml.JSONToYAML(generic)
	if err != nil {
		return "", fmt.Errorf("failed to encode object as YAML: %w", err)
	}

	return string(final), nil
}

func objectKey(obj *unstructured.Unstructured) string {
	key := obj.GetName()
	if ns := obj.GetNamespace(); ns != "" {
		key = fmt.Sprintf("%s/%s", ns, key)
	}

	return key
}

func diffTitle(obj *unstructured.Unstructured, lastSeen time.Time) string {
	if obj == nil {
		return "(none)"
	}

	timestamp := lastSeen.Format(time.RFC3339)
	kind := obj.GroupVersionKind().Kind

	return fmt.Sprintf("%s %s v%s (%s) (gen. %d)", kind, objectKey(obj), obj.GetResourceVersion(), timestamp, obj.GetGeneration())
}

// this ensures that the first line of a context/diff is not placed in the
// same line as the @@...@@ marker
func fixBadSection(output string, theme map[cdiff.Tag]color.Style) string {
	regex := theme[cdiff.OpenSection].Render("@@ _placeholder_ @@")
	regex = regexp.QuoteMeta(regex)
	regex = strings.Replace(regex, "_placeholder_", "[0-9-+, ]+", -1)
	expr := regexp.MustCompile(regex)

	return expr.ReplaceAllString(output, `$0`+"\n")
}
