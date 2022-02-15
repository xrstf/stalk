package cache

import (
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type cacheItem struct {
	resource *unstructured.Unstructured
	lastSeen time.Time
}

type ResourceCache struct {
	resources map[string]cacheItem
}

func NewCache() *ResourceCache {
	return &ResourceCache{
		resources: map[string]cacheItem{},
	}
}

func (rc *ResourceCache) Get(obj *unstructured.Unstructured) (*unstructured.Unstructured, time.Time) {
	existing, exists := rc.resources[rc.objectKey(obj)]
	if !exists {
		return nil, time.Time{}
	}

	return existing.resource.DeepCopy(), existing.lastSeen
}

func (rc *ResourceCache) Set(obj *unstructured.Unstructured) {
	rc.resources[rc.objectKey(obj)] = cacheItem{
		resource: obj.DeepCopy(),
		lastSeen: time.Now(),
	}
}

func (rc *ResourceCache) Delete(obj *unstructured.Unstructured) {
	delete(rc.resources, rc.objectKey(obj))
}

func (rc *ResourceCache) objectKey(obj *unstructured.Unstructured) string {
	return fmt.Sprintf("%s/%s", obj.GetNamespace(), obj.GetName())
}
