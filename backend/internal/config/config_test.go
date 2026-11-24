package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/adirangel/airgap-sre/backend/internal/repo"
)

func infra(t *testing.T, name string) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	p, err := repo.Find(wd)
	if err != nil {
		t.Fatal(err)
	}
	return p.Infra(name)
}

// The real pinned files must always load and be complete.
func TestRealVersionsFile(t *testing.T) {
	v, err := LoadVersions(infra(t, "versions.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"kind", "kubectl", "helm"} {
		tool, err := v.Tool(name)
		if err != nil {
			t.Error(err)
			continue
		}
		for _, platform := range []string{"windows/amd64", "linux/amd64"} {
			if tool.SHA256[platform] == "" {
				t.Errorf("%s has no checksum for %s", name, platform)
			}
		}
	}
	if !strings.Contains(v.Lab.NodeImage, "@sha256:") || !strings.Contains(v.Lab.Registry.Image, "@sha256:") {
		t.Error("node and registry images must be pinned by digest")
	}
	if got := v.Lab.Registry.ClusterEndpoint(); got != "http://"+v.Lab.Registry.ContainerName+":5000" {
		t.Errorf("ClusterEndpoint = %s", got)
	}
}

func TestRealImagesFile(t *testing.T) {
	im, err := LoadImages(infra(t, "images.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if len(im.Group("base")) == 0 {
		t.Error("images.yaml needs at least one image in group base")
	}
	if len(im.Group("all")) != len(im.Images) {
		t.Error(`Group("all") must return every image`)
	}
	v, err := LoadVersions(infra(t, "versions.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	for _, blocked := range v.Lab.UnmirroredImages {
		for _, i := range im.Images {
			if i.Ref == blocked {
				t.Errorf("%s is an un-mirrored probe and must not be in images.yaml", blocked)
			}
		}
	}
}

func TestLoadRejectsBadFiles(t *testing.T) {
	dir := t.TempDir()
	write := func(name, content string) string {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		return p
	}
	if _, err := LoadImages(write("unknown.yaml", "images:\n  - ref: docker.io/library/a:1\n    groups: [base]\n    typo: x\n")); err == nil {
		t.Error("unknown field should be rejected")
	}
	if _, err := LoadImages(write("nogroup.yaml", "images:\n  - ref: docker.io/library/a:1\n")); err == nil {
		t.Error("image without a group should be rejected")
	}
	if _, err := LoadImages(write("dup.yaml", "images:\n  - {ref: a/b:1, groups: [x]}\n  - {ref: a/b:1, groups: [x]}\n")); err == nil {
		t.Error("duplicate image should be rejected")
	}
	if _, err := LoadImages(write("digestref.yaml", "images:\n  - {ref: \"a/b:1@sha256:abc\", groups: [x]}\n")); err == nil {
		t.Error("digest inside ref should be rejected")
	}
	if _, err := LoadVersions(write("badsum.yaml", "tools:\n  - {name: kind, version: v1, url: http://x, sha256: {linux/amd64: nothex}}\n")); err == nil {
		t.Error("bad checksum should be rejected")
	}
}
