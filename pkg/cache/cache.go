// SPDX-FileCopyrightText: 2023 Christoph Mewes
// SPDX-License-Identifier: MIT

package cache

import (
	"fmt"
	"sync"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type cacheItem struct {
	resource *unstructured.Unstructured
	lastSeen time.Time
}

type ResourceCache struct {
	resources map[string]cacheItem
	lock      *sync.RWMutex
}

func NewCache() *ResourceCache {
	return &ResourceCache{
		resources: map[string]cacheItem{},
		lock:      &sync.RWMutex{},
	}
}

func (rc *ResourceCache) Get(obj *unstructured.Unstructured) (*unstructured.Unstructured, time.Time) {
	rc.lock.RLock()
	defer rc.lock.RUnlock()

	existing, exists := rc.resources[rc.objectKey(obj)]
	if !exists {
		return nil, time.Time{}
	}

	return existing.resource.DeepCopy(), existing.lastSeen
}

func (rc *ResourceCache) Set(obj *unstructured.Unstructured) {
	rc.lock.Lock()
	defer rc.lock.Unlock()

	rc.resources[rc.objectKey(obj)] = cacheItem{
		resource: obj.DeepCopy(),
		lastSeen: time.Now(),
	}
}

func (rc *ResourceCache) Delete(obj *unstructured.Unstructured) {
	rc.lock.Lock()
	defer rc.lock.Unlock()

	delete(rc.resources, rc.objectKey(obj))
}

func (rc *ResourceCache) objectKey(obj *unstructured.Unstructured) string {
	return fmt.Sprintf("%s/%s/%s", obj.GroupVersionKind().String(), obj.GetNamespace(), obj.GetName())
}
