package lab

import (
	"context"
	"fmt"
	"io"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/crane"
	v1 "github.com/google/go-containerregistry/pkg/v1"

	"github.com/adirangel/airgap-sre/backend/internal/config"
)

// MirrorResult describes one copied image.
type MirrorResult struct {
	Source string
	Dest   string
	Digest string // upstream digest of the tag
}

// Mirror copies images registry-to-registry into the lab registry at
// registryAddr (host view, e.g. "localhost:5001"). It never goes through the
// Docker image store, so it leaves nothing behind on the host.
//
// Only the image for the given platform is copied (multi-arch indexes are
// resolved first), which keeps the registry small. Pods must therefore
// reference images by tag, which is what the scenarios and charts do.
func Mirror(ctx context.Context, out io.Writer, registryAddr string, images []config.Image, platform v1.Platform) ([]MirrorResult, error) {
	base := []crane.Option{crane.WithContext(ctx), crane.WithAuthFromKeychain(authn.DefaultKeychain)}
	var results []MirrorResult
	for _, img := range images {
		dest, err := MirrorRef(registryAddr, img.Ref)
		if err != nil {
			return results, err
		}
		digest, err := crane.Digest(img.Ref, base...)
		if err != nil {
			return results, fmt.Errorf("resolve %s: %w", img.Ref, err)
		}
		if img.Digest != "" && img.Digest != digest {
			return results, fmt.Errorf("%s: upstream digest is %s but images.yaml pins %s — refusing to mirror", img.Ref, digest, img.Digest)
		}
		_, _ = fmt.Fprintf(out, "mirroring %s\n       -> %s (%s/%s)\n", img.Ref, dest, platform.OS, platform.Architecture)
		p := platform
		if err := crane.Copy(img.Ref, dest, append(base, crane.WithPlatform(&p))...); err != nil {
			return results, fmt.Errorf("copy %s: %w", img.Ref, err)
		}
		results = append(results, MirrorResult{Source: img.Ref, Dest: dest, Digest: digest})
	}
	return results, nil
}

// UpstreamExists reports whether a public image can be resolved from the
// host. `lab verify` uses it to show that an image blocked in the cluster is
// a real, pullable image.
func UpstreamExists(ctx context.Context, ref string) (string, error) {
	return crane.Digest(ref, crane.WithContext(ctx), crane.WithAuthFromKeychain(authn.DefaultKeychain))
}
