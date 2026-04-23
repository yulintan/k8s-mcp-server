package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/yulintan/k8s-mcp-server/internal/k8s"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func registerResourceTools(s *server.MCPServer, cm k8s.ClientManager) {
	s.AddTool(
		mcp.NewTool("k8s_resources_list",
			mcp.WithDescription("List any Kubernetes resource type by apiVersion and kind."),
			mcp.WithString("api_version", mcp.Required(), mcp.Description("API version, e.g. apps/v1 or v1.")),
			mcp.WithString("kind", mcp.Required(), mcp.Description("Resource kind, e.g. Deployment, ConfigMap.")),
			mcp.WithString("namespace", mcp.Description("Namespace. Empty = all namespaces (if namespaced).")),
			mcp.WithString("context", mcp.Description("Kubeconfig context. Empty = current context.")),
			mcp.WithString("label_selector", mcp.Description("Label selector.")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			apiVersion := req.GetString("api_version", "")
			kind := req.GetString("kind", "")
			ns := req.GetString("namespace", "")
			contextName := req.GetString("context", "")
			labelSel := req.GetString("label_selector", "")

			gvr, err := k8s.ParseGVR(apiVersion, kind)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			dynClient, err := cm.GetDynamicClient(contextName)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			list, err := dynClient.Resource(gvr).Namespace(ns).List(ctx, metav1.ListOptions{
				LabelSelector: labelSel,
			})
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			return mcp.NewToolResultText(formatUnstructuredList(list.Items, kind)), nil
		},
	)

	s.AddTool(
		mcp.NewTool("k8s_resources_get",
			mcp.WithDescription("Get a specific Kubernetes resource by apiVersion, kind, and name. Returns full JSON."),
			mcp.WithString("api_version", mcp.Required(), mcp.Description("API version, e.g. apps/v1 or v1.")),
			mcp.WithString("kind", mcp.Required(), mcp.Description("Resource kind, e.g. Deployment.")),
			mcp.WithString("name", mcp.Required(), mcp.Description("Resource name.")),
			mcp.WithString("namespace", mcp.Description("Namespace. Required for namespaced resources.")),
			mcp.WithString("context", mcp.Description("Kubeconfig context. Empty = current context.")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			apiVersion := req.GetString("api_version", "")
			kind := req.GetString("kind", "")
			name := req.GetString("name", "")
			ns := req.GetString("namespace", "")
			contextName := req.GetString("context", "")

			gvr, err := k8s.ParseGVR(apiVersion, kind)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			dynClient, err := cm.GetDynamicClient(contextName)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			obj, err := dynClient.Resource(gvr).Namespace(ns).Get(ctx, name, metav1.GetOptions{})
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			data, err := json.MarshalIndent(obj.Object, "", "  ")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			return mcp.NewToolResultText(string(data)), nil
		},
	)
}

func formatUnstructuredList(items []unstructured.Unstructured, kind string) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%-50s %-20s %s\n", "NAME", "NAMESPACE", "AGE"))
	sb.WriteString(strings.Repeat("-", 90) + "\n")
	for _, item := range items {
		ns := item.GetNamespace()
		createdAt := item.GetCreationTimestamp().Time
		sb.WriteString(fmt.Sprintf("%-50s %-20s %s\n",
			item.GetName(),
			ns,
			age(createdAt),
		))
	}
	if len(items) == 0 {
		sb.WriteString(fmt.Sprintf("No %s resources found.\n", kind))
	}
	return sb.String()
}
