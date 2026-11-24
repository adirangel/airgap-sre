# CI

GitHub only reads workflows from `.github/workflows/`, and this repo does not
use symlinks, so the workflow file lives there:
[`../../.github/workflows/ci.yml`](../../.github/workflows/ci.yml).

| Job | Runner | What it does |
| --- | --- | --- |
| `lint-test` | `ubuntu-latest` | `go vet ./...`, golangci-lint (pinned), `go test ./...` |

Action and linter versions are recorded in [`../VERSIONS.md`](../VERSIONS.md).
