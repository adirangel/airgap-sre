# airgap-sre

An AI SRE agent for Kubernetes that is being built to run **fully air-gapped**:
local LLM, internal registry, RKE2.

This repository currently covers the first four milestones:

| Milestone | What it gives you | Status |
| --- | --- | --- |
| M0 Bootstrap | Layout, rules, CI, footprint tracking | done |
| M1 Lab | kind cluster + local registry; the cluster can pull **only** from that registry | planned |
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

_Filled in as milestones land._

```powershell
git clone https://github.com/adirangel/airgap-sre.git
cd airgap-sre
go run ./backend/cmd/labctl --help
```

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
