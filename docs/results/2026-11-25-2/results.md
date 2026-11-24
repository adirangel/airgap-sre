# Eval results — 2026-11-25

| | |
| --- | --- |
| Started | 2026-11-25 00:23 IST |
| Model | `qwen3.5:4b` (Ollama) |
| Agent | `sre-agent` on kagent 0.10.7 |
| Tools | `k8s_get_resources`, `k8s_describe_resource`, `k8s_get_resource_yaml`, `k8s_get_pod_logs`, `k8s_get_events`, `k8s_get_available_api_resources` |
| Commit | `5721347` |

| # | Scenario | Expected category | Agent's category | Resources named | In JSON block | Points | Result | Time | Tool calls |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| 1 | [`crashloop-bad-config`](transcripts/01-crashloop-bad-config.md) | `bad_config` | `bad_config` | 3/3 | 3/3 | 1.00 | PASS | 14s | 5 |
| 2 | [`oomkilled`](transcripts/02-oomkilled.md) | `oom_killed` | `oom_killed` | 1/1 | 1/1 | 1.00 | PASS | 20s | 8 |
| 3 | [`imagepull-not-mirrored`](transcripts/03-imagepull-not-mirrored.md) | `image_pull_failure` | `image_pull_failure` | 2/2 | 2/2 | 1.00 | PASS | 16s | 5 |
| 4 | [`pending-unschedulable`](transcripts/04-pending-unschedulable.md) | `unschedulable` | `unschedulable` | 2/2 | 2/2 | 1.00 | PASS | 14s | 5 |
| 5 | [`service-selector-mismatch`](transcripts/05-service-selector-mismatch.md) | `service_selector_mismatch` | `service_selector_mismatch` | 2/2 | 2/2 | 1.00 | PASS | 11s | 4 |

**5/5 passed**, 5.00/5 points.

Scoring is deterministic (no LLM judge): the category is read from the JSON block that ends the
agent's answer and must match exactly; each expected resource name must appear in the answer as a whole
token. Points = 0.5 for the category + 0.5 × (resources named / expected). PASS needs both in full.
"In JSON block" is stricter and informational only: how many of those names the agent also put inside
its final JSON block. A PASS with a low number there leaned on a name mentioned only in passing.
