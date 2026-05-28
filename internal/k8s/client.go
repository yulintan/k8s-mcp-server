package k8s

import (
	"context"
	"fmt"
	"sync"

	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 -o k8sfakes/fake_client_manager.go --fake-name FakeClientManager . ClientManager

// ClientManager provides Kubernetes clients per kubeconfig context.
type ClientManager interface {
	GetClient(contextName string) (kubernetes.Interface, error)
	GetDynamicClient(contextName string) (dynamic.Interface, error)
	GetDiscoveryClient(contextName string) (discovery.DiscoveryInterface, error)
	GetRESTConfig(contextName string) (*rest.Config, error)
	RawConfig() (clientcmdapi.Config, error)
	CurrentContext() (string, error)

	// Bulk concurrent operations.
	BulkListPods(ctx context.Context, targets []BulkTarget, labelSelector string, maxConcurrency int) []PodListResult
	BulkExec(ctx context.Context, targets []ExecTarget, command []string, maxConcurrency int) []BulkExecResult
	BulkCreateDebugPods(ctx context.Context, targets []DebugPodTarget, image string, command []string, maxConcurrency int) []DebugPodResult
}

type clientEntry struct {
	typed   kubernetes.Interface
	dynamic dynamic.Interface
	disco   discovery.DiscoveryInterface
	rest    *rest.Config
}

type clientManager struct {
	mu           sync.RWMutex
	cache        map[string]*clientEntry
	loadingRules *clientcmd.ClientConfigLoadingRules
}

func NewClientManager(kubeconfigPath string) ClientManager {
	rules := clientcmd.NewDefaultClientConfigLoadingRules()
	if kubeconfigPath != "" {
		rules.ExplicitPath = kubeconfigPath
	}
	return &clientManager{
		cache:        make(map[string]*clientEntry),
		loadingRules: rules,
	}
}

func (m *clientManager) resolveContext(contextName string) (string, error) {
	if contextName != "" {
		return contextName, nil
	}
	return m.CurrentContext()
}

func (m *clientManager) getOrCreate(contextName string) (*clientEntry, error) {
	resolved, err := m.resolveContext(contextName)
	if err != nil {
		return nil, err
	}

	m.mu.RLock()
	entry, ok := m.cache[resolved]
	m.mu.RUnlock()
	if ok {
		return entry, nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	// Double-check after acquiring write lock.
	if entry, ok = m.cache[resolved]; ok {
		return entry, nil
	}

	overrides := &clientcmd.ConfigOverrides{CurrentContext: resolved}
	cfg := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(m.loadingRules, overrides)
	restCfg, err := cfg.ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("building REST config for context %q: %w", resolved, err)
	}

	typed, err := kubernetes.NewForConfig(restCfg)
	if err != nil {
		return nil, fmt.Errorf("creating typed client for context %q: %w", resolved, err)
	}
	dyn, err := dynamic.NewForConfig(restCfg)
	if err != nil {
		return nil, fmt.Errorf("creating dynamic client for context %q: %w", resolved, err)
	}
	disco, err := discovery.NewDiscoveryClientForConfig(restCfg)
	if err != nil {
		return nil, fmt.Errorf("creating discovery client for context %q: %w", resolved, err)
	}

	entry = &clientEntry{typed: typed, dynamic: dyn, disco: disco, rest: restCfg}
	m.cache[resolved] = entry
	return entry, nil
}

func (m *clientManager) GetClient(contextName string) (kubernetes.Interface, error) {
	e, err := m.getOrCreate(contextName)
	if err != nil {
		return nil, err
	}
	return e.typed, nil
}

func (m *clientManager) GetDynamicClient(contextName string) (dynamic.Interface, error) {
	e, err := m.getOrCreate(contextName)
	if err != nil {
		return nil, err
	}
	return e.dynamic, nil
}

func (m *clientManager) GetDiscoveryClient(contextName string) (discovery.DiscoveryInterface, error) {
	e, err := m.getOrCreate(contextName)
	if err != nil {
		return nil, err
	}
	return e.disco, nil
}

func (m *clientManager) GetRESTConfig(contextName string) (*rest.Config, error) {
	e, err := m.getOrCreate(contextName)
	if err != nil {
		return nil, err
	}
	return e.rest, nil
}

func (m *clientManager) RawConfig() (clientcmdapi.Config, error) {
	cfg := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(m.loadingRules, &clientcmd.ConfigOverrides{})
	return cfg.RawConfig()
}

func (m *clientManager) CurrentContext() (string, error) {
	raw, err := m.RawConfig()
	if err != nil {
		return "", err
	}
	if raw.CurrentContext == "" {
		return "", fmt.Errorf("no current context set in kubeconfig")
	}
	return raw.CurrentContext, nil
}
