package k8s

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/discovery/fake"
	clientgotesting "k8s.io/client-go/testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("ParseGVR", func() {
	It("parses core resources", func() {
		gvr, err := ParseGVR("v1", "Pod")

		Expect(err).NotTo(HaveOccurred())
		Expect(gvr.Group).To(BeEmpty())
		Expect(gvr.Version).To(Equal("v1"))
		Expect(gvr.Resource).To(Equal("pods"))
	})

	It("parses grouped resources", func() {
		gvr, err := ParseGVR("apps/v1", "Deployment")

		Expect(err).NotTo(HaveOccurred())
		Expect(gvr.Group).To(Equal("apps"))
		Expect(gvr.Version).To(Equal("v1"))
		Expect(gvr.Resource).To(Equal("deployments"))
	})

	It("uses a better fallback for irregular-looking resource names", func() {
		gvr, err := ParseGVR("networking.k8s.io/v1", "Ingress")

		Expect(err).NotTo(HaveOccurred())
		Expect(gvr.Resource).To(Equal("ingresses"))
	})

	It("rejects missing values", func() {
		_, err := ParseGVR("", "")

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("required"))
	})

	It("resolves resources from discovery instead of guessing plurals", func() {
		disco := &fake.FakeDiscovery{Fake: &clientgotesting.Fake{}}
		disco.Resources = []*metav1.APIResourceList{
			NewAPIResourceList("networking.k8s.io/v1", metav1.APIResource{
				Name:       "ingresses",
				Kind:       "Ingress",
				Namespaced: true,
				Verbs:      []string{"get", "list"},
			}),
		}

		gvr, err := ResolveGVR(disco, "networking.k8s.io/v1", "Ingress")

		Expect(err).NotTo(HaveOccurred())
		Expect(gvr.Group).To(Equal("networking.k8s.io"))
		Expect(gvr.Version).To(Equal("v1"))
		Expect(gvr.Resource).To(Equal("ingresses"))
	})

	It("lists API resources from discovery", func() {
		disco := &fake.FakeDiscovery{Fake: &clientgotesting.Fake{}}
		disco.Resources = []*metav1.APIResourceList{
			NewAPIResourceList("v1", metav1.APIResource{
				Name:       "services",
				Kind:       "Service",
				Namespaced: true,
				Verbs:      []string{"get", "list"},
			}),
		}

		resources, err := ListAPIResources(disco)

		Expect(err).NotTo(HaveOccurred())
		Expect(resources).To(HaveLen(1))
		Expect(resources[0].GroupVersion).To(Equal("v1"))
		Expect(resources[0].Kind).To(Equal("Service"))
		Expect(resources[0].Name).To(Equal("services"))
	})
})
