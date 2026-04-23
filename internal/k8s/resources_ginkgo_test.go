package k8s

import (
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

	It("rejects missing values", func() {
		_, err := ParseGVR("", "")

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("required"))
	})
})
