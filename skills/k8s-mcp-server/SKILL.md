---
name: k8s_mcp_server
description: Use this skill when the user wants Kubernetes inspection, debugging, logs, events, pod exec, or cluster reachability checks through this workspace's k8s MCP server, or when they want OpenClaw connected to this server.
---

# k8s MCP Server

This workspace contains a local MCP server for Kubernetes operations.

Use this skill when:

- the user wants to inspect clusters, namespaces, nodes, pods, logs, events, or Kubernetes resources
- the user wants to inspect common resources such as deployments, services, ingresses, jobs, or PVCs
- the user needs to discover available API resources before querying CRDs or unfamiliar resources
- the user wants to run commands in pods or create short-lived debug pods
- the user wants OpenClaw connected to this repository's Kubernetes MCP server

## Expected server name

Register the MCP server in OpenClaw as `k8s` unless the user asks for a different name.

## Preferred registration

Prefer the built binary if it exists:

```bash
openclaw mcp set k8s '{"command":"{baseDir}/../../k8s-mcp-server","args":[]}'
```

If the binary does not exist yet, build it from the workspace root:

```bash
make build
```

If the user prefers a development setup, use `go run` instead:

```bash
openclaw mcp set k8s '{"command":"go","args":["run","{baseDir}/../.."]}'
```

If the user needs a specific kubeconfig, pass it in `args`:

```bash
openclaw mcp set k8s '{"command":"{baseDir}/../../k8s-mcp-server","args":["--kubeconfig","/absolute/path/to/config"]}'
```

## Verification

After registration, verify the saved definition:

```bash
openclaw mcp list
openclaw mcp show k8s
```

Do not claim the server is actually usable just because `openclaw mcp set` succeeded. That only saves config. Prefer a real Kubernetes request to confirm end-to-end behavior.

## Operating guidance

- Use the `k8s_*` MCP tools directly when they are available.
- For cluster or namespace discovery, start with contexts, namespaces, nodes, or pods before jumping into resource-specific queries.
- For common Kubernetes objects, prefer the high-level deployment, service, ingress, job, and PVC tools over the generic resource tools.
- For unfamiliar resources or CRDs, use `k8s_api_resources_list` first, then call the generic resource tools with the discovered `apiVersion` and `kind`.
- For network checks from inside a cluster, create a short-lived debug pod, run the connectivity test, then delete the pod.
- For destructive actions, require clear user intent.
- If there are multiple kubeconfig contexts, be explicit about which context you are using.

## Example user requests

- `List my Kubernetes contexts`
- `Show pods in namespace default`
- `Get warning events in kube-system`
- `List ingresses in namespace prod`
- `List API resources available in this cluster`
- `Run hostname in pod api-0 in namespace prod`
- `Check whether this cluster can reach example.internal on port 443`
