package diff

import (
	"time"

	"go.xrstf.de/stalk/pkg/cache"

	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/watch"
)

type Printer struct {
	differ *Differ
	log    logrus.FieldLogger
	cache  *cache.ResourceCache
}

func NewPrinter(differ *Differ, log logrus.FieldLogger) *Printer {
	return &Printer{
		differ: differ,
		log:    log,
		cache:  cache.NewCache(),
	}
}

func (p *Printer) Print(obj *unstructured.Unstructured, event watch.EventType) {
	switch event {
	case watch.Added:
		if err := p.differ.PrintDiff(nil, obj, time.Time{}); err != nil {
			p.log.Errorf("Failed to show diff: %w", err)
		}
		p.cache.Set(obj)

	case watch.Modified:
		previous, lastSeen := p.cache.Get(obj)
		if err := p.differ.PrintDiff(previous, obj, lastSeen); err != nil {
			p.log.Errorf("Failed to show diff: %w", err)
		}
		p.cache.Set(obj)

	case watch.Deleted:
		if err := p.differ.PrintDiff(obj, nil, time.Now()); err != nil {
			p.log.Errorf("Failed to show diff: %w", err)
		}
		p.cache.Delete(obj)
	}
}
