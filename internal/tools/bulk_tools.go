package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/yulintan/k8s-mcp-server/internal/k8s"
)

func registerBulkTools(s *server.MCPServer, cm k8s.ClientManager) {
	// k8s_pods_list_bulk
	s.AddTool(
		mcp.NewTool("k8s_pods_list_bulk",
			mcp.WithDescription("List pods across multiple namespaces and/or contexts concurrently. Faster and more token-efficient than sequential calls."),
			mcp.WithArray("targets", mcp.Required(), mcp.Description(`Array of {context, namespace} objects. Example: [{"context":"prod","namespace":"default"},{"context":"","namespace":"kube-system"}]`)),
			mcp.WithString("label_selector", mcp.Description("Label selector applied to all targets.")),
			mcp.WithNumber("max_concurrency", mcp.Description("Max concurrent requests. Default 20.")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			targets, err := parseBulkTargets(req, "targets")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			labelSel := req.GetString("label_selector", "")
			maxConc := int(req.GetFloat("max_concurrency", 20))

			results := cm.BulkListPods(ctx, targets, labelSel, maxConc)

			var sb strings.Builder
			for _, r := range results {
				ctxLabel := r.Target.Context
				if ctxLabel == "" {
					ctxLabel = "(current)"
				}
				nsLabel := r.Target.Namespace
				if nsLabel == "" {
					nsLabel = "(all)"
				}
				sb.WriteString(fmt.Sprintf("=== context=%s namespace=%s ===\n", ctxLabel, nsLabel))
				if r.Error != "" {
					sb.WriteString(fmt.Sprintf("ERROR: %s\n", r.Error))
				} else {
					sb.WriteString(formatPodList(r.Pods))
				}
				sb.WriteString("\n")
			}
			return mcp.NewToolResultText(sb.String()), nil
		},
	)

	// k8s_pods_exec_bulk
	s.AddTool(
		mcp.NewTool("k8s_pods_exec_bulk",
			mcp.WithDescription("Execute the same command across multiple pods concurrently. Returns aggregated stdout/stderr per pod."),
			mcp.WithArray("targets", mcp.Required(), mcp.Description(`Array of {context, namespace, pod_name, container} objects. container is optional.`)),
			mcp.WithArray("command", mcp.Required(), mcp.Description("Command to run, e.g. [\"hostname\"].")),
			mcp.WithNumber("max_concurrency", mcp.Description("Max concurrent exec calls. Default 20.")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			targets, err := parseExecTargets(req, "targets")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			cmdRaw := req.GetStringSlice("command", nil)
			if len(cmdRaw) == 0 {
				return mcp.NewToolResultError("command must not be empty"), nil
			}
			maxConc := int(req.GetFloat("max_concurrency", 20))

			results := cm.BulkExec(ctx, targets, cmdRaw, maxConc)

			var sb strings.Builder
			for _, r := range results {
				ctxLabel := r.Target.Context
				if ctxLabel == "" {
					ctxLabel = "(current)"
				}
				sb.WriteString(fmt.Sprintf("=== %s/%s/%s ===\n", ctxLabel, r.Target.Namespace, r.Target.PodName))
				if r.Error != "" {
					sb.WriteString(fmt.Sprintf("ERROR: %s\n", r.Error))
				}
				if r.Stdout != "" {
					sb.WriteString(r.Stdout)
				}
				if r.Stderr != "" {
					sb.WriteString(fmt.Sprintf("[stderr] %s", r.Stderr))
				}
				sb.WriteString("\n")
			}
			return mcp.NewToolResultText(sb.String()), nil
		},
	)

	// k8s_debug_pods_create_bulk
	s.AddTool(
		mcp.NewTool("k8s_debug_pods_create_bulk",
			mcp.WithDescription("Create debug pods in multiple namespaces/contexts concurrently."),
			mcp.WithArray("targets", mcp.Required(), mcp.Description(`Array of {context, namespace, name} objects. name is optional (auto-generated if empty).`)),
			mcp.WithString("image", mcp.Description("Container image. Default: busybox:latest.")),
			mcp.WithArray("command", mcp.Description("Command to run. Default: [\"sleep\",\"3600\"].")),
			mcp.WithNumber("max_concurrency", mcp.Description("Max concurrent create calls. Default 20.")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			targets, err := parseDebugPodTargets(req, "targets")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			image := req.GetString("image", "busybox:latest")
			cmdRaw := req.GetStringSlice("command", nil)
			if len(cmdRaw) == 0 {
				cmdRaw = []string{"sleep", "3600"}
			}
			maxConc := int(req.GetFloat("max_concurrency", 20))

			results := cm.BulkCreateDebugPods(ctx, targets, image, cmdRaw, maxConc)

			var sb strings.Builder
			sb.WriteString(fmt.Sprintf("%-20s %-20s %-40s %-10s %s\n", "CONTEXT", "NAMESPACE", "POD NAME", "STATUS", "ERROR"))
			sb.WriteString(strings.Repeat("-", 110) + "\n")
			for _, r := range results {
				ctxLabel := r.Target.Context
				if ctxLabel == "" {
					ctxLabel = "(current)"
				}
				podName := r.Target.Name
				status := ""
				if r.CreatedPod != nil {
					podName = r.CreatedPod.Name
					status = string(r.CreatedPod.Status.Phase)
					if status == "" {
						status = "Pending"
					}
				}
				sb.WriteString(fmt.Sprintf("%-20s %-20s %-40s %-10s %s\n",
					ctxLabel, r.Target.Namespace, podName, status, r.Error))
			}
			return mcp.NewToolResultText(sb.String()), nil
		},
	)
}

// marshalParam extracts a parameter from request arguments and marshals it to JSON bytes.
func marshalParam(req mcp.CallToolRequest, param string) ([]byte, error) {
	args := req.GetArguments()
	val, ok := args[param]
	if !ok {
		return []byte("[]"), nil
	}
	return json.Marshal(val)
}

// parseBulkTargets decodes the "targets" parameter as []k8s.BulkTarget.
func parseBulkTargets(req mcp.CallToolRequest, param string) ([]k8s.BulkTarget, error) {
	raw, err := marshalParam(req, param)
	if err != nil {
		return nil, fmt.Errorf("encoding targets: %w", err)
	}
	var items []struct {
		Context   string `json:"context"`
		Namespace string `json:"namespace"`
	}
	if err := json.Unmarshal(raw, &items); err != nil {
		return nil, fmt.Errorf("parsing targets: %w", err)
	}
	out := make([]k8s.BulkTarget, len(items))
	for i, item := range items {
		out[i] = k8s.BulkTarget{Context: item.Context, Namespace: item.Namespace}
	}
	return out, nil
}

func parseExecTargets(req mcp.CallToolRequest, param string) ([]k8s.ExecTarget, error) {
	raw, err := marshalParam(req, param)
	if err != nil {
		return nil, fmt.Errorf("encoding targets: %w", err)
	}
	var items []struct {
		Context   string `json:"context"`
		Namespace string `json:"namespace"`
		PodName   string `json:"pod_name"`
		Container string `json:"container"`
	}
	if err := json.Unmarshal(raw, &items); err != nil {
		return nil, fmt.Errorf("parsing targets: %w", err)
	}
	out := make([]k8s.ExecTarget, len(items))
	for i, item := range items {
		out[i] = k8s.ExecTarget{Context: item.Context, Namespace: item.Namespace, PodName: item.PodName, Container: item.Container}
	}
	return out, nil
}

func parseDebugPodTargets(req mcp.CallToolRequest, param string) ([]k8s.DebugPodTarget, error) {
	raw, err := marshalParam(req, param)
	if err != nil {
		return nil, fmt.Errorf("encoding targets: %w", err)
	}
	var items []struct {
		Context   string `json:"context"`
		Namespace string `json:"namespace"`
		Name      string `json:"name"`
	}
	if err := json.Unmarshal(raw, &items); err != nil {
		return nil, fmt.Errorf("parsing targets: %w", err)
	}
	out := make([]k8s.DebugPodTarget, len(items))
	for i, item := range items {
		out[i] = k8s.DebugPodTarget{Context: item.Context, Namespace: item.Namespace, Name: item.Name}
	}
	return out, nil
}
