# Failure scenarios

Each directory is one reproducible failure. `labctl scenario` runs them and
`labctl eval` scores an agent against them.

```
<id>/
  scenario.yaml   what it is, where it runs, how to prove the failure is present
  manifests.yaml  the broken workload (applied into the scenario namespace)
  expected.yaml   the correct diagnosis, used by the deterministic scorer
```

## scenario.yaml

| Field | Meaning |
| --- | --- |
| `id` | Must equal the directory name |
| `order` | Position in lists and reports (1..n) |
| `title`, `description` | For humans. The agent never sees these |
| `namespace` | Must start with `scn-`. Must **not** contain words from the id or the category, because the agent can see it |
| `symptom` | The only thing the agent is told: what a user would report. Must name the namespace and must not give the cause away |
| `unmirroredImages` | Images the manifests use on purpose although they are not in `images.yaml` |
| `verify.timeoutSeconds` | How long `scenario verify` waits for the failure to appear |
| `verify.checks` | Conditions that must all hold at the same time |

Check types:

| `type` | Arguments | True when |
| --- | --- | --- |
| `containerWaiting` | `selector`, `reasons` | a container of a matching pod is waiting with one of the reasons |
| `containerTerminated` | `selector`, `reasons` | a container's current or last state is terminated with one of the reasons |
| `podUnschedulable` | `selector` | a matching pod has `PodScheduled=False` with reason `Unschedulable` |
| `podsReady` | `selector` | at least one pod matches and all matching pods are Ready |
| `serviceNoEndpoints` | `service` | the Service has no endpoint addresses |

## expected.yaml

| Field | Meaning |
| --- | --- |
| `rootCauseCategory` | One value from the closed list in `backend/internal/scenario/scenario.go` (`Categories`) |
| `mustName` | Things a correct diagnosis has to name. `kind` is for the report; the scorer looks for `name` |

## Rules (enforced by unit tests)

- Every image in `manifests.yaml` is either in `infrastructure/images.yaml`
  (group `base`) or declared in `unmirroredImages`. The cluster cannot pull
  anything else.
- Namespace names, namespace labels and symptoms do not leak the answer.
- `reset` only ever deletes namespaces that start with `scn-`.

Manifest comments such as `# The fault: ...` are safe: `kubectl apply` does
not send YAML comments to the cluster.
