package k8s

import (
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/runtime/schema"
)

// ParseGVR parses an apiVersion string (e.g. "apps/v1" or "v1") and a kind
// into a GroupVersionResource. The resource name is the lowercased plural kind.
func ParseGVR(apiVersion, kind string) (schema.GroupVersionResource, error) {
	if apiVersion == "" || kind == "" {
		return schema.GroupVersionResource{}, fmt.Errorf("apiVersion and kind are required")
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

	resource := strings.ToLower(kind) + "s"
	return schema.GroupVersionResource{Group: group, Version: version, Resource: resource}, nil
}
