# Versions

Versions are never taken from memory. Each entry records what was chosen,
where it was looked up, and why. Checked on **2026-11-24**.

Machine-readable pins live next to this file:

- [`versions.yaml`](versions.yaml) — tool binaries (with per-platform
  checksums), node image, registry image.
- [`images.yaml`](images.yaml) — every image the lab cluster may run.

## CI (M0)

| Thing | Version | Source checked | Why |
| --- | --- | --- | --- |
| Go | from `go.mod` | `go version` on the dev machine (go1.26.4) | CI uses `go-version-file: go.mod`, so local and CI match |
| `actions/checkout` | v7.0.1 | GitHub releases API (`releases/latest`) | latest stable |
| `actions/setup-go` | v7.1.0 | GitHub releases API | latest stable |
| `golangci/golangci-lint-action` | v9.3.0 | GitHub releases API | latest stable |
| golangci-lint | v2.15.0 | GitHub releases API | latest stable; config is the v2 format (`.golangci.yml`) |
| `spf13/cobra` | see `go.mod` | `go get ...@latest` | de-facto standard Go CLI library |

## Lab (M1)

| Thing | Version | Source checked | Why |
| --- | --- | --- | --- |
| kind | v0.33.0 | GitHub releases API for `kubernetes-sigs/kind`; checksums from the release's `*.sha256sum` assets | latest stable |
| Node image | `kindest/node:v1.37.0@sha256:a1ed56cf…` | kind v0.33.0 release notes | The default image of this kind release. The notes say the `@sha256` digest must be used |
| kubectl | v1.37.2 | `https://dl.k8s.io/release/stable.txt`; checksums from `dl.k8s.io/.../kubectl.sha256` | latest stable patch of the same minor as the node (1.37), inside the supported ±1 skew |
| Helm | v4.3.2 | GitHub releases API for `helm/helm`; checksums from `get.helm.sh/*.sha256sum` | latest stable. Used from M3 |
| Registry | `registry:3.1.2@sha256:82f6ff4a…` | GitHub releases API for `distribution/distribution` (v3.1.2); digest from Docker Hub | The plain CNCF Distribution registry, as required (not Harbor). kind's own local-registry guide uses `registry:3` |
| busybox | `docker.io/library/busybox:1.37.0@sha256:a7dbc319…` | Docker Hub | Small, pinned workload image for `lab verify` and the scenarios |
| `google/go-containerregistry` | see `go.mod` (v0.22.1) | GitHub releases API | Copies images registry-to-registry inside `labctl`, so mirroring needs no extra binary and leaves nothing in the host Docker image store |

### How the image-level air gap works

Read first: the kind [local registry guide](https://kind.sigs.k8s.io/docs/user/local-registry/)
and containerd's [`hosts.md`](https://github.com/containerd/containerd/blob/main/docs/hosts.md).

- With kind v0.27+ node images, containerd already reads
  `/etc/containerd/certs.d`, so no `containerdConfigPatches` are needed
  (stated in kind's `kind-with-registry.sh`). The node runs containerd v2.3.4.
- `labctl lab up` writes one `hosts.toml` per upstream registry (every
  registry in `images.yaml` plus docker.io, registry.k8s.io, ghcr.io, quay.io,
  gcr.io) and one `_default/hosts.toml` catch-all.
- Each file sets **both** `server` and the only `[host]` entry to the lab
  registry, with `override_path = true`. Setting `server` matters: when it is
  omitted, containerd falls back to the real upstream registry.
- An upstream image `docker.io/library/busybox:1.37.0` is stored in the lab
  registry as `docker.io/library/busybox:1.37.0` (upstream registry as the
  first path segment), so two registries can never collide. Pod specs keep
  the original image name; no chart or manifest has to be rewritten.
- `labctl lab verify` proves it on every run, including in CI: a mirrored
  image runs; `docker.io/library/hello-world` (own redirect) and
  `public.ecr.aws/docker/library/hello-world` (catch-all) cannot be pulled.

Limits of this scope (by design for M0–M3): network egress from pods is still
open, because M3 calls a cloud model API. Only image pulls are air-gapped.
The node image itself is pulled by the host Docker, not by the cluster.
