package k8s

import (
	"context"

	"golang.org/x/sync/errgroup"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// BulkTarget identifies a (context, namespace) pair for bulk operations.
type BulkTarget struct {
	Context   string
	Namespace string
}

// PodListResult is the result from a single BulkTarget for list operations.
type PodListResult struct {
	Target BulkTarget
	Pods   []corev1.Pod
	Error  string
}

// BulkListPods fans out List calls across all targets concurrently.
func (m *clientManager) BulkListPods(ctx context.Context, targets []BulkTarget, labelSelector string, maxConcurrency int) []PodListResult {
	results := make([]PodListResult, len(targets))

	g, gctx := errgroup.WithContext(ctx)
	if maxConcurrency > 0 {
		g.SetLimit(maxConcurrency)
	}

	for i, target := range targets {
		i, target := i, target
		g.Go(func() error {
			client, err := m.GetClient(target.Context)
			if err != nil {
				results[i] = PodListResult{Target: target, Error: err.Error()}
				return nil
			}
			list, err := client.CoreV1().Pods(target.Namespace).List(gctx, metav1.ListOptions{
				LabelSelector: labelSelector,
			})
			if err != nil {
				results[i] = PodListResult{Target: target, Error: err.Error()}
				return nil
			}
			results[i] = PodListResult{Target: target, Pods: list.Items}
			return nil
		})
	}

	_ = g.Wait()
	return results
}

// ExecTarget specifies a single pod+command for bulk exec.
type ExecTarget struct {
	Context   string
	Namespace string
	PodName   string
	Container string
}

// BulkExecResult is the result from a single ExecTarget.
type BulkExecResult struct {
	Target ExecTarget
	Stdout string
	Stderr string
	Error  string
}

// BulkExec fans out exec calls across all targets concurrently.
func (m *clientManager) BulkExec(ctx context.Context, targets []ExecTarget, command []string, maxConcurrency int) []BulkExecResult {
	results := make([]BulkExecResult, len(targets))

	g, gctx := errgroup.WithContext(ctx)
	if maxConcurrency > 0 {
		g.SetLimit(maxConcurrency)
	}

	for i, target := range targets {
		i, target := i, target
		g.Go(func() error {
			client, err := m.GetClient(target.Context)
			if err != nil {
				results[i] = BulkExecResult{Target: target, Error: err.Error()}
				return nil
			}
			restCfg, err := m.GetRESTConfig(target.Context)
			if err != nil {
				results[i] = BulkExecResult{Target: target, Error: err.Error()}
				return nil
			}
			out, err := ExecInPod(gctx, restCfg, client, ExecOptions{
				Namespace: target.Namespace,
				PodName:   target.PodName,
				Container: target.Container,
				Command:   command,
			})
			if err != nil {
				results[i] = BulkExecResult{Target: target, Stdout: out.Stdout, Stderr: out.Stderr, Error: err.Error()}
				return nil
			}
			results[i] = BulkExecResult{Target: target, Stdout: out.Stdout, Stderr: out.Stderr}
			return nil
		})
	}

	_ = g.Wait()
	return results
}

// DebugPodTarget specifies a debug pod to create in a given context/namespace.
type DebugPodTarget struct {
	Context   string
	Namespace string
	Name      string
}

// DebugPodResult is the result from a single DebugPodTarget creation.
type DebugPodResult struct {
	Target     DebugPodTarget
	CreatedPod *corev1.Pod
	Error      string
}

// BulkCreateDebugPods fans out pod creation calls across all targets concurrently.
func (m *clientManager) BulkCreateDebugPods(ctx context.Context, targets []DebugPodTarget, image string, command []string, maxConcurrency int) []DebugPodResult {
	results := make([]DebugPodResult, len(targets))

	g, gctx := errgroup.WithContext(ctx)
	if maxConcurrency > 0 {
		g.SetLimit(maxConcurrency)
	}

	for i, target := range targets {
		i, target := i, target
		g.Go(func() error {
			client, err := m.GetClient(target.Context)
			if err != nil {
				results[i] = DebugPodResult{Target: target, Error: err.Error()}
				return nil
			}
			pod := buildDebugPod(target.Name, target.Namespace, image, command)
			created, err := client.CoreV1().Pods(target.Namespace).Create(gctx, pod, metav1.CreateOptions{})
			if err != nil {
				results[i] = DebugPodResult{Target: target, Error: err.Error()}
				return nil
			}
			results[i] = DebugPodResult{Target: target, CreatedPod: created}
			return nil
		})
	}

	_ = g.Wait()
	return results
}

func buildDebugPod(name, namespace, image string, command []string) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: func() string {
				if name == "" {
					return "debug-"
				}
				return ""
			}(),
			Name:      name,
			Namespace: namespace,
			Labels:    map[string]string{"app": "k8s-mcp-debug"},
		},
		Spec: corev1.PodSpec{
			RestartPolicy: corev1.RestartPolicyNever,
			Containers: []corev1.Container{
				{
					Name:    "debug",
					Image:   image,
					Command: command,
				},
			},
		},
	}
}
