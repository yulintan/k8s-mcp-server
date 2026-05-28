package tools

import (
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	discoveryfake "k8s.io/client-go/discovery/fake"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	clientgotesting "k8s.io/client-go/testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/yulintan/k8s-mcp-server/internal/k8s"
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
		disco := &discoveryfake.FakeDiscovery{Fake: &clientgotesting.Fake{}}
		disco.Resources = []*metav1.APIResourceList{
			k8s.NewAPIResourceList("apps/v1", metav1.APIResource{
				Name:       "deployments",
				Kind:       "Deployment",
				Namespaced: true,
				Verbs:      []string{"get", "list"},
			}),
		}

		srv := newTestServer(newFakeClientManager(fakeClientManagerConfig{
			dynamicClient:   dynClient,
			discoveryClient: disco,
		}))

		result := callToolText(srv, "k8s_resources_list", map[string]any{
			"api_version": "apps/v1",
			"kind":        "Deployment",
			"namespace":   "prod",
		})

		Expect(result).To(ContainSubstring("api"))
		Expect(result).To(ContainSubstring("prod"))
	})

	It("lists API resources through discovery", func() {
		disco := &discoveryfake.FakeDiscovery{Fake: &clientgotesting.Fake{}}
		disco.Resources = []*metav1.APIResourceList{
			k8s.NewAPIResourceList("networking.k8s.io/v1", metav1.APIResource{
				Name:       "ingresses",
				Kind:       "Ingress",
				Namespaced: true,
				Verbs:      []string{"get", "list", "watch"},
			}),
		}

		srv := newTestServer(newFakeClientManager(fakeClientManagerConfig{
			discoveryClient: disco,
		}))

		result := callToolText(srv, "k8s_api_resources_list", map[string]any{})

		Expect(result).To(ContainSubstring("networking.k8s.io/v1"))
		Expect(result).To(ContainSubstring("Ingress"))
		Expect(result).To(ContainSubstring("ingresses"))
	})

	It("uses the high-level ingress list tool with discovery-backed resource resolution", func() {
		gvr := schema.GroupVersionResource{Group: "networking.k8s.io", Version: "v1", Resource: "ingresses"}
		ingress := &unstructured.Unstructured{
			Object: map[string]any{
				"apiVersion": "networking.k8s.io/v1",
				"kind":       "Ingress",
				"metadata": map[string]any{
					"name":              "web",
					"namespace":         "default",
					"creationTimestamp": metav1.NewTime(time.Now().Add(-time.Hour)).Format(time.RFC3339),
				},
			},
		}

		dynClient := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(
			runtime.NewScheme(),
			map[schema.GroupVersionResource]string{gvr: "IngressList"},
			ingress,
		)
		disco := &discoveryfake.FakeDiscovery{Fake: &clientgotesting.Fake{}}
		disco.Resources = []*metav1.APIResourceList{
			k8s.NewAPIResourceList("networking.k8s.io/v1", metav1.APIResource{
				Name:       "ingresses",
				Kind:       "Ingress",
				Namespaced: true,
				Verbs:      []string{"get", "list"},
			}),
		}

		srv := newTestServer(newFakeClientManager(fakeClientManagerConfig{
			dynamicClient:   dynClient,
			discoveryClient: disco,
		}))

		result := callToolText(srv, "k8s_ingresses_list", map[string]any{
			"namespace": "default",
		})

		Expect(result).To(ContainSubstring("web"))
		Expect(result).To(ContainSubstring("default"))
	})
})
