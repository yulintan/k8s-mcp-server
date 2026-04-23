package tools

import (
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubernetesfake "k8s.io/client-go/kubernetes/fake"
)

var _ = Describe("cluster tools", func() {
	It("lists namespaces from the typed Kubernetes client", func() {
		clientset := kubernetesfake.NewSimpleClientset(
			&corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "default",
					CreationTimestamp: metav1.NewTime(time.Now().Add(-2 * time.Hour)),
				},
				Status: corev1.NamespaceStatus{Phase: corev1.NamespaceActive},
			},
			&corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "kube-system",
					CreationTimestamp: metav1.NewTime(time.Now().Add(-24 * time.Hour)),
				},
				Status: corev1.NamespaceStatus{Phase: corev1.NamespaceActive},
			},
		)
		srv := newTestServer(newFakeClientManager(fakeClientManagerConfig{
			typedClient: clientset,
		}))

		result := callToolText(srv, "k8s_namespaces_list", map[string]any{})

		Expect(result).To(ContainSubstring("default"))
		Expect(result).To(ContainSubstring("kube-system"))
		Expect(result).To(ContainSubstring("Active"))
	})
})
