package tools

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/yulintan/k8s-mcp-server/internal/k8s"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("bulk tools", func() {
	It("formats bulk pod list results from the client manager", func() {
		srv := newTestServer(newFakeClientManager(fakeClientManagerConfig{
			bulkListPodsFn: func(ctx context.Context, targets []k8s.BulkTarget, labelSelector string, maxConcurrency int) []k8s.PodListResult {
				Expect(targets).To(HaveLen(2))
				Expect(labelSelector).To(Equal("app=api"))
				Expect(maxConcurrency).To(Equal(7))

				return []k8s.PodListResult{
					{
						Target: targets[0],
						Pods: []corev1.Pod{
							{
								ObjectMeta: metav1.ObjectMeta{
									Name:              "api-0",
									Namespace:         "default",
									CreationTimestamp: metav1.NewTime(time.Now().Add(-time.Hour)),
								},
								Status: corev1.PodStatus{Phase: corev1.PodRunning},
							},
						},
					},
					{
						Target: targets[1],
						Error:  "boom",
					},
				}
			},
		}))

		result := callToolText(srv, "k8s_pods_list_bulk", map[string]any{
			"targets": []map[string]any{
				{"context": "prod", "namespace": "default"},
				{"context": "", "namespace": "kube-system"},
			},
			"label_selector":  "app=api",
			"max_concurrency": 7,
		})

		Expect(result).To(ContainSubstring("=== context=prod namespace=default ==="))
		Expect(result).To(ContainSubstring("api-0"))
		Expect(result).To(ContainSubstring("=== context=(current) namespace=kube-system ==="))
		Expect(result).To(ContainSubstring("ERROR: boom"))
	})
})
