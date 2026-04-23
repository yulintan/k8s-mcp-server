package tools

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/yulintan/k8s-mcp-server/internal/k8s"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func registerPodTools(s *server.MCPServer, cm k8s.ClientManager) {
	// k8s_pods_list
	s.AddTool(
		mcp.NewTool("k8s_pods_list",
			mcp.WithDescription("List pods in a namespace."),
			mcp.WithString("namespace", mcp.Description("Namespace. Empty = all namespaces.")),
			mcp.WithString("context", mcp.Description("Kubeconfig context. Empty = current context.")),
			mcp.WithString("label_selector", mcp.Description("Label selector, e.g. app=nginx.")),
			mcp.WithString("field_selector", mcp.Description("Field selector, e.g. status.phase=Running.")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			ns := req.GetString("namespace", "")
			contextName := req.GetString("context", "")
			labelSel := req.GetString("label_selector", "")
			fieldSel := req.GetString("field_selector", "")

			client, err := cm.GetClient(contextName)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			list, err := client.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{
				LabelSelector: labelSel,
				FieldSelector: fieldSel,
			})
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			return mcp.NewToolResultText(formatPodList(list.Items)), nil
		},
	)

	// k8s_pods_get
	s.AddTool(
		mcp.NewTool("k8s_pods_get",
			mcp.WithDescription("Get details of a specific pod."),
			mcp.WithString("name", mcp.Required(), mcp.Description("Pod name.")),
			mcp.WithString("namespace", mcp.Required(), mcp.Description("Pod namespace.")),
			mcp.WithString("context", mcp.Description("Kubeconfig context. Empty = current context.")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			name := req.GetString("name", "")
			ns := req.GetString("namespace", "")
			contextName := req.GetString("context", "")

			client, err := cm.GetClient(contextName)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			pod, err := client.CoreV1().Pods(ns).Get(ctx, name, metav1.GetOptions{})
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			return mcp.NewToolResultText(formatPodDetail(pod)), nil
		},
	)

	// k8s_pods_logs
	s.AddTool(
		mcp.NewTool("k8s_pods_logs",
			mcp.WithDescription("Get logs from a pod container."),
			mcp.WithString("name", mcp.Required(), mcp.Description("Pod name.")),
			mcp.WithString("namespace", mcp.Required(), mcp.Description("Pod namespace.")),
			mcp.WithString("context", mcp.Description("Kubeconfig context. Empty = current context.")),
			mcp.WithString("container", mcp.Description("Container name. Empty = first container.")),
			mcp.WithNumber("tail_lines", mcp.Description("Number of lines from end. Default 100.")),
			mcp.WithBoolean("previous", mcp.Description("Return logs from previous terminated container.")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			name := req.GetString("name", "")
			ns := req.GetString("namespace", "")
			contextName := req.GetString("context", "")
			container := req.GetString("container", "")
			tailLines := int64(req.GetFloat("tail_lines", 100))
			previous := req.GetBool("previous", false)

			client, err := cm.GetClient(contextName)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			opts := &corev1.PodLogOptions{
				Container: container,
				TailLines: &tailLines,
				Previous:  previous,
			}
			stream, err := client.CoreV1().Pods(ns).GetLogs(name, opts).Stream(ctx)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			defer stream.Close()
			data, err := io.ReadAll(stream)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			return mcp.NewToolResultText(string(data)), nil
		},
	)

	// k8s_pods_exec
	s.AddTool(
		mcp.NewTool("k8s_pods_exec",
			mcp.WithDescription("Execute a command in a pod container. Returns stdout and stderr."),
			mcp.WithString("name", mcp.Required(), mcp.Description("Pod name.")),
			mcp.WithString("namespace", mcp.Required(), mcp.Description("Pod namespace.")),
			mcp.WithString("context", mcp.Description("Kubeconfig context. Empty = current context.")),
			mcp.WithString("container", mcp.Description("Container name. Empty = first container.")),
			mcp.WithArray("command", mcp.Required(), mcp.Description("Command to execute, e.g. [\"ls\", \"-la\"].")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			name := req.GetString("name", "")
			ns := req.GetString("namespace", "")
			contextName := req.GetString("context", "")
			container := req.GetString("container", "")
			cmdRaw := req.GetStringSlice("command", nil)

			if len(cmdRaw) == 0 {
				return mcp.NewToolResultError("command must not be empty"), nil
			}

			client, err := cm.GetClient(contextName)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			restCfg, err := cm.GetRESTConfig(contextName)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			out, err := k8s.ExecInPod(ctx, restCfg, client, k8s.ExecOptions{
				Namespace: ns,
				PodName:   name,
				Container: container,
				Command:   cmdRaw,
			})

			var sb strings.Builder
			if out.Stdout != "" {
				sb.WriteString("=== STDOUT ===\n")
				sb.WriteString(out.Stdout)
			}
			if out.Stderr != "" {
				sb.WriteString("=== STDERR ===\n")
				sb.WriteString(out.Stderr)
			}
			if err != nil {
				sb.WriteString(fmt.Sprintf("=== ERROR ===\n%s\n", err.Error()))
			}
			return mcp.NewToolResultText(sb.String()), nil
		},
	)

	// k8s_pods_run
	s.AddTool(
		mcp.NewTool("k8s_pods_run",
			mcp.WithDescription("Create and run a debug pod in a namespace."),
			mcp.WithString("namespace", mcp.Required(), mcp.Description("Namespace to create the pod in.")),
			mcp.WithString("context", mcp.Description("Kubeconfig context. Empty = current context.")),
			mcp.WithString("name", mcp.Description("Pod name. Empty = auto-generated.")),
			mcp.WithString("image", mcp.Description("Container image. Default: busybox:latest.")),
			mcp.WithArray("command", mcp.Description("Command to run. Default: [\"sleep\", \"3600\"].")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			ns := req.GetString("namespace", "")
			contextName := req.GetString("context", "")
			name := req.GetString("name", "")
			image := req.GetString("image", "busybox:latest")
			cmdRaw := req.GetStringSlice("command", nil)
			if len(cmdRaw) == 0 {
				cmdRaw = []string{"sleep", "3600"}
			}

			client, err := cm.GetClient(contextName)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			pod := buildDebugPodSpec(name, ns, image, cmdRaw)
			created, err := client.CoreV1().Pods(ns).Create(ctx, pod, metav1.CreateOptions{})
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			return mcp.NewToolResultText(fmt.Sprintf("Pod %q created in namespace %q\nImage: %s\nStatus: %s\n",
				created.Name, created.Namespace, image, string(created.Status.Phase))), nil
		},
	)

	// k8s_pods_delete
	s.AddTool(
		mcp.NewTool("k8s_pods_delete",
			mcp.WithDescription("Delete a pod."),
			mcp.WithString("name", mcp.Required(), mcp.Description("Pod name.")),
			mcp.WithString("namespace", mcp.Required(), mcp.Description("Pod namespace.")),
			mcp.WithString("context", mcp.Description("Kubeconfig context. Empty = current context.")),
			mcp.WithNumber("grace_period_seconds", mcp.Description("Grace period in seconds. -1 = server default.")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			name := req.GetString("name", "")
			ns := req.GetString("namespace", "")
			contextName := req.GetString("context", "")
			gracePeriod := int64(req.GetFloat("grace_period_seconds", -1))

			client, err := cm.GetClient(contextName)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			opts := metav1.DeleteOptions{}
			if gracePeriod >= 0 {
				opts.GracePeriodSeconds = &gracePeriod
			}
			if err := client.CoreV1().Pods(ns).Delete(ctx, name, opts); err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			return mcp.NewToolResultText(fmt.Sprintf("Pod %q deleted from namespace %q.\n", name, ns)), nil
		},
	)
}

func formatPodList(pods []corev1.Pod) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%-50s %-20s %-10s %-40s %s\n", "NAME", "NAMESPACE", "STATUS", "NODE", "AGE"))
	sb.WriteString(strings.Repeat("-", 140) + "\n")
	for _, p := range pods {
		sb.WriteString(fmt.Sprintf("%-50s %-20s %-10s %-40s %s\n",
			p.Name,
			p.Namespace,
			string(p.Status.Phase),
			p.Spec.NodeName,
			age(p.CreationTimestamp.Time),
		))
	}
	if len(pods) == 0 {
		sb.WriteString("No pods found.\n")
	}
	return sb.String()
}

func formatPodDetail(p *corev1.Pod) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Name:       %s\n", p.Name))
	sb.WriteString(fmt.Sprintf("Namespace:  %s\n", p.Namespace))
	sb.WriteString(fmt.Sprintf("Status:     %s\n", p.Status.Phase))
	sb.WriteString(fmt.Sprintf("Node:       %s\n", p.Spec.NodeName))
	sb.WriteString(fmt.Sprintf("IP:         %s\n", p.Status.PodIP))
	sb.WriteString(fmt.Sprintf("Created:    %s (%s ago)\n", p.CreationTimestamp.Format(time.RFC3339), age(p.CreationTimestamp.Time)))
	sb.WriteString("Labels:\n")
	for k, v := range p.Labels {
		sb.WriteString(fmt.Sprintf("  %s=%s\n", k, v))
	}
	sb.WriteString("Containers:\n")
	for _, c := range p.Spec.Containers {
		sb.WriteString(fmt.Sprintf("  - %s (%s)\n", c.Name, c.Image))
	}
	sb.WriteString("Conditions:\n")
	for _, cond := range p.Status.Conditions {
		sb.WriteString(fmt.Sprintf("  %s: %s\n", cond.Type, cond.Status))
	}
	return sb.String()
}

func buildDebugPodSpec(name, namespace, image string, command []string) *corev1.Pod {
	genName := ""
	if name == "" {
		genName = "debug-"
	}
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: genName,
			Name:         name,
			Namespace:    namespace,
			Labels:       map[string]string{"app": "k8s-mcp-debug"},
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
