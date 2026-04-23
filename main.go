package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/mark3labs/mcp-go/server"
	"github.com/yulintan/k8s-mcp-server/internal/k8s"
	"github.com/yulintan/k8s-mcp-server/internal/tools"
)

const version = "0.1.0"

func main() {
	port := flag.Int("port", 0, "HTTP/SSE port. 0 = stdio mode (default, for Claude Desktop/Cursor/VS Code).")
	kubeconfigPath := flag.String("kubeconfig", "", "Path to kubeconfig file. Default: ~/.kube/config.")
	flag.Parse()

	cm := k8s.NewClientManager(*kubeconfigPath)

	s := server.NewMCPServer(
		"k8s-mcp-server",
		version,
		server.WithToolCapabilities(true),
		server.WithRecovery(),
	)

	tools.RegisterAllTools(s, cm)

	if *port == 0 {
		// Stdio mode: used by Claude Desktop, Cursor, VS Code Continue, etc.
		log.SetOutput(os.Stderr)
		if err := server.ServeStdio(s); err != nil {
			fmt.Fprintf(os.Stderr, "stdio server error: %v\n", err)
			os.Exit(1)
		}
	} else {
		// HTTP/SSE mode: for web-based clients.
		addr := fmt.Sprintf(":%d", *port)
		baseURL := fmt.Sprintf("http://localhost:%d", *port)
		sseServer := server.NewSSEServer(s, server.WithBaseURL(baseURL))
		log.Printf("k8s-mcp-server v%s listening on %s (SSE mode)", version, addr)
		if err := sseServer.Start(addr); err != nil {
			fmt.Fprintf(os.Stderr, "SSE server error: %v\n", err)
			os.Exit(1)
		}
	}
}
