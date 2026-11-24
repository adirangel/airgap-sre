# infrastructure/

Everything that describes the lab and the systems the agent will work with.

| File | Purpose |
| --- | --- |
| `VERSIONS.md` | Which versions were chosen, when, from which source, and why |
| `APPLICATION_STACK.md` | Inventory of the application stack the agent targets, and which existing MCP servers we reuse |
| `ci/README.md` | Notes on CI (the workflow files themselves must live in `.github/workflows/`) |
| `versions.yaml` | Pinned tool binaries (with checksums), node image, registry image |
| `images.yaml` | Every image the lab cluster may run; input of `labctl lab mirror` |
| `kind/cluster.yaml` | kind cluster config |
| `helm/kagent-values.yaml` | Helm values for the minimal, read-only kagent install |
| `scenarios/` | The failure scenarios: `scenario.yaml`, `manifests.yaml`, `expected.yaml` per scenario (see `scenarios/README.md`) |
