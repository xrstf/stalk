package kubernetes

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	memory "k8s.io/client-go/discovery/cached"
	"k8s.io/client-go/discovery/cached/disk"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
)

func CreateRESTMapper(config *rest.Config, log logrus.FieldLogger) (meta.RESTMapper, discovery.CachedDiscoveryInterface, error) {
	var (
		discoveryClient discovery.DiscoveryInterface
		cache           discovery.CachedDiscoveryInterface
	)

	home, err := os.UserHomeDir()
	if err != nil {
		log.Warn("Cannot determine home directory, will disable discovery cache.")

		discoveryClient, err = discovery.NewDiscoveryClientForConfig(config)
		if err != nil {
			return nil, nil, err
		}

		cache = memory.NewMemCacheClient(discoveryClient)
	} else {
		cacheDir := filepath.Join(home, ".kube", "cache")

		httpCacheDir := filepath.Join(cacheDir, "http")
		discoveryCacheDir := computeDiscoverCacheDir(filepath.Join(cacheDir, "discovery"), config.Host)

		client, err := disk.NewCachedDiscoveryClientForConfig(config, discoveryCacheDir, httpCacheDir, 6*time.Hour)
		if err != nil {
			return nil, nil, err
		}

		discoveryClient = client
		cache = client
	}

	mapper := restmapper.NewDeferredDiscoveryRESTMapper(cache)
	fancyMapper := restmapper.NewShortcutExpander(mapper, discoveryClient)

	return fancyMapper, cache, nil
}

// overlyCautiousIllegalFileCharacters matches characters that *might* not be supported.  Windows is really restrictive, so this is really restrictive
var overlyCautiousIllegalFileCharacters = regexp.MustCompile(`[^(\w/\.)]`)

// computeDiscoverCacheDir takes the parentDir and the host and comes up with a "usually non-colliding" name.
// This is copied from
// https://github.com/kubernetes/kubernetes/blob/0b8d725f5a04178caf09cd802305c4b8370db65e/staging/src/k8s.io/cli-runtime/pkg/genericclioptions/config_flags.go
func computeDiscoverCacheDir(parentDir, host string) string {
	// strip the optional scheme from host if its there:
	schemelessHost := strings.Replace(strings.Replace(host, "https://", "", 1), "http://", "", 1)
	// now do a simple collapse of non-AZ09 characters.  Collisions are possible but unlikely.  Even if we do collide the problem is short lived
	safeHost := overlyCautiousIllegalFileCharacters.ReplaceAllString(schemelessHost, "_")
	return filepath.Join(parentDir, safeHost)
}

func GetDynamicInterface(gvk schema.GroupVersionKind, namespace string, dynamicClient dynamic.Interface, mapper meta.RESTMapper) (dynamic.ResourceInterface, error) {
	mapping, err := mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return nil, fmt.Errorf("failed to determine mapping: %w", err)
	}

	namespaced := mapping.Scope.Name() == meta.RESTScopeNameNamespace

	var dr dynamic.ResourceInterface
	if namespaced {
		// namespaced resources should specify the namespace
		dr = dynamicClient.Resource(mapping.Resource).Namespace(namespace)
	} else {
		// for cluster-wide resources
		dr = dynamicClient.Resource(mapping.Resource)
	}

	return dr, nil
}

func MappingFor(restMapper meta.RESTMapper, cache discovery.CachedDiscoveryInterface, resourceOrKindArg string) (*meta.RESTMapping, error) {
	mapping, err := mappingFor(restMapper, resourceOrKindArg)
	if meta.IsNoMatchError(err) {
		cache.Invalidate()

		// try again
		mapping, err = mappingFor(restMapper, resourceOrKindArg)
	}

	if meta.IsNoMatchError(err) {
		return nil, fmt.Errorf("the server doesn't have a resource type %q", resourceOrKindArg)
	}

	return mapping, err
}

// mappingFor is copied straight from kubectl:
// https://github.com/kubernetes/kubernetes/blob/0b8d725f5a04178caf09cd802305c4b8370db65e/staging/src/k8s.io/cli-runtime/pkg/resource/builder.go
func mappingFor(restMapper meta.RESTMapper, resourceOrKindArg string) (*meta.RESTMapping, error) {
	fullySpecifiedGVR, groupResource := schema.ParseResourceArg(resourceOrKindArg)
	gvk := schema.GroupVersionKind{}

	if fullySpecifiedGVR != nil {
		gvk, _ = restMapper.KindFor(*fullySpecifiedGVR)
	}
	if gvk.Empty() {
		gvk, _ = restMapper.KindFor(groupResource.WithVersion(""))
	}
	if !gvk.Empty() {
		return restMapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	}

	fullySpecifiedGVK, groupKind := schema.ParseKindArg(resourceOrKindArg)
	if fullySpecifiedGVK == nil {
		gvk := groupKind.WithVersion("")
		fullySpecifiedGVK = &gvk
	}

	if !fullySpecifiedGVK.Empty() {
		if mapping, err := restMapper.RESTMapping(fullySpecifiedGVK.GroupKind(), fullySpecifiedGVK.Version); err == nil {
			return mapping, nil
		}
	}

	return restMapper.RESTMapping(groupKind, gvk.Version)
}
