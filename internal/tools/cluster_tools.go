package tools

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/yulintan/k8s-mcp-server/internal/k8s"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func registerClusterTools(s *server.MCPServer, cm k8s.ClientManager) {
	s.AddTool(
		mcp.NewTool("k8s_namespaces_list",
			mcp.WithDescription("List all namespaces in a Kubernetes cluster."),
			mcp.WithString("context", mcp.Description("Kubeconfig context name. Empty = current context.")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			contextName := req.GetString("context", "")
			client, err := cm.GetClient(contextName)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			list, err := client.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			var sb strings.Builder
			sb.WriteString(fmt.Sprintf("%-40s %-10s %s\n", "NAME", "STATUS", "AGE"))
			sb.WriteString(strings.Repeat("-", 60) + "\n")
			for _, ns := range list.Items {
				sb.WriteString(fmt.Sprintf("%-40s %-10s %s\n",
					ns.Name,
					string(ns.Status.Phase),
					age(ns.CreationTimestamp.Time),
				))
			}
			return mcp.NewToolResultText(sb.String()), nil
		},
	)

	s.AddTool(
		mcp.NewTool("k8s_nodes_list",
			mcp.WithDescription("List all nodes in a Kubernetes cluster with status, roles, version, and IP."),
			mcp.WithString("context", mcp.Description("Kubeconfig context name. Empty = current context.")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			contextName := req.GetString("context", "")
			client, err := cm.GetClient(contextName)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			list, err := client.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			var sb strings.Builder
			sb.WriteString(fmt.Sprintf("%-40s %-8s %-20s %-10s %-16s %s\n", "NAME", "STATUS", "ROLES", "AGE", "VERSION", "INTERNAL-IP"))
			sb.WriteString(strings.Repeat("-", 120) + "\n")
			for _, node := range list.Items {
				sb.WriteString(fmt.Sprintf("%-40s %-8s %-20s %-10s %-16s %s\n",
					node.Name,
					nodeStatus(node),
					nodeRoles(node),
					age(node.CreationTimestamp.Time),
					node.Status.NodeInfo.KubeletVersion,
					nodeInternalIP(node),
				))
			}
			return mcp.NewToolResultText(sb.String()), nil
		},
	)

	s.AddTool(
		mcp.NewTool("k8s_events_list",
			mcp.WithDescription("List Kubernetes events in a namespace."),
			mcp.WithString("namespace", mcp.Description("Namespace. Empty = all namespaces.")),
			mcp.WithString("context", mcp.Description("Kubeconfig context name. Empty = current context.")),
			mcp.WithBoolean("warnings_only", mcp.Description("If true, only return Warning events.")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			ns := req.GetString("namespace", "")
			contextName := req.GetString("context", "")
			warningsOnly := req.GetBool("warnings_only", false)

			client, err := cm.GetClient(contextName)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			opts := metav1.ListOptions{}
			if warningsOnly {
				opts.FieldSelector = "type=Warning"
			}
			list, err := client.CoreV1().Events(ns).List(ctx, opts)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			var sb strings.Builder
			sb.WriteString(fmt.Sprintf("%-12s %-8s %-25s %-35s %s\n", "LAST SEEN", "TYPE", "REASON", "OBJECT", "MESSAGE"))
			sb.WriteString(strings.Repeat("-", 140) + "\n")
			for _, ev := range list.Items {
				obj := fmt.Sprintf("%s/%s", ev.InvolvedObject.Kind, ev.InvolvedObject.Name)
				msg := ev.Message
				if len(msg) > 60 {
					msg = msg[:57] + "..."
				}
				sb.WriteString(fmt.Sprintf("%-12s %-8s %-25s %-35s %s\n",
					age(ev.LastTimestamp.Time),
					ev.Type,
					ev.Reason,
					obj,
					msg,
				))
			}
			return mcp.NewToolResultText(sb.String()), nil
		},
	)
}

func nodeStatus(node corev1.Node) string {
	for _, cond := range node.Status.Conditions {
		if cond.Type == corev1.NodeReady {
			if cond.Status == corev1.ConditionTrue {
				return "Ready"
			}
			return "NotReady"
		}
	}
	return "Unknown"
}

func nodeRoles(node corev1.Node) string {
	var roles []string
	for label := range node.Labels {
		if strings.HasPrefix(label, "node-role.kubernetes.io/") {
			role := strings.TrimPrefix(label, "node-role.kubernetes.io/")
			roles = append(roles, role)
		}
	}
	if len(roles) == 0 {
		return "<none>"
	}
	return strings.Join(roles, ",")
}

func nodeInternalIP(node corev1.Node) string {
	for _, addr := range node.Status.Addresses {
		if addr.Type == corev1.NodeInternalIP {
			return addr.Address
		}
	}
	return "<none>"
}

func age(t time.Time) string {
	if t.IsZero() {
		return "<unknown>"
	}
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	}
}
