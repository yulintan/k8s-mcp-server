package tools

import (
	"github.com/mark3labs/mcp-go/server"
	"github.com/yulintan/k8s-mcp-server/internal/k8s"
)

// RegisterAllTools wires every MCP tool to the server.
func RegisterAllTools(s *server.MCPServer, cm k8s.ClientManager) {
	registerConfigTools(s, cm)
	registerClusterTools(s, cm)
	registerPodTools(s, cm)
	registerBulkTools(s, cm)
	registerResourceTools(s, cm)
}
