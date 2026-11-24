# Versions

Versions are never taken from memory. Each entry records what was chosen,
where it was looked up, and why. Checked on **2026-11-24**.

## CI (M0)

| Thing | Version | Source checked | Why |
| --- | --- | --- | --- |
| Go | from `go.mod` | `go version` on the dev machine (go1.26.4) | CI uses `go-version-file: go.mod`, so local and CI match |
| `actions/checkout` | v7.0.1 | GitHub releases API (`releases/latest`) | latest stable |
| `actions/setup-go` | v7.1.0 | GitHub releases API | latest stable |
| `golangci/golangci-lint-action` | v9.3.0 | GitHub releases API | latest stable |
| golangci-lint | v2.15.0 | GitHub releases API | latest stable; config is the v2 format (`.golangci.yml`) |
| `spf13/cobra` | see `go.mod` | `go get ...@latest` | de-facto standard Go CLI library |
