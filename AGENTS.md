# AGENTS.md

Rules for humans and coding agents working in this repository.
`CLAUDE.md` is only a shim that imports this file.

## What this project is

`airgap-sre` is an AI SRE agent for Kubernetes that will eventually run fully
air-gapped (local LLM, internal registry, RKE2). The current scope is a local
lab, reproducible failure scenarios with a deterministic scoring harness, and a
[kagent](https://kagent.dev)-based agent measured against those scenarios.

## Language

- Everything inside the repository is English: code, comments, commit
  messages, README and docs.

## Layout

| Path              | Contents                                                                 |
| ----------------- | ------------------------------------------------------------------------ |
| `backend/`        | The Go CLI `labctl` (`backend/cmd/labctl`, `backend/internal/...`).       |
| `agents/`         | kagent `Agent` / `ModelConfig` manifests and system prompts.              |
| `infrastructure/` | kind config, Helm values, pinned versions, image list, scenarios, CI notes, and the "Application stack" inventory. |
| `docs/`           | `FOOTPRINT.md`, `gaps.md`, eval results.                                  |

There is no `frontend/`. The Go module lives at the repo root
(`github.com/adirangel/airgap-sre`).

## Hard rules

1. **Windows-first.** Everything must work from PowerShell on Windows with
   Docker Desktop (WSL2 backend). No symlinks. No bash-only scripts. CI runs on
   Linux, so code must be portable: use `filepath`, never hard-code path
   separators, never shell out through `sh`/`cmd`.
2. **One CLI.** All lab, scenario, eval and cleanup operations go through
   `labctl`. Do not add shell scripts or Makefiles that users must run.
3. **No global installs.** No winget/choco/scoop, no `go install`, no `setx`,
   no edits to `PATH` or the user profile. `labctl` downloads pinned,
   checksum-verified `kind`, `kubectl` and `helm` into `.tools/` and uses only
   those.
4. **Repo-local kubeconfig.** The lab kubeconfig is `.lab/kubeconfig`. Pass
   `--kubeconfig` everywhere. Never read or write `~/.kube/config`.
5. **Footprint is documented.** Any change that makes the project leave
   something outside the repo folder adds a row to `docs/FOOTPRINT.md` in the
   same commit, and (if project-owned) a matching target in
   `backend/internal/cleanup`. A unit test enforces the mapping.
6. **Image-level air gap.** The lab cluster may pull images only from the lab
   registry. Every image the lab needs is listed in
   `infrastructure/images.yaml` and goes through `labctl lab mirror` first.
7. **No versions from memory.** Tool, chart and image versions are pinned in
   `infrastructure/versions.yaml` / `infrastructure/images.yaml`, with the
   reasoning in `infrastructure/VERSIONS.md`. Read upstream docs before
   changing them.
8. **Reuse, don't rebuild.** Do not write MCP servers for Harbor or Jenkins;
   `oreondev/harbor-mcp` and the Jenkins `mcp-server` plugin already exist.
9. **No secrets in the repo.** API keys come from environment variables only.
10. **License** is Apache-2.0.

## Workflow

- One branch per milestone (`m1-lab`, `m2-scenarios`, ...). Merge to `main`
  only when unit tests and CI pass.
- Before committing: `go vet ./...`, `go test ./...`, and golangci-lint (CI
  runs the pinned version).
- Scoring in `labctl eval` must stay deterministic: no LLM judges.

## Common commands (PowerShell or any shell)

```powershell
go run ./backend/cmd/labctl --help
go test ./...
go vet ./...
```
