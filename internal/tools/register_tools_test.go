package tools

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("tool registration", func() {
	It("lists the expected tools through MCP", func() {
		srv := newTestServer(newFakeClientManager(fakeClientManagerConfig{}))

		resp := srv.HandleMessage(context.Background(), []byte(`{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}`))

		payload := decodeResponsePayload(resp)
		toolsPayload, ok := payload["result"].(map[string]any)
		Expect(ok).To(BeTrue())

		names := make([]string, 0)
		for _, item := range toolsPayload["tools"].([]any) {
			tool := item.(map[string]any)
			names = append(names, tool["name"].(string))
		}

		Expect(names).To(ContainElements(
			"k8s_contexts_list",
			"k8s_context_current",
			"k8s_namespaces_list",
			"k8s_nodes_list",
			"k8s_events_list",
			"k8s_pods_list",
			"k8s_pods_get",
			"k8s_pods_logs",
			"k8s_pods_exec",
			"k8s_pods_run",
			"k8s_pods_delete",
			"k8s_pods_list_bulk",
			"k8s_pods_exec_bulk",
			"k8s_debug_pods_create_bulk",
			"k8s_resources_list",
			"k8s_resources_get",
			"k8s_api_resources_list",
			"k8s_deployments_list",
			"k8s_deployments_get",
			"k8s_services_list",
			"k8s_services_get",
			"k8s_ingresses_list",
			"k8s_ingresses_get",
			"k8s_jobs_list",
			"k8s_jobs_get",
			"k8s_pvcs_list",
			"k8s_pvcs_get",
		))
	})
})
