package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	. "github.com/onsi/gomega"
	"github.com/yulintan/k8s-mcp-server/internal/k8s"
	"github.com/yulintan/k8s-mcp-server/internal/k8s/k8sfakes"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/discovery"
	discoveryfake "k8s.io/client-go/discovery/fake"
	"k8s.io/client-go/dynamic"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes"
	kubernetesfake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"
	clientgotesting "k8s.io/client-go/testing"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

type fakeClientManagerConfig struct {
	typedClient     kubernetes.Interface
	dynamicClient   dynamic.Interface
	discoveryClient discovery.DiscoveryInterface
	restConfig      *rest.Config
	rawConfig       clientcmdapi.Config
	currentCtx      string
	bulkListPodsFn  func(context.Context, []k8s.BulkTarget, string, int) []k8s.PodListResult
	bulkExecFn      func(context.Context, []k8s.ExecTarget, []string, int) []k8s.BulkExecResult
	bulkDebugPodsFn func(context.Context, []k8s.DebugPodTarget, string, []string, int) []k8s.DebugPodResult
}

func newFakeClientManager(cfg fakeClientManagerConfig) *k8sfakes.FakeClientManager {
	typedClient := cfg.typedClient
	if typedClient == nil {
		typedClient = kubernetesfake.NewSimpleClientset()
	}
	dynamicClient := cfg.dynamicClient
	if dynamicClient == nil {
		dynamicClient = dynamicfake.NewSimpleDynamicClient(runtime.NewScheme())
	}
	discoveryClient := cfg.discoveryClient
	if discoveryClient == nil {
		discoveryClient = &discoveryfake.FakeDiscovery{Fake: &clientgotesting.Fake{}}
	}
	restConfig := cfg.restConfig
	if restConfig == nil {
		restConfig = &rest.Config{}
	}
	currentCtx := cfg.currentCtx
	if currentCtx == "" {
		currentCtx = cfg.rawConfig.CurrentContext
	}

	fake := &k8sfakes.FakeClientManager{}
	fake.GetClientReturns(typedClient, nil)
	fake.GetDynamicClientReturns(dynamicClient, nil)
	fake.GetDiscoveryClientReturns(discoveryClient, nil)
	fake.GetRESTConfigReturns(restConfig, nil)
	fake.RawConfigReturns(cfg.rawConfig, nil)
	if currentCtx == "" {
		fake.CurrentContextReturns("", fmt.Errorf("no current context set in kubeconfig"))
	} else {
		fake.CurrentContextReturns(currentCtx, nil)
	}
	if cfg.bulkListPodsFn != nil {
		fake.BulkListPodsCalls(cfg.bulkListPodsFn)
	}
	if cfg.bulkExecFn != nil {
		fake.BulkExecCalls(cfg.bulkExecFn)
	}
	if cfg.bulkDebugPodsFn != nil {
		fake.BulkCreateDebugPodsCalls(cfg.bulkDebugPodsFn)
	}
	return fake
}

func newTestServer(cm k8s.ClientManager) *server.MCPServer {
	srv := server.NewMCPServer("test-k8s", "test", server.WithToolCapabilities(true))
	RegisterAllTools(srv, cm)
	return srv
}

func callToolText(srv *server.MCPServer, toolName string, args map[string]any) string {
	reqBytes, err := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/call",
		"params": map[string]any{
			"name":      toolName,
			"arguments": args,
		},
	})
	Expect(err).NotTo(HaveOccurred())

	payload := decodeResponsePayload(srv.HandleMessage(context.Background(), reqBytes))
	result, ok := payload["result"].(map[string]any)
	Expect(ok).To(BeTrue())

	contentItems, ok := result["content"].([]any)
	Expect(ok).To(BeTrue())
	Expect(contentItems).NotTo(BeEmpty())

	textContent, ok := contentItems[0].(map[string]any)
	Expect(ok).To(BeTrue())
	text, ok := textContent["text"].(string)
	Expect(ok).To(BeTrue())
	return text
}

func decodeResponsePayload(resp mcp.JSONRPCMessage) map[string]any {
	Expect(resp).NotTo(BeNil())

	data, err := json.Marshal(resp)
	Expect(err).NotTo(HaveOccurred())

	var payload map[string]any
	Expect(json.Unmarshal(data, &payload)).To(Succeed())
	return payload
}
