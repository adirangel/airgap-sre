package lab

import (
	"fmt"
	"sort"
	"strings"

	"github.com/google/go-containerregistry/pkg/name"
)

// WellKnownRegistries always get a hosts.toml, even when no listed image
// comes from them, so an un-mirrored pull fails fast with a clear "not found"
// from the lab registry instead of reaching the internet.
var WellKnownRegistries = []string{"docker.io", "registry.k8s.io", "ghcr.io", "quay.io", "gcr.io"}

// DefaultRegistryDir is containerd's fallback directory for every registry
// that has no directory of its own.
const DefaultRegistryDir = "_default"

// CertsDir is where containerd on a kind node reads per-registry config.
const CertsDir = "/etc/containerd/certs.d"

// SplitRef normalises an image reference the way containerd does and returns
// its registry host ("docker.io" for Docker Hub), repository path and tag.
func SplitRef(ref string) (registry, repository, tag string, err error) {
	t, err := name.NewTag(ref, name.StrictValidation)
	if err != nil {
		return "", "", "", fmt.Errorf("image %q must be a fully qualified name with a tag: %w", ref, err)
	}
	registry = t.RegistryStr()
	if registry == name.DefaultRegistry {
		registry = "docker.io"
	}
	return registry, t.RepositoryStr(), t.TagStr(), nil
}

// MirrorRef is where an upstream image lives in the lab registry:
// <registryAddr>/<upstream registry>/<repository>:<tag>. Keeping the upstream
// registry as the first path segment avoids collisions between registries.
func MirrorRef(registryAddr, ref string) (string, error) {
	registry, repository, tag, err := SplitRef(ref)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s/%s/%s:%s", registryAddr, Namespace(registry), repository, tag), nil
}

// Namespace is the first path segment used for an upstream registry inside
// the lab registry. A port separator is not valid in a repository path, so
// "registry.example.com:5000" becomes "registry.example.com_5000".
func Namespace(registry string) string {
	return strings.ReplaceAll(registry, ":", "_")
}

// Registries returns the sorted set of upstream registries that need a
// hosts.toml: every registry used by refs plus the well-known ones.
func Registries(refs []string) ([]string, error) {
	set := map[string]bool{}
	for _, r := range WellKnownRegistries {
		set[r] = true
	}
	for _, ref := range refs {
		registry, _, _, err := SplitRef(ref)
		if err != nil {
			return nil, err
		}
		set[registry] = true
	}
	out := make([]string, 0, len(set))
	for r := range set {
		out = append(out, r)
	}
	sort.Strings(out)
	return out, nil
}

// HostsTOML renders the containerd hosts.toml that sends every pull for one
// upstream registry to the lab registry and nowhere else.
//
// endpoint is the lab registry as seen from the nodes, e.g.
// "http://airgap-sre-registry:5000". With override_path the URL path is used
// as the API root, so docker.io/library/busybox is served from the lab
// repository docker.io/library/busybox.
//
// "server" is set explicitly: when it is omitted containerd falls back to the
// real upstream registry, which would break the air gap.
func HostsTOML(endpoint, registry string) string {
	return render(strings.TrimRight(endpoint, "/") + "/v2/" + Namespace(registry))
}

// DefaultHostsTOML is the catch-all for registries with no directory of their
// own. Nothing is stored at the lab registry root, so these pulls fail.
func DefaultHostsTOML(endpoint string) string {
	return render(strings.TrimRight(endpoint, "/") + "/v2/_unmirrored")
}

func render(url string) string {
	return fmt.Sprintf(`# Managed by labctl (airgap-sre). Do not edit.
# Image-level air gap: pulls go to the lab registry only.
server = %[1]q
capabilities = ["pull", "resolve"]
override_path = true

[host.%[1]q]
  capabilities = ["pull", "resolve"]
  override_path = true
`, url)
}
