package tools

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

var _ = Describe("config tools", func() {
	It("renders sorted contexts and marks the current one", func() {
		srv := newTestServer(newFakeClientManager(fakeClientManagerConfig{
			rawConfig: clientcmdapi.Config{
				CurrentContext: "prod",
				Contexts: map[string]*clientcmdapi.Context{
					"staging": {Cluster: "cluster-staging"},
					"prod":    {Cluster: "cluster-prod"},
					"dev":     {Cluster: "cluster-dev"},
				},
				Clusters: map[string]*clientcmdapi.Cluster{
					"cluster-dev":     {Server: "https://dev.example.com"},
					"cluster-prod":    {Server: "https://prod.example.com"},
					"cluster-staging": {Server: "https://staging.example.com"},
				},
			},
		}))

		result := callToolText(srv, "k8s_contexts_list", map[string]any{})

		Expect(result).To(ContainSubstring("*    prod"))
		Expect(result).To(ContainSubstring("dev"))
		Expect(result).To(ContainSubstring("staging"))
		Expect(result).To(ContainSubstring("https://prod.example.com"))
	})
})
