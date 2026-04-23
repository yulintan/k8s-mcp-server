package k8s

import (
	"context"
	"os"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var _ = Describe("bulk Kubernetes operations", func() {
	var missingKubeconfig string

	BeforeEach(func() {
		tmpDir, err := os.MkdirTemp("", "k8s-bulk-tests-*")
		Expect(err).NotTo(HaveOccurred())
		missingKubeconfig = filepath.Join(tmpDir, "missing-kubeconfig")
	})

	newManager := func(explicitPath string) *clientManager {
		rules := clientcmd.NewDefaultClientConfigLoadingRules()
		rules.ExplicitPath = explicitPath
		return &clientManager{
			cache:        make(map[string]*clientEntry),
			loadingRules: rules,
		}
	}

	It("lists pods across cached contexts and preserves target ordering", func() {
		mgr := newManager(missingKubeconfig)
		now := time.Now()
		mgr.cache["prod"] = &clientEntry{
			typed: fake.NewSimpleClientset(
				&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "api-0",
						Namespace:         "default",
						CreationTimestamp: metav1.NewTime(now.Add(-time.Hour)),
						Labels:            map[string]string{"app": "api"},
					},
					Status: corev1.PodStatus{Phase: corev1.PodRunning},
				},
				&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "ignored",
						Namespace:         "default",
						CreationTimestamp: metav1.NewTime(now.Add(-time.Hour)),
						Labels:            map[string]string{"app": "worker"},
					},
				},
			),
			rest: &rest.Config{Host: "https://prod.example.invalid"},
		}

		results := mgr.BulkListPods(context.Background(), []BulkTarget{
			{Context: "prod", Namespace: "default"},
			{Context: "missing", Namespace: "default"},
		}, "app=api", 2)

		Expect(results).To(HaveLen(2))
		Expect(results[0].Target).To(Equal(BulkTarget{Context: "prod", Namespace: "default"}))
		Expect(results[0].Error).To(BeEmpty())
		Expect(results[0].Pods).To(HaveLen(1))
		Expect(results[0].Pods[0].Name).To(Equal("api-0"))

		Expect(results[1].Target).To(Equal(BulkTarget{Context: "missing", Namespace: "default"}))
		Expect(results[1].Error).NotTo(BeEmpty())
		Expect(results[1].Pods).To(BeEmpty())
	})

	It("returns exec errors per target without aborting the batch", func() {
		mgr := newManager(missingKubeconfig)

		results := mgr.BulkExec(context.Background(), []ExecTarget{
			{Context: "missing-a", Namespace: "default", PodName: "api-0", Container: "app"},
			{Context: "missing", Namespace: "default", PodName: "api-1", Container: "app"},
		}, []string{"echo", "hello"}, 2)

		Expect(results).To(HaveLen(2))

		Expect(results[0].Target).To(Equal(ExecTarget{
			Context:   "missing-a",
			Namespace: "default",
			PodName:   "api-0",
			Container: "app",
		}))
		Expect(results[0].Error).NotTo(BeEmpty())
		Expect(results[0].Stdout).To(BeEmpty())
		Expect(results[0].Stderr).To(BeEmpty())

		Expect(results[1].Target).To(Equal(ExecTarget{
			Context:   "missing",
			Namespace: "default",
			PodName:   "api-1",
			Container: "app",
		}))
		Expect(results[1].Error).NotTo(BeEmpty())
	})
})
