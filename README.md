# k8s-mcp-server

An MCP server that lets Codex, Claude Desktop, and other MCP clients use Kubernetes tools directly.

## Quick Start For Users

If you want Codex or Claude Desktop to help inspect or operate your cluster, this is the setup flow:

1. Build the binary.
2. Register it in your MCP client.
3. Restart the client if needed.
4. Ask Kubernetes questions in natural language.

You do not manually call MCP methods yourself. The client discovers the available tools automatically and chooses which one to use based on your prompt.

## Requirements

- Go `1.25.5` or newer
- Access to a valid kubeconfig

By default, the server uses `~/.kube/config`.

## 1. Build The Server

```bash
make build
```

This creates:

```bash
./k8s-mcp-server
```

## 2. Connect It To An LLM Client

This server is designed for MCP clients that launch local tools in `stdio` mode.

### Codex

Add this to your Codex config:

`~/.codex/config.toml`

```toml
[mcp_servers.k8s]
command = "go"
args = ["run", "/absolute/path/to/k8s-mcp-server"]
startup_timeout_sec = 30
tool_timeout_sec = 120
```

If you prefer running a built binary instead:

```toml
[mcp_servers.k8s]
command = "/absolute/path/to/k8s-mcp-server/k8s-mcp-server"
args = []
startup_timeout_sec = 15
tool_timeout_sec = 120
```

If you want to force a specific kubeconfig, pass the flag as args. For example:

```toml
[mcp_servers.k8s]
command = "/absolute/path/to/k8s-mcp-server/k8s-mcp-server"
args = ["--kubeconfig", "/Users/you/.kube/config"]
startup_timeout_sec = 15
tool_timeout_sec = 120
```

Then start a new Codex session in the project and confirm the MCP server is available:

```bash
codex mcp list
```

### Claude Desktop

Claude Desktop should launch this server in `stdio` mode. You do not need to run `go run main.go` yourself for that setup.

Edit the Claude Desktop config on macOS:

`~/Library/Application Support/Claude/claude_desktop_config.json`

Add:

```json
{
  "mcpServers": {
    "k8s": {
      "command": "/Users/ytan/workstation/yulintan/k8s-mcp-server/k8s-mcp-server",
      "args": []
    }
  }
}
```

If you want to force a specific kubeconfig:

```json
{
  "mcpServers": {
    "k8s": {
      "command": "/Users/ytan/workstation/yulintan/k8s-mcp-server/k8s-mcp-server",
      "args": ["--kubeconfig", "/Users/ytan/.kube/config"]
    }
  }
}
```

Then restart Claude Desktop.

## 3. Start Using It

Once Codex or Claude Desktop sees the server, you can ask things like:

- `List my Kubernetes contexts`
- `Show pods in the default namespace`
- `Get logs for pod nginx-123 in namespace default`
- `List warning events in kube-system`
- `Show me all nodes and their status`
- `Get the deployment my-api in namespace prod`

The client will inspect the tool list exposed by this server and choose the right tool automatically.

## Demo Prompts

Here are practical examples of what you can ask once this server is connected.

### Cluster Discovery

Use prompts like:

- `What Kubernetes context am I currently using?`
- `List all namespaces in my current cluster`
- `Show me all nodes and whether they are ready`

Typical outcomes:

- Codex or Claude shows the current kubeconfig context
- Codex or Claude lists namespaces with age and status
- Codex or Claude summarizes node readiness, roles, version, and internal IP

### Pod Investigation

Use prompts like:

- `List pods in namespace default`
- `Show me details for pod api-7c9d8d4b6f-xyz12 in namespace prod`
- `Get the last 100 lines of logs from pod nginx-123 in namespace default`
- `Show me warning events in kube-system`

Typical outcomes:

- Codex or Claude lists matching pods
- Codex or Claude inspects a pod's status, node, IP, labels, containers, and conditions
- Codex or Claude fetches logs from the selected container
- Codex or Claude surfaces recent warning events for troubleshooting

### Resource Lookup

Use prompts like:

- `List deployments in namespace prod`
- `Get deployment my-api in namespace prod`
- `List configmaps in kube-system`

Typical outcomes:

- Codex or Claude maps your request to `apiVersion` and `kind`
- Codex or Claude lists matching resources
- Codex or Claude returns the full JSON for a specific resource when needed

### Exec And Debug Workflows

Use prompts like:

- `Run hostname inside pod api-0 in namespace default`
- `Create a debug pod in namespace default using busybox`
- `Delete pod debug-abc123 from namespace default`

Typical outcomes:

- Codex or Claude executes a command inside a pod and returns stdout and stderr
- Codex or Claude creates a temporary debug pod for investigation
- Codex or Claude deletes the pod when requested

### Bulk Operations

Use prompts like:

- `List pods in default and kube-system namespaces`
- `Run hostname across these pods: api-0 in default and worker-0 in jobs`
- `Create debug pods in default, kube-system, and monitoring`

Typical outcomes:

- Codex or Claude uses the bulk tools instead of making many small sequential calls
- Results are grouped by namespace, context, or target pod

## What Users Can Ask For

This server exposes Kubernetes tools for:

- kubeconfig context discovery
- namespace listing
- node inspection
- event listing
- pod listing and detail lookup
- pod logs
- pod exec
- debug pod creation and deletion
- bulk pod operations
- generic Kubernetes resource lookup by `apiVersion` and `kind`

Current tool names:

- `k8s_contexts_list`
- `k8s_context_current`
- `k8s_namespaces_list`
- `k8s_nodes_list`
- `k8s_events_list`
- `k8s_pods_list`
- `k8s_pods_get`
- `k8s_pods_logs`
- `k8s_pods_exec`
- `k8s_pods_run`
- `k8s_pods_delete`
- `k8s_pods_list_bulk`
- `k8s_pods_exec_bulk`
- `k8s_debug_pods_create_bulk`
- `k8s_resources_list`
- `k8s_resources_get`

## Other Run Modes

If you are not using a local `stdio` MCP client and need an HTTP endpoint instead, run SSE mode:

```bash
go run main.go --port 8080
```

or:

```bash
make run-sse
```

For local MCP clients that spawn the process directly, the default is `stdio` mode:

```bash
go run main.go
```

or:

```bash
make run-stdio
```

## Troubleshooting

- If Codex does not see the server, confirm the entry in `~/.codex/config.toml` is correct and start a new Codex session.
- If Claude Desktop does not see the server, confirm the binary path in `claude_desktop_config.json` is correct.
- If the client starts the server but tools fail, verify your kubeconfig works and points at the cluster you expect.
- If `ginkgo` is not found, ensure `$(go env GOPATH)/bin` is on your `PATH`.
- In `stdio` mode, logs are written to `stderr`.

## Development

Run the Go test suite with:

```bash
go test ./...
```

If you use Ginkgo and have the CLI installed:

```bash
ginkgo -r
```
