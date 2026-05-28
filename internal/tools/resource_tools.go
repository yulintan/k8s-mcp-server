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
	"k8s.io/apimachinery/pkg/runtime/schema"
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

			return listResource(ctx, cm, contextName, apiVersion, kind, ns, labelSel)
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

			return getResource(ctx, cm, contextName, apiVersion, kind, ns, name)
		},
	)

	s.AddTool(
		mcp.NewTool("k8s_api_resources_list",
			mcp.WithDescription("List API resources exposed by the cluster, including kind, resource name, apiVersion, scope, and verbs."),
			mcp.WithString("context", mcp.Description("Kubeconfig context. Empty = current context.")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			_ = ctx
			contextName := req.GetString("context", "")
			disco, err := cm.GetDiscoveryClient(contextName)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			resources, err := k8s.ListAPIResources(disco)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			return mcp.NewToolResultText(formatAPIResourceList(resources)), nil
		},
	)

	registerCommonResourceTools(s, cm)
}

func formatUnstructuredList(items []unstructured.Unstructured, kind string) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%-50s %-20s %-14s %-12s %s\n", "NAME", "NAMESPACE", "READY", "STATUS", "AGE"))
	sb.WriteString(strings.Repeat("-", 115) + "\n")
	for _, item := range items {
		ns := item.GetNamespace()
		createdAt := item.GetCreationTimestamp().Time
		sb.WriteString(fmt.Sprintf("%-50s %-20s %-14s %-12s %s\n",
			item.GetName(),
			ns,
			resourceReady(item),
			resourceStatus(item),
			age(createdAt),
		))
	}
	if len(items) == 0 {
		sb.WriteString(fmt.Sprintf("No %s resources found.\n", kind))
	}
	return sb.String()
}

func registerCommonResourceTools(s *server.MCPServer, cm k8s.ClientManager) {
	common := []struct {
		prefix     string
		plural     string
		apiVersion string
		kind       string
	}{
		{prefix: "k8s_deployments", plural: "deployments", apiVersion: "apps/v1", kind: "Deployment"},
		{prefix: "k8s_services", plural: "services", apiVersion: "v1", kind: "Service"},
		{prefix: "k8s_ingresses", plural: "ingresses", apiVersion: "networking.k8s.io/v1", kind: "Ingress"},
		{prefix: "k8s_jobs", plural: "jobs", apiVersion: "batch/v1", kind: "Job"},
		{prefix: "k8s_pvcs", plural: "persistent volume claims", apiVersion: "v1", kind: "PersistentVolumeClaim"},
	}

	for _, cfg := range common {
		cfg := cfg
		s.AddTool(
			mcp.NewTool(cfg.prefix+"_list",
				mcp.WithDescription(fmt.Sprintf("List Kubernetes %s.", cfg.plural)),
				mcp.WithString("namespace", mcp.Description("Namespace. Empty = all namespaces.")),
				mcp.WithString("context", mcp.Description("Kubeconfig context. Empty = current context.")),
				mcp.WithString("label_selector", mcp.Description("Label selector.")),
			),
			func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				ns := req.GetString("namespace", "")
				contextName := req.GetString("context", "")
				labelSel := req.GetString("label_selector", "")
				return listResource(ctx, cm, contextName, cfg.apiVersion, cfg.kind, ns, labelSel)
			},
		)

		s.AddTool(
			mcp.NewTool(cfg.prefix+"_get",
				mcp.WithDescription(fmt.Sprintf("Get a Kubernetes %s by name. Returns full JSON.", cfg.kind)),
				mcp.WithString("name", mcp.Required(), mcp.Description("Resource name.")),
				mcp.WithString("namespace", mcp.Description("Namespace. Required for namespaced resources.")),
				mcp.WithString("context", mcp.Description("Kubeconfig context. Empty = current context.")),
			),
			func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				name := req.GetString("name", "")
				ns := req.GetString("namespace", "")
				contextName := req.GetString("context", "")
				return getResource(ctx, cm, contextName, cfg.apiVersion, cfg.kind, ns, name)
			},
		)
	}
}

func listResource(ctx context.Context, cm k8s.ClientManager, contextName, apiVersion, kind, ns, labelSel string) (*mcp.CallToolResult, error) {
	gvr, err := resolveResource(cm, contextName, apiVersion, kind)
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
}

func getResource(ctx context.Context, cm k8s.ClientManager, contextName, apiVersion, kind, ns, name string) (*mcp.CallToolResult, error) {
	gvr, err := resolveResource(cm, contextName, apiVersion, kind)
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
}

func resolveResource(cm k8s.ClientManager, contextName, apiVersion, kind string) (schema.GroupVersionResource, error) {
	disco, err := cm.GetDiscoveryClient(contextName)
	if err != nil {
		return schema.GroupVersionResource{}, err
	}
	return k8s.ResolveGVR(disco, apiVersion, kind)
}

func formatAPIResourceList(resources []k8s.APIResourceInfo) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%-30s %-32s %-35s %-12s %s\n", "APIVERSION", "KIND", "NAME", "NAMESPACED", "VERBS"))
	sb.WriteString(strings.Repeat("-", 130) + "\n")
	for _, res := range resources {
		sb.WriteString(fmt.Sprintf("%-30s %-32s %-35s %-12t %s\n",
			res.GroupVersion,
			res.Kind,
			res.Name,
			res.Namespaced,
			strings.Join(res.Verbs, ","),
		))
	}
	if len(resources) == 0 {
		sb.WriteString("No API resources found.\n")
	}
	return sb.String()
}

func resourceReady(item unstructured.Unstructured) string {
	ready, found, _ := unstructured.NestedString(item.Object, "status", "ready")
	if found {
		return ready
	}
	readyReplicas, readyFound, _ := unstructured.NestedInt64(item.Object, "status", "readyReplicas")
	replicas, replicasFound, _ := unstructured.NestedInt64(item.Object, "status", "replicas")
	if readyFound || replicasFound {
		return fmt.Sprintf("%d/%d", readyReplicas, replicas)
	}
	succeeded, succeededFound, _ := unstructured.NestedInt64(item.Object, "status", "succeeded")
	active, activeFound, _ := unstructured.NestedInt64(item.Object, "status", "active")
	if succeededFound || activeFound {
		return fmt.Sprintf("succeeded=%d active=%d", succeeded, active)
	}
	return "-"
}

func resourceStatus(item unstructured.Unstructured) string {
	phase, found, _ := unstructured.NestedString(item.Object, "status", "phase")
	if found && phase != "" {
		return phase
	}
	status, found, _ := unstructured.NestedString(item.Object, "status", "status")
	if found && status != "" {
		return status
	}
	typ, found, _ := unstructured.NestedString(item.Object, "spec", "type")
	if found && typ != "" {
		return typ
	}
	return "-"
}
