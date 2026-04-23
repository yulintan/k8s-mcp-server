package tools

import (
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	dynamicfake "k8s.io/client-go/dynamic/fake"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("resource tools", func() {
	It("lists generic resources through the dynamic client", func() {
		gvr := schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"}
		deployment := &unstructured.Unstructured{
			Object: map[string]any{
				"apiVersion": "apps/v1",
				"kind":       "Deployment",
				"metadata": map[string]any{
					"name":              "api",
					"namespace":         "prod",
					"creationTimestamp": metav1.NewTime(time.Now().Add(-time.Hour)).Format(time.RFC3339),
				},
			},
		}

		dynClient := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(
			runtime.NewScheme(),
			map[schema.GroupVersionResource]string{gvr: "DeploymentList"},
			deployment,
		)

		srv := newTestServer(newFakeClientManager(fakeClientManagerConfig{
			dynamicClient: dynClient,
		}))

		result := callToolText(srv, "k8s_resources_list", map[string]any{
			"api_version": "apps/v1",
			"kind":        "Deployment",
			"namespace":   "prod",
		})

		Expect(result).To(ContainSubstring("api"))
		Expect(result).To(ContainSubstring("prod"))
	})
})
