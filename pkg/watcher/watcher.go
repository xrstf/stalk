// SPDX-FileCopyrightText: 2023 Christoph Mewes
// SPDX-License-Identifier: MIT

package watcher

import (
	"context"
	"path/filepath"
	"strings"

	"go.xrstf.de/stalk/pkg/diff"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/watch"
)

type Watcher struct {
	printer       *diff.Printer
	namespaces    []string
	resourceNames []string
}

func NewWatcher(printer *diff.Printer, namespaces, resourceNames []string) *Watcher {
	return &Watcher{
		printer:       printer,
		namespaces:    namespaces,
		resourceNames: resourceNames,
	}
}

func (w *Watcher) Watch(ctx context.Context, wi watch.Interface) {
	for event := range wi.ResultChan() {
		obj, ok := event.Object.(*unstructured.Unstructured)
		if !ok {
			continue
		}

		if w.resourceNameMatches(obj) && w.resourceNamespaceMatches(obj) {
			w.printer.Print(obj, event.Type)
		}
	}
}

func (w *Watcher) resourceNameMatches(obj *unstructured.Unstructured) bool {
	// no names given, so all resources match
	if len(w.resourceNames) == 0 {
		return true
	}

	for _, wantedName := range w.resourceNames {
		if nameMatches(obj.GetName(), wantedName) {
			return true
		}
	}

	return false
}

func (w *Watcher) resourceNamespaceMatches(obj *unstructured.Unstructured) bool {
	// no namespaces given, so all resources match
	if len(w.namespaces) == 0 {
		return true
	}

	for _, wantedNamespace := range w.namespaces {
		if nameMatches(obj.GetNamespace(), wantedNamespace) {
			return true
		}
	}

	return false
}

func nameMatches(name string, pattern string) bool {
	if strings.Contains(pattern, "*") {
		matched, _ := filepath.Match(pattern, name)
		return matched
	}

	return name == pattern
}
