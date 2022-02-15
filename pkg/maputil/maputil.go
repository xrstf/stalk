package maputil

import (
	"errors"
	"fmt"
)

func RemovePath(obj map[string]interface{}, path Path) (map[string]interface{}, error) {
	if len(path) == 0 {
		return obj, errors.New("path cannot be empty")
	}

	fmt.Printf("%s against %s\n", path, toJSON(obj))

	head := path.Head()
	tail := path.Tail()

	fmt.Printf("head: %v\ntail: %v\n", head, tail)

	if len(tail) == 0 {
		delete(obj, head)
		return obj, nil
	}

	childValue, exists := obj[head]
	if !exists {
		fmt.Printf("head %v does not exist\n", head)
		return obj, nil
	}

	childObj, ok := childValue.(map[string]interface{})
	if !ok {
		fmt.Printf("head %v is not a map, but %#v\n", head, childValue)
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
