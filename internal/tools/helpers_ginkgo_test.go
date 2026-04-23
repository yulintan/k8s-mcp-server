package tools

import (
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

var _ = Describe("tool helpers", func() {
	Describe("parseBulkTargets", func() {
		It("decodes bulk target arguments", func() {
			req := mcp.CallToolRequest{}
			req.Params.Arguments = map[string]any{
				"targets": []map[string]any{
					{"context": "prod", "namespace": "default"},
					{"context": "", "namespace": "kube-system"},
				},
			}

			targets, err := parseBulkTargets(req, "targets")

			Expect(err).NotTo(HaveOccurred())
			Expect(targets).To(HaveLen(2))
			Expect(targets[0].Context).To(Equal("prod"))
			Expect(targets[0].Namespace).To(Equal("default"))
			Expect(targets[1].Context).To(BeEmpty())
			Expect(targets[1].Namespace).To(Equal("kube-system"))
		})
	})

	Describe("parseExecTargets", func() {
		It("returns a parsing error for invalid shapes", func() {
			req := mcp.CallToolRequest{}
			req.Params.Arguments = map[string]any{"targets": "not-an-array"}

			_, err := parseExecTargets(req, "targets")

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("parsing targets"))
		})
	})

	Describe("buildDebugPodSpec", func() {
		It("fills the generated name path for unnamed pods", func() {
			pod := buildDebugPodSpec("", "tools", "busybox:latest", []string{"sleep", "60"})

			Expect(pod.Name).To(BeEmpty())
			Expect(pod.GenerateName).To(Equal("debug-"))
			Expect(pod.Namespace).To(Equal("tools"))
			Expect(pod.Spec.RestartPolicy).To(Equal(corev1.RestartPolicyNever))
			Expect(pod.Spec.Containers).To(HaveLen(1))
			Expect(pod.Spec.Containers[0].Name).To(Equal("debug"))
			Expect(pod.Spec.Containers[0].Image).To(Equal("busybox:latest"))
			Expect(pod.Spec.Containers[0].Command).To(Equal([]string{"sleep", "60"}))
		})
	})

	Describe("formatters", func() {
		It("renders an empty pod list with headers and empty state", func() {
			out := formatPodList(nil)

			Expect(out).To(ContainSubstring("NAME"))
			Expect(out).To(ContainSubstring("No pods found."))
		})

		It("renders an empty unstructured list with kind-specific empty state", func() {
			out := formatUnstructuredList(nil, "Deployment")

			Expect(out).To(ContainSubstring("No Deployment resources found."))
		})

		It("renders pod detail with the important fields", func() {
			now := time.Now().Add(-2 * time.Hour)
			pod := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "api-0",
					Namespace:         "default",
					CreationTimestamp: metav1.NewTime(now),
					Labels:            map[string]string{"app": "api"},
				},
				Spec: corev1.PodSpec{
					NodeName: "node-a",
					Containers: []corev1.Container{
						{Name: "api", Image: "nginx:1.0"},
					},
				},
				Status: corev1.PodStatus{
					Phase: corev1.PodRunning,
					PodIP: "10.0.0.10",
					Conditions: []corev1.PodCondition{
						{Type: corev1.PodReady, Status: corev1.ConditionTrue},
					},
				},
			}

			out := formatPodDetail(pod)

			Expect(out).To(ContainSubstring("Name:       api-0"))
			Expect(out).To(ContainSubstring("Namespace:  default"))
			Expect(out).To(ContainSubstring("Status:     Running"))
			Expect(out).To(ContainSubstring("Node:       node-a"))
			Expect(out).To(ContainSubstring("IP:         10.0.0.10"))
			Expect(out).To(ContainSubstring("app=api"))
			Expect(out).To(ContainSubstring("api (nginx:1.0)"))
			Expect(out).To(ContainSubstring("Ready: True"))
		})

		It("renders unstructured items", func() {
			item := unstructured.Unstructured{}
			item.SetName("demo")
			item.SetNamespace("default")
			item.SetCreationTimestamp(metav1.NewTime(time.Now().Add(-time.Hour)))

			out := formatUnstructuredList([]unstructured.Unstructured{item}, "Deployment")

			Expect(out).To(ContainSubstring("demo"))
			Expect(out).To(ContainSubstring("default"))
		})
	})

	Describe("node helpers", func() {
		It("extracts status, roles, and internal IP", func() {
			node := corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"node-role.kubernetes.io/control-plane": "",
						"node-role.kubernetes.io/worker":        "",
					},
				},
				Status: corev1.NodeStatus{
					Conditions: []corev1.NodeCondition{
						{Type: corev1.NodeReady, Status: corev1.ConditionTrue},
					},
					Addresses: []corev1.NodeAddress{
						{Type: corev1.NodeInternalIP, Address: "192.168.1.10"},
					},
				},
			}

			Expect(nodeStatus(node)).To(Equal("Ready"))
			Expect(strings.Split(nodeRoles(node), ",")).To(ConsistOf("control-plane", "worker"))
			Expect(nodeInternalIP(node)).To(Equal("192.168.1.10"))
		})
	})

	Describe("age", func() {
		It("handles zero timestamps", func() {
			Expect(age(time.Time{})).To(Equal("<unknown>"))
		})

		It("formats recent durations in the expected unit", func() {
			Expect(age(time.Now().Add(-30 * time.Second))).To(HaveSuffix("s"))
			Expect(age(time.Now().Add(-5 * time.Minute))).To(HaveSuffix("m"))
			Expect(age(time.Now().Add(-3 * time.Hour))).To(HaveSuffix("h"))
			Expect(age(time.Now().Add(-48 * time.Hour))).To(HaveSuffix("d"))
		})
	})
})
