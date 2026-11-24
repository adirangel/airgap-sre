# Machine footprint

Everything this project leaves on a machine, inside and outside the repo
folder. This file is the source of truth for `labctl cleanup`:

- Rows marked **project-owned** are removed by `labctl cleanup`
  (`--dry-run` prints them first). A unit test
  (`backend/internal/cleanup/cleanup_test.go`) fails if a project-owned row
  has no cleanup target, or a target has no row.
- Rows marked **shared** are never removed by `labctl`; it only prints the
  command. They are **optional — affects other projects**.
- Rows marked **manual** need a human (accounts, environment variables).

Rule: any change that makes the project leave something new outside the repo
folder adds a row here in the same commit.

Nothing is installed globally. No `PATH`, registry, or user-profile edits.
`~/.kube/config`, `~/.kube/cache` and the user's Helm directories are never
touched: `labctl` points `KUBECONFIG`, `KUBECACHEDIR` and `HELM_*_HOME` at
`.lab/` for every tool it runs.

`labctl` shorthand below means `go run ./backend/cmd/labctl`.

## Inside the repo folder (gitignored)

| ID | Item | Location | Approx. size | Ownership | Removal (PowerShell) |
| --- | --- | --- | --- | --- | --- |
| `lab-dir` | Lab kubeconfig, kubectl cache, Helm cache/config/data, list of host images | `.lab/` | < 50 MB | project-owned | `go run ./backend/cmd/labctl cleanup` |
| `tools-dir` | Pinned, checksum-verified `kind`, `kubectl`, `helm` binaries | `.tools/` | ~140 MB | project-owned | `go run ./backend/cmd/labctl cleanup` |

## Outside the repo folder: Docker Desktop

These live inside Docker Desktop's WSL2 disk (`%LOCALAPPDATA%\Docker\wsl\`),
not as loose files. Docker Desktop does not shrink that disk automatically
after items are removed.

| ID | Item | Location | Approx. size | Ownership | Removal (PowerShell) |
| --- | --- | --- | --- | --- | --- |
| `kind-cluster` | kind node container `airgap-sre-control-plane` and its anonymous volume (cluster state, images pulled by the cluster) | Docker container | 1–6 GB, grows with pulled images | project-owned | `go run ./backend/cmd/labctl cleanup` |
| `registry-container` | Lab registry container `airgap-sre-registry` (port `127.0.0.1:5001`, restarts with Docker Desktop until removed) | Docker container | < 1 MB | project-owned | `go run ./backend/cmd/labctl cleanup` |
| `registry-volume` | Volume `airgap-sre-registry-data`: every mirrored image. Kept by `lab down`, removed by `cleanup` | Docker volume | 2 MB after M1, a few GB with kagent images | project-owned | `go run ./backend/cmd/labctl cleanup` |
| `kind-network` | Docker network `kind`. Removed only if no foreign container is attached | Docker network | n/a | project-owned | `go run ./backend/cmd/labctl cleanup` |
| `docker-images` | Host images pulled for the lab: `kindest/node` (pinned) and `registry` (pinned), plus older pins recorded in `.lab/host-images.txt` | Docker image store | ~1.4 GB | project-owned | `go run ./backend/cmd/labctl cleanup` |

Mirrored images are copied registry-to-registry, so they never enter the host
Docker image store.

## Outside the repo folder: shared and manual

| ID | Item | Location | Approx. size | Ownership | Removal (PowerShell) |
| --- | --- | --- | --- | --- | --- |
| `go-mod-cache` | Go module cache (this project's dependencies, plus golangci-lint sources if you lint locally with `go run`) | `%USERPROFILE%\go\pkg\mod` | ~400 MB for this project | shared (optional — affects other projects) | `go clean -modcache` |
| `go-build-cache` | Go build cache | `%LOCALAPPDATA%\go-build` | ~500 MB for this project | shared (optional — affects other projects) | `go clean -cache` |
| `docker-vhdx` | Docker Desktop WSL2 disk does not shrink after cleanup. The command only shows usage; to reclaim the space use Docker Desktop > Troubleshoot > Clean / Purge data, which removes ALL Docker data | `%LOCALAPPDATA%\Docker\wsl\disk\docker_data.vhdx` | whatever the lab used | shared (optional — affects other projects) | `docker system df` |
| `github-repo` | Private GitHub repository `adirangel/airgap-sre` (code, Actions logs and artifacts) | github.com | n/a | manual | `gh repo delete adirangel/airgap-sre --yes` |
