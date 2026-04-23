# AI Workflow

## Project

`k8s-mcp-server` is a Kubernetes-focused MCP server that exposes cluster operations as MCP tools over:

- `stdio` for local MCP clients such as Claude Code / desktop tooling
- `SSE` for HTTP-based clients and transport experiments

The current implementation includes:

- kubeconfig context discovery
- namespace / node / event inspection
- pod listing, lookup, logs, exec, create, delete
- bulk pod operations across contexts / namespaces
- generic Kubernetes resource lookup through the dynamic client

## Why I Chose This Project

I chose this project because it is directly relevant to DevOps / SRE-style work:

- it automates operational access to Kubernetes
- it requires clear API and transport design
- it touches concurrency, system boundaries, and failure handling
- it is a good fit for discussing MCP, JSON-RPC, SSE, and Kubernetes client design

It also gives me a concrete way to demonstrate AI-assisted engineering on top of a real infrastructure-oriented problem rather than a toy demo.

## Development Environment

Primary development interface:

- Claude Code

Main languages / tools:

- Go
- `client-go`
- `mcp-go`
- Ginkgo / Gomega
- `counterfeiter` for generated test doubles

## How I Used AI During Development

I used Claude Code as the primary interface for:

- codebase exploration and summarization
- drafting new MCP tool handlers and helper functions
- refactoring package structure
- improving test coverage
- generating and reviewing fake-based test scaffolding
- validating request / response flows for SSE-based MCP communication
- tightening documentation and interview-facing explanations

I used AI as an implementation partner, not as an autopilot. I reviewed the generated code, tested it, and adjusted it when needed.

## Typical Workflow

My workflow was generally:

1. Inspect the existing code and identify the next narrow change.
2. Ask Claude Code to propose or implement a targeted change.
3. Review the generated code for correctness, API fit, and testability.
4. Run tests and inspect failures.
5. Adjust the implementation or narrow the scope when the generated approach was too broad.
6. Commit incremental, reviewable steps rather than one large final dump.

## How I Validated AI-Generated Code

I validated generated code by:

- reading the affected files end to end
- checking that changes matched existing package boundaries and naming
- verifying behavior with `go test ./...`
- testing SSE / MCP flows manually with `curl`
- checking whether fake clients were realistic enough for the unit under test
- revising brittle or over-coupled tests when they validated formatting artifacts instead of behavior

Examples of corrections made during development:

- replacing a handwritten test double with a generated `counterfeiter` fake at the `ClientManager` seam
- reorganizing tool tests from one combined file into feature-scoped test files
- adjusting lower-level bulk tests to avoid invalid assumptions about `client-go` fake exec transport
- refining assertions when initial generated tests were too coupled to fixed-width output formatting

## How AI Helped Most

AI was most useful for:

- accelerating boilerplate-heavy Go changes
- generating first-pass tests and fake wiring
- comparing transport / protocol concepts quickly while I refined the design
- helping turn implementation work into clearer documentation and interview-ready explanations
