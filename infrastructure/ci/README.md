# CI

GitHub only reads workflows from `.github/workflows/`, and this repo does not
use symlinks, so the workflow file lives there:
[`../../.github/workflows/ci.yml`](../../.github/workflows/ci.yml).

| Job | Runner | What it does |
| --- | --- | --- |
| `lint-test` | `ubuntu-latest` | `go vet ./...`, golangci-lint (pinned), `go test ./...` |
| `lab-e2e` | `ubuntu-latest` | Builds `labctl`, brings the lab up with kind, mirrors the `base` image group, proves the air gap with `lab verify`, runs inject + verify + reset for every scenario (from M2), then `cleanup`. No LLM and no secrets are needed. |

Action and linter versions are recorded in [`../VERSIONS.md`](../VERSIONS.md).
