// SPDX-FileCopyrightText: 2023 Christoph Mewes
// SPDX-License-Identifier: MIT

package maputil

import (
	"errors"
	"fmt"
)

func RemovePath(obj map[string]interface{}, path Path) (map[string]interface{}, error) {
	if len(path) == 0 {
		return obj, errors.New("path cannot be empty")
	}

	head := path.Head()
	tail := path.Tail()

	if len(tail) == 0 {
		delete(obj, head)
		return obj, nil
	}

	childValue, exists := obj[head]
	if !exists {
		return obj, nil
	}

	childObj, ok := childValue.(map[string]interface{})
	if !ok {
		return obj, nil
	}

	modifiedChildObj, err := RemovePath(childObj, tail)
	if err != nil {
		return obj, errors.New("failed to remove child")
	}

	if len(modifiedChildObj) == 0 {
		delete(obj, head)
		return obj, nil
	}

	obj[head] = modifiedChildObj

	return obj, nil
}

func PruneObject(obj map[string]interface{}, paths []Path) (map[string]interface{}, error) {
	if len(paths) == 0 {
		return obj, errors.New("paths cannot be empty")
	}

	for i, path := range paths {
		if len(path) == 0 {
			return obj, fmt.Errorf("path %d is empty", i+1)
		}
	}

	for key, value := range obj {
		filteredValue, err := pruneValue(key, value, paths)
		if err != nil {
			return obj, err
		}

		if filteredValue == nil {
			delete(obj, key)
		} else {
			obj[key] = filteredValue
		}
	}

	return obj, nil
}

func pruneValue(key string, value interface{}, paths []Path) (interface{}, error) {
	// is this key in any of the paths?
	for _, path := range paths {
		head := path.Head()
		if key == head {
			// this path matches the current key

			// if the current value is not a map, we keep it;
			// e.g. paths = ["metadata.name.foo"] should keep metadata.name in the output
			valueMap, ok := value.(map[string]interface{})
			if !ok {
				return value, nil
			}

			// Take all paths, like ["metadata.name", "metadata.namespace", "status"]
			// and return the tails of those that have a matching head; if head is
			// "metadata", this returns ["name", "namespace"];
			// subPaths cannot be empty.
			subPaths := getSubPaths(paths, head)

			// if there is an empty tail in the subPaths, this means the entire
			// value should be included
			for _, subPath := range subPaths {
				if len(subPath) == 0 {
					return value, nil // final decision, keep this value in the object
				}
			}

			// strip down the object
			valueMap, err := PruneObject(valueMap, subPaths)
			if err != nil {
				return nil, err
			}

			// There is no point continuing walking through the outer for-loop, as
			// getSubPaths() included all relevant values already.
			return valueMap, nil
		}
	}

	// no path matched the value
	return nil, nil
}

func getSubPaths(paths []Path, head string) []Path {
	result := []Path{}

	for _, path := range paths {
		if path.Head() == head {
			result = append(result, path.Tail())
		}
	}

	return result
}
