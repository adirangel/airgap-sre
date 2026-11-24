# airgap-sre

An AI SRE agent for Kubernetes that is being built to run **fully air-gapped**:
local LLM, internal registry, RKE2.

This repository currently covers the first four milestones:

| Milestone | What it gives you | Status |
| --- | --- | --- |
| M0 Bootstrap | Layout, rules, CI, footprint tracking | done |
| M1 Lab | kind cluster + local registry; the cluster can pull **only** from that registry | done |
| M2 Scenarios | Five reproducible failures with inject / verify / reset | planned |
| M3 Agent baseline | A [kagent](https://kagent.dev) agent scored against the scenarios | planned |

Local models and the full air-gap bundle come after M3.

## Requirements

Only these, on a clean Windows 11 machine:

- Docker Desktop (WSL2 backend), running
- Go
- git

Everything else (`kind`, `kubectl`, `helm`) is downloaded by `labctl` into
`.tools/` inside the repo, pinned and checksum-verified. Nothing is installed
globally. See [docs/FOOTPRINT.md](docs/FOOTPRINT.md).

## Quickstart

All commands run from the repo root, in PowerShell (or any shell).

```powershell
git clone https://github.com/adirangel/airgap-sre.git
cd airgap-sre

# 1. Registry + kind cluster. Downloads kind/kubectl/helm into .tools/ first.
go run ./backend/cmd/labctl lab up

# 2. Copy the allowed images into the lab registry.
go run ./backend/cmd/labctl lab mirror --group base

# 3. Prove the air gap: a mirrored image runs, un-mirrored public images cannot be pulled.
go run ./backend/cmd/labctl lab verify

go run ./backend/cmd/labctl lab status
go run ./backend/cmd/labctl lab down      # delete cluster + registry container (mirrored images are kept)
```

The lab kubeconfig is `.lab/kubeconfig`; `~/.kube/config` is never touched.
To use kubectl yourself:

```powershell
.\.tools\kubectl-v1.37.2\kubectl.exe --kubeconfig .lab\kubeconfig get pods -A
```

### How the air gap works

Image-level only (for now): containerd on the node is configured so that
every registry — docker.io, ghcr.io, and a catch-all for everything else —
resolves **only** to the lab registry. Pod specs keep their original image
names. An image that was not mirrored fails with `ImagePullBackOff`.
Network egress stays open until local models arrive. Details:
[infrastructure/VERSIONS.md](infrastructure/VERSIONS.md).

## Removing everything

```powershell
go run ./backend/cmd/labctl cleanup --dry-run   # show what would be removed
go run ./backend/cmd/labctl cleanup             # remove project-owned items
```

Shared items (Go caches, etc.) are never removed; the command prints how to
remove them yourself.

## Layout

| Path | Contents |
| --- | --- |
| `backend/` | The Go CLI `labctl` |
| `agents/` | kagent Agent / ModelConfig manifests and system prompts |
| `infrastructure/` | kind config, Helm values, pinned versions, image list, scenarios, application stack inventory |
| `docs/` | Footprint, gaps, eval results |

## License

[Apache-2.0](LICENSE)
