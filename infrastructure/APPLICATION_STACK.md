# Application stack inventory

What the agent will eventually operate on, and where its tools come from.
"Reuse, don't rebuild": if an MCP server already exists for a system, we use
it. `docs/gaps.md` (written after the M3 eval) decides what custom MCP server,
if any, we build next.

## Lab (this run, M0–M3)

| Component | Role | Tooling for the agent | Status |
| --- | --- | --- | --- |
| kind (Kubernetes in Docker) | Lab cluster | kagent built-in Kubernetes tools | M1 |
| `registry` (CNCF Distribution) | Lab image registry; the only place the cluster may pull from | none yet (candidate gap) | M1 |
| kagent 0.10.7 | Agent runtime (controller, tool server, UI, PostgreSQL, one pod per agent) | — | M3 |
| Ollama on the Windows host | Model for the M3 baseline (`qwen3.5:4b`). Pods reach it at `host.docker.internal:11434`. No cloud API is called | — | M3 |

## Target environment (after M3)

| Component | Role | Tooling for the agent | Status |
| --- | --- | --- | --- |
| RKE2 | Production-like Kubernetes | kagent built-in Kubernetes tools | later |
| Harbor | Internal registry | Reuse [`oreondev/harbor-mcp`](https://github.com/oreondev/harbor-mcp) — do not build. Read-only MCP server (projects, artifacts, vulnerabilities, robots, replication); image `ghcr.io/oreondev/harbor-mcp`; stdio or streamable HTTP. Checked 2026-11-24: v0.2.0, a very new project, so review it before giving it registry credentials | later |
| Jenkins | CI/CD | Reuse the Jenkins [`mcp-server` plugin](https://plugins.jenkins.io/mcp-server/) ([source](https://github.com/jenkinsci/mcp-server-plugin)) — do not build. Jenkins itself becomes the MCP server (`/mcp-server/mcp`); auth is a Jenkins user + API token | later |
| Ollama (Windows host, owns the GPU) | Local LLM; pods reach it at `host.docker.internal:11434` | — | already used in M3 |

No GPU passthrough into kind is attempted.
