package watcher

import (
	"context"
	"time"

	"go.xrstf.de/stalk/pkg/cache"
	"go.xrstf.de/stalk/pkg/diff"

	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/watch"
)

type Watcher struct {
	differ        *diff.Differ
	log           logrus.FieldLogger
	resourceNames []string
	cache         *cache.ResourceCache
}

func NewWatcher(differ *diff.Differ, log logrus.FieldLogger, resourceNames []string) *Watcher {
	return &Watcher{
		differ:        differ,
		log:           log,
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
			if err := w.differ.PrintDiff(nil, obj, time.Time{}); err != nil {
				w.log.Errorf("Failed to show diff: %w", err)
			}
			w.cache.Set(obj)

		case watch.Modified:
			previous, lastSeen := w.cache.Get(obj)
			if err := w.differ.PrintDiff(previous, obj, lastSeen); err != nil {
				w.log.Errorf("Failed to show diff: %w", err)
			}
			w.cache.Set(obj)

		case watch.Deleted:
			if err := w.differ.PrintDiff(obj, nil, time.Now()); err != nil {
				w.log.Errorf("Failed to show diff: %w", err)
			}
			w.cache.Delete(obj)
		}
	}
}
