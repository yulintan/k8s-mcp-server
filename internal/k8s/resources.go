package k8s

import (
	"fmt"
	"sort"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/restmapper"
)

type APIResourceInfo struct {
	GroupVersion string
	Kind         string
	Name         string
	Namespaced   bool
	Verbs        []string
}

// ParseGroupVersionKind parses an apiVersion string and kind into a GVK.
func ParseGroupVersionKind(apiVersion, kind string) (schema.GroupVersionKind, error) {
	group, version, err := parseAPIGroupVersion(apiVersion)
	if err != nil {
		return schema.GroupVersionKind{}, err
	}
	kind = strings.TrimSpace(kind)
	if kind == "" {
		return schema.GroupVersionKind{}, fmt.Errorf("kind is required")
	}
	return schema.GroupVersionKind{Group: group, Version: version, Kind: kind}, nil
}

func parseAPIGroupVersion(apiVersion string) (string, string, error) {
	apiVersion = strings.TrimSpace(apiVersion)
	if apiVersion == "" {
		return "", "", fmt.Errorf("apiVersion is required")
	}
	var group, version string
	parts := strings.SplitN(apiVersion, "/", 2)
	if len(parts) == 2 {
		group = parts[0]
		version = parts[1]
	} else {
		group = ""
		version = parts[0]
	}
	if version == "" {
		return "", "", fmt.Errorf("apiVersion %q is missing version", apiVersion)
	}
	return group, version, nil
}

// ParseGVR keeps a deterministic fallback for tests and callers that cannot
// reach discovery. Prefer ResolveGVR for real cluster operations.
func ParseGVR(apiVersion, kind string) (schema.GroupVersionResource, error) {
	gvk, err := ParseGroupVersionKind(apiVersion, kind)
	if err != nil {
		return schema.GroupVersionResource{}, err
	}
	return schema.GroupVersionResource{
		Group:    gvk.Group,
		Version:  gvk.Version,
		Resource: naiveResourceName(gvk.Kind),
	}, nil
}

func naiveResourceName(kind string) string {
	lower := strings.ToLower(kind)
	switch {
	case strings.HasSuffix(lower, "s"):
		return lower + "es"
	case strings.HasSuffix(lower, "y"):
		return strings.TrimSuffix(lower, "y") + "ies"
	default:
		return lower + "s"
	}
}

// ResolveGVR resolves a Kind/apiVersion to the actual REST resource exposed by
// the target cluster, including irregular names and CRDs.
func ResolveGVR(dc discovery.DiscoveryInterface, apiVersion, kind string) (schema.GroupVersionResource, error) {
	gvk, err := ParseGroupVersionKind(apiVersion, kind)
	if err != nil {
		return schema.GroupVersionResource{}, err
	}
	apiGroupResources, err := restmapper.GetAPIGroupResources(dc)
	if err != nil {
		return schema.GroupVersionResource{}, fmt.Errorf("discovering API resources: %w", err)
	}
	mapper := restmapper.NewDiscoveryRESTMapper(apiGroupResources)
	mapping, err := mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return schema.GroupVersionResource{}, fmt.Errorf("resolving %s: %w", gvk.String(), err)
	}
	return mapping.Resource, nil
}

func ListAPIResources(dc discovery.DiscoveryInterface) ([]APIResourceInfo, error) {
	_, lists, err := dc.ServerGroupsAndResources()
	if err != nil && !discovery.IsGroupDiscoveryFailedError(err) {
		return nil, fmt.Errorf("discovering API resources: %w", err)
	}

	var out []APIResourceInfo
	for _, list := range lists {
		if list == nil {
			continue
		}
		for _, res := range list.APIResources {
			if strings.Contains(res.Name, "/") {
				continue
			}
			out = append(out, APIResourceInfo{
				GroupVersion: list.GroupVersion,
				Kind:         res.Kind,
				Name:         res.Name,
				Namespaced:   res.Namespaced,
				Verbs:        append([]string(nil), res.Verbs...),
			})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].GroupVersion != out[j].GroupVersion {
			return out[i].GroupVersion < out[j].GroupVersion
		}
		return out[i].Kind < out[j].Kind
	})
	return out, err
}

func NewAPIResourceList(groupVersion string, resources ...metav1.APIResource) *metav1.APIResourceList {
	return &metav1.APIResourceList{GroupVersion: groupVersion, APIResources: resources}
}
