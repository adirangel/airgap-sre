package lab

import (
	"bytes"
	"context"
	"io"
	"log"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/go-containerregistry/pkg/crane"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/registry"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/random"
	"github.com/google/go-containerregistry/pkg/v1/remote"

	"github.com/adirangel/airgap-sre/backend/internal/config"
)

func TestSplitRef(t *testing.T) {
	cases := []struct {
		ref, registry, repo, tag string
	}{
		{"docker.io/library/busybox:1.37.0", "docker.io", "library/busybox", "1.37.0"},
		{"ghcr.io/kagent-dev/kagent/controller:0.1.0", "ghcr.io", "kagent-dev/kagent/controller", "0.1.0"},
		{"registry.k8s.io/pause:3.10", "registry.k8s.io", "pause", "3.10"},
	}
	for _, c := range cases {
		registry, repo, tag, err := SplitRef(c.ref)
		if err != nil {
			t.Errorf("%s: %v", c.ref, err)
			continue
		}
		if registry != c.registry || repo != c.repo || tag != c.tag {
			t.Errorf("%s: got (%s, %s, %s)", c.ref, registry, repo, tag)
		}
	}
	for _, bad := range []string{"busybox", "docker.io/library/busybox", "docker.io/library/busybox@sha256:" + strings.Repeat("a", 64)} {
		if _, _, _, err := SplitRef(bad); err == nil {
			t.Errorf("%s: expected an error (fully qualified name with a tag is required)", bad)
		}
	}
}

func TestMirrorRef(t *testing.T) {
	got, err := MirrorRef("localhost:5001", "docker.io/library/busybox:1.37.0")
	if err != nil {
		t.Fatal(err)
	}
	if want := "localhost:5001/docker.io/library/busybox:1.37.0"; got != want {
		t.Errorf("got %s want %s", got, want)
	}
}

func TestRegistries(t *testing.T) {
	got, err := Registries([]string{"docker.io/library/busybox:1", "cr.example.com/team/app:2"})
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(got, ",")
	for _, want := range append([]string{"cr.example.com"}, WellKnownRegistries...) {
		if !strings.Contains(joined, want) {
			t.Errorf("registries %v missing %s", got, want)
		}
	}
}

// The air gap depends on these exact properties of hosts.toml.
func TestHostsTOML(t *testing.T) {
	got := HostsTOML("http://airgap-sre-registry:5000", "docker.io")
	for _, want := range []string{
		`server = "http://airgap-sre-registry:5000/v2/docker.io"`,
		`[host."http://airgap-sre-registry:5000/v2/docker.io"]`,
		`override_path = true`,
		`capabilities = ["pull", "resolve"]`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("hosts.toml missing %q:\n%s", want, got)
		}
	}
	// No public endpoint may appear anywhere.
	for _, banned := range []string{"https://", "registry-1.docker.io"} {
		if strings.Contains(got, banned) {
			t.Errorf("hosts.toml must not mention %q:\n%s", banned, got)
		}
	}
	if withPort := HostsTOML("http://r:5000", "registry.example.com:5000"); !strings.Contains(withPort, `/v2/registry.example.com_5000"`) {
		t.Errorf("registry with a port must be sanitised:\n%s", withPort)
	}
	if def := DefaultHostsTOML("http://airgap-sre-registry:5000/"); !strings.Contains(def, `server = "http://airgap-sre-registry:5000/v2/_unmirrored"`) {
		t.Errorf("default hosts.toml wrong:\n%s", def)
	}
}

func newRegistry(t *testing.T) string {
	t.Helper()
	srv := httptest.NewServer(registry.New(registry.Logger(log.New(io.Discard, "", 0))))
	t.Cleanup(srv.Close)
	return strings.TrimPrefix(srv.URL, "http://")
}

func TestMirrorCopiesOnePlatformAndChecksDigest(t *testing.T) {
	upstream := newRegistry(t)
	labReg := newRegistry(t)

	// Build a two-platform index upstream.
	amd, _ := random.Image(256, 1)
	arm, _ := random.Image(256, 1)
	idx := mutate.AppendManifests(empty.Index,
		mutate.IndexAddendum{Add: amd, Descriptor: v1.Descriptor{Platform: &v1.Platform{OS: "linux", Architecture: "amd64"}}},
		mutate.IndexAddendum{Add: arm, Descriptor: v1.Descriptor{Platform: &v1.Platform{OS: "linux", Architecture: "arm64"}}},
	)
	src := upstream + "/team/app:1.0"
	srcRef, err := name.ParseReference(src)
	if err != nil {
		t.Fatal(err)
	}
	if err := remote.WriteIndex(srcRef, idx); err != nil {
		t.Fatal(err)
	}
	indexDigest, err := crane.Digest(src)
	if err != nil {
		t.Fatal(err)
	}
	amdDigest, _ := amd.Digest()

	var out bytes.Buffer
	images := []config.Image{{Ref: src, Digest: indexDigest, Groups: []string{"base"}}}
	res, err := Mirror(context.Background(), &out, labReg, images, v1.Platform{OS: "linux", Architecture: "amd64"})
	if err != nil {
		t.Fatalf("%v\n%s", err, out.String())
	}
	wantDest := labReg + "/" + strings.ReplaceAll(upstream, ":", "_") + "/team/app:1.0"
	if len(res) != 1 || res[0].Dest != wantDest {
		t.Fatalf("results = %+v, want dest %s", res, wantDest)
	}
	got, err := crane.Digest(wantDest)
	if err != nil {
		t.Fatal(err)
	}
	if got != amdDigest.String() {
		t.Errorf("mirrored digest %s, want the linux/amd64 image %s", got, amdDigest)
	}

	// A pin that no longer matches upstream must stop the mirror.
	images[0].Digest = "sha256:" + strings.Repeat("0", 64)
	if _, err := Mirror(context.Background(), &out, labReg, images, v1.Platform{OS: "linux", Architecture: "amd64"}); err == nil ||
		!strings.Contains(err.Error(), "refusing to mirror") {
		t.Fatalf("want a digest mismatch error, got %v", err)
	}
}
