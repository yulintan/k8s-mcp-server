package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/yulintan/k8s-mcp-server/internal/k8s"
)

func registerConfigTools(s *server.MCPServer, cm k8s.ClientManager) {
	s.AddTool(
		mcp.NewTool("k8s_contexts_list",
			mcp.WithDescription("List all Kubernetes contexts from kubeconfig. Marks the current context with *."),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			raw, err := cm.RawConfig()
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			current := raw.CurrentContext
			var sb strings.Builder
			sb.WriteString(fmt.Sprintf("%-4s %-40s %-40s %s\n", "", "NAME", "CLUSTER", "SERVER"))
			sb.WriteString(strings.Repeat("-", 120) + "\n")
			for name, ctx := range raw.Contexts {
				marker := " "
				if name == current {
					marker = "*"
				}
				cluster := raw.Clusters[ctx.Cluster]
				server := ""
				if cluster != nil {
					server = cluster.Server
				}
				sb.WriteString(fmt.Sprintf("%-4s %-40s %-40s %s\n", marker, name, ctx.Cluster, server))
			}
			return mcp.NewToolResultText(sb.String()), nil
		},
	)

	s.AddTool(
		mcp.NewTool("k8s_context_current",
			mcp.WithDescription("Get the current active Kubernetes context name and cluster server URL."),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			raw, err := cm.RawConfig()
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			current := raw.CurrentContext
			if current == "" {
				return mcp.NewToolResultText("No current context set."), nil
			}
			ctxInfo := raw.Contexts[current]
			server := ""
			if ctxInfo != nil {
				if cl := raw.Clusters[ctxInfo.Cluster]; cl != nil {
					server = cl.Server
				}
			}
			return mcp.NewToolResultText(fmt.Sprintf("Context: %s\nServer:  %s\n", current, server)), nil
		},
	)
}
