package watcher

import (
	"context"
	"fmt"
	"time"

	"go.xrstf.de/stalk/pkg/cache"
	"go.xrstf.de/stalk/pkg/diff"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/watch"
)

type Watcher struct {
	differ *diff.Differ
	cache  *cache.ResourceCache
}

func NewWatcher(differ *diff.Differ) *Watcher {
	return &Watcher{
		differ: differ,
		cache:  cache.NewCache(),
	}
}

func (w *Watcher) Watch(ctx context.Context, wi watch.Interface) {
	for event := range wi.ResultChan() {
		obj, ok := event.Object.(*unstructured.Unstructured)
		if !ok {
			continue
		}

		// if hideManagedFields {
		// 	metaObject.SetManagedFields(nil)
		// }

		switch event.Type {
		case watch.Added:
			w.printDiff(nil, obj, time.Time{})
			w.cache.Set(obj)

		case watch.Modified:
			previous, lastSeen := w.cache.Get(obj)
			w.printDiff(previous, obj, lastSeen)
			w.cache.Set(obj)

		case watch.Deleted:
			w.printDiff(obj, nil, time.Now())
			w.cache.Delete(obj)
		}
	}
}

func (w *Watcher) printDiff(oldObj, newObj *unstructured.Unstructured, lastSeen time.Time) {
	w.differ.PrintDiff(oldObj, newObj, lastSeen)
	fmt.Printf("\n")
}
