# Machine footprint

Everything this project leaves on a machine, inside and outside the repo
folder. This file is the source of truth for `labctl cleanup`:

- Rows marked **project-owned** are removed by `labctl cleanup`
  (`--dry-run` prints them first). A unit test
  (`backend/internal/cleanup/cleanup_test.go`) fails if a project-owned row
  has no cleanup target.
- Rows marked **shared** are never removed by `labctl`; it only prints the
  command. They are **optional — affects other projects**.
- Rows marked **manual** need a human (accounts, environment variables).

Rule: any change that makes the project leave something new outside the repo
folder adds a row here in the same commit.

Nothing is installed globally. No `PATH`, registry, or user-profile edits.
`~/.kube/config` is never touched.

## Inside the repo folder (gitignored)

| ID | Item | Location | Approx. size | Ownership | Removal (PowerShell) |
| --- | --- | --- | --- | --- | --- |
| `tools-dir` | Pinned `kind`, `kubectl`, `helm` binaries | `.tools/` | ~200 MB | project-owned | `go run ./backend/cmd/labctl cleanup` |
| `lab-dir` | Lab kubeconfig, Helm cache/config/data, rendered configs | `.lab/` | < 50 MB | project-owned | `go run ./backend/cmd/labctl cleanup` |

## Outside the repo folder

| ID | Item | Location | Approx. size | Ownership | Removal (PowerShell) |
| --- | --- | --- | --- | --- | --- |
| `go-mod-cache` | Go module cache (this project's dependencies, plus golangci-lint sources if you lint locally with `go run`) | `%USERPROFILE%\go\pkg\mod` | ~300 MB for this project | shared (optional — affects other projects) | `go clean -modcache` |
| `go-build-cache` | Go build cache | `%LOCALAPPDATA%\go-build` | ~500 MB for this project | shared (optional — affects other projects) | `go clean -cache` |
| `github-repo` | Private GitHub repository `adirangel/airgap-sre` (code, Actions logs and artifacts) | github.com | n/a | manual | `gh repo delete adirangel/airgap-sre --yes` |
