package watcher

import (
	"context"

	"go.xrstf.de/stalk/pkg/diff"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/watch"
)

type Watcher struct {
	printer       *diff.Printer
	resourceNames []string
}

func NewWatcher(printer *diff.Printer, resourceNames []string) *Watcher {
	return &Watcher{
		printer:       printer,
		resourceNames: resourceNames,
	}
}

func (w *Watcher) Watch(ctx context.Context, wi watch.Interface) {
	for event := range wi.ResultChan() {
		obj, ok := event.Object.(*unstructured.Unstructured)
		if !ok {
			continue
		}

		include := false
		if len(w.resourceNames) > 0 {
			for _, wantedName := range w.resourceNames {
				if wantedName == obj.GetName() {
					include = true
					break
				}
			}

			if !include {
				continue
			}
		}

		w.printer.Print(obj, event.Type)
	}
}
