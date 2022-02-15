package watcher

import (
	"context"
	"time"

	"go.xrstf.de/stalk/pkg/cache"
	"go.xrstf.de/stalk/pkg/diff"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/watch"
)

type Watcher struct {
	differ        *diff.Differ
	resourceNames []string
	cache         *cache.ResourceCache
}

func NewWatcher(differ *diff.Differ, resourceNames []string) *Watcher {
	return &Watcher{
		differ:        differ,
		resourceNames: resourceNames,
		cache:         cache.NewCache(),
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

		switch event.Type {
		case watch.Added:
			w.differ.PrintDiff(nil, obj, time.Time{})
			w.cache.Set(obj)

		case watch.Modified:
			previous, lastSeen := w.cache.Get(obj)
			w.differ.PrintDiff(previous, obj, lastSeen)
			w.cache.Set(obj)

		case watch.Deleted:
			w.differ.PrintDiff(obj, nil, time.Now())
			w.cache.Delete(obj)
		}
	}
}
