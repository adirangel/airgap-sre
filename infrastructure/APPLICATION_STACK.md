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
| kagent | Agent runtime (controller, tool server, agent pods) | — | M3 |
| Cloud LLM API | Model for the M3 baseline (egress stays open for this) | — | M3 |

## Target environment (after M3)

| Component | Role | Tooling for the agent | Status |
| --- | --- | --- | --- |
| RKE2 | Production-like Kubernetes | kagent built-in Kubernetes tools | later |
| Harbor | Internal registry | Reuse [`oreondev/harbor-mcp`](https://github.com/oreondev/harbor-mcp) — do not build | later |
| Jenkins | CI/CD | Reuse the Jenkins [`mcp-server` plugin](https://plugins.jenkins.io/mcp-server/) — do not build | later |
| Ollama (Windows host, owns the GPU) | Local LLM; pods reach it at `host.docker.internal:11434` | — | later |

No GPU passthrough into kind is attempted.
