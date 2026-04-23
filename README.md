# k8s-mcp-server

An MCP server that exposes Kubernetes tools to MCP-capable clients such as Claude Desktop, Cursor, or other MCP hosts.

## What It Does

This server lets an LLM use Kubernetes operations through MCP tools instead of shelling out to `kubectl`.

It currently exposes 16 tools:

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

## Requirements

- Go `1.25.5` or newer
- Access to a valid kubeconfig

By default, the server uses `~/.kube/config`. You can override that with `--kubeconfig`.

## Build

```bash
make build
```

This produces:

```bash
./k8s-mcp-server
```

## Run

### Stdio Mode

Use `stdio` mode when an MCP client launches the process directly, for example Claude Desktop.

```bash
go run main.go
```

or:

```bash
make run-stdio
```

### SSE Mode

Use SSE mode when an MCP client connects over HTTP.

```bash
go run main.go --port 8080
```

or:

```bash
make run-sse
```

## Claude Desktop Setup

Build the binary first:

```bash
make build
```

Then add this to your Claude Desktop config on macOS:

`~/Library/Application Support/Claude/claude_desktop_config.json`

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

If you want to use a specific kubeconfig:

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

Restart Claude Desktop after updating the config.

## Examples

Once connected from an MCP client, you can ask for things like:

- `List my Kubernetes contexts`
- `Show pods in the default namespace`
- `Get logs for pod nginx-123 in namespace default`
- `List warning events in kube-system`

## Tests

Run the Go test suite with:

```bash
go test ./...
```

If you use Ginkgo and have the CLI installed:

```bash
ginkgo -r
```

## Notes

- `stdio` is the normal mode for local desktop clients.
- `SSE` is useful when the MCP server runs separately from the client.
- In `stdio` mode, logs are written to `stderr`.
