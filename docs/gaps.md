# Gaps after the M3 baseline

What the baseline agent could not do, and which tool would have helped. This
file decides what, if anything, we build next.

Baseline: agent `sre-agent` on kagent 0.10.7, model `qwen3.5:4b` (4.7B, local,
Ollama), six read-only built-in Kubernetes tools. Two runs on 2026-11-25:

| Run | Result | Notes |
| --- | --- | --- |
| [results/2026-11-25](results/2026-11-25/results.md) | 5/5 | Tool server could not read nodes (gap 1) |
| [results/2026-11-25-2](results/2026-11-25-2/results.md) | 5/5 | After the RBAC fix |

## Read this first: 5/5 does not mean "no gaps"

The deterministic scorer found **no misses**. That says more about the
benchmark than about the agent:

- The five scenarios are single-fault textbook cases in one namespace. A
  4.7B local model solves all of them in 4–11 tool calls. The set cannot
  tell a good agent from a mediocre one, or a small model from a large one.
- The scorer checks the category and that names appear. It does not check
  how deep or how correct the explanation is.

So the gaps below come from reading the transcripts by hand, which is what
they are kept for.

## Gaps

| # | Where the agent fell short | Evidence | What was missing | Build it? |
| --- | --- | --- | --- | --- |
| 1 | **Could not read nodes.** On the Pending-pod scenario it tried `get node` six times; every call failed. It still answered "no node has the label", but only by repeating the scheduler event, not by checking the node. | [run 1, scenario 4](results/2026-11-25/transcripts/04-pending-unschedulable.md): 11 tool calls, 7 failed | RBAC. The chart's read-only role for the tool server has no `nodes`, `endpointslices`, quotas or storage objects. | **Fixed** in `infrastructure/helm/kagent-values.yaml` (`extraObjects`). [Run 2](results/2026-11-25-2/transcripts/04-pending-unschedulable.md): 5 calls, node described, label checked. |
| 2 | **Tool errors are opaque.** The agent saw `Tool execution failed ... exit status 1`. The real message (`nodes is forbidden: ... cannot list resource "nodes"`) was only in the tool server's log. With it, the agent would have stopped after one call and could have reported the permission problem. | Same transcript; `kubectl logs -n kagent deploy/kagent-tools` | The built-in tool server (`kagent-tools` 0.3.2) drops kubectl's stderr. | No. Report upstream to `kagent-dev/tools`. Not a reason for a custom server. |
| 3 | **Image pull: found the symptom, not the cause.** Both runs say the image "is not available in the registry (or the registry is unreachable)". True, but the real cause is: the image exists upstream, the cluster is air-gapped, and nobody mirrored it. The agent cannot know that. In an air-gapped site this is the most common failure, and the fix ("mirror it") depends on exactly this distinction. | [run 2, scenario 3](results/2026-11-25-2/transcripts/03-imagepull-not-mirrored.md) | A registry view: which repositories and tags the internal registry has, and whether image X was mirrored. Also useful: the list of allowed images (`images.yaml`). | **Yes, this is the first candidate** — see "What to build next". |
| 4 | **OOM: no numbers.** The agent sees `OOMKilled` and the 32Mi limit but cannot say how much memory the container used or what limit would be enough. In run 2 it also explained the workload wrongly ("writes to stdout"). | [run 2, scenario 2](results/2026-11-25-2/transcripts/02-oomkilled.md) | Metrics. The lab has no metrics-server or Prometheus. | No. kagent's tool server already has Prometheus tools (`enabledTools`). Mirror a metrics stack and switch them on. Reuse, don't build. |
| 5 | **Noisy tool output.** `k8s_get_events` returns full JSON (about 4 KB for a handful of events), and the agent sometimes fetches the same YAML twice. It is harmless here, but on a real namespace it will fill the context of a small model. | Any transcript; 12k–48k tokens per scenario | A compact events view (reason, object, message, count, age). | Not yet. Measure it on harder scenarios first. |

## Limits of this benchmark

These are gaps in the measuring stick, not in the agent:

- **Too easy** (see above). Needed next: faults with a misleading symptom,
  two-hop causes (Service → Deployment → ConfigMap → Secret), faults in
  another namespace, NetworkPolicy, probes, quotas, PVCs. The category list
  in the prompt already covers them.
- **The symptom names the namespace**, which removes the search step.
- **The prompt defines every category**, close to the wording of the faults.
- **Name matching is lenient**: a name counts if it appears anywhere in the
  answer. The report now also shows a stricter "In JSON block" count.
- **Two runs only.** Temperature is 0 and the seed is fixed, yet the two runs
  used different tool calls. Run-to-run variance is not measured.
- **One model.** The plan was a cloud model for the baseline; the owner chose
  a local one. There is no stronger model to compare against yet.

## What to build next

1. **Harder scenarios first.** With 5/5 on a 4.7B model, no tool can show an
   improvement on this set. Without this step, every later decision is a
   guess.
2. **A read-only registry tool (gap 3)** is the one place where a custom MCP
   server is justified so far — with two checks before writing any code:
   - If the target registry is **Harbor**: do not build. Use
     `oreondev/harbor-mcp` (see `infrastructure/APPLICATION_STACK.md`).
   - For the lab's plain `registry` (OCI Distribution API): first search for
     an existing MCP server for the Distribution API. Only if there is none,
     build a small read-only one: list repositories and tags, answer "is
     image X present?", and compare the images used by workloads with the
     allowed list.
3. **Metrics (gap 4) and upstream bug report (gap 2)**: configuration and
   reuse only. No custom server.
4. **Jenkins**: nothing in this baseline touches CI. No evidence yet either
   way; the `mcp-server` plugin stays the plan.
