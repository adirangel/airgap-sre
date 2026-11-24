package cleanup

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/adirangel/airgap-sre/backend/internal/config"
	"github.com/adirangel/airgap-sre/backend/internal/execx"
	"github.com/adirangel/airgap-sre/backend/internal/footprint"
	"github.com/adirangel/airgap-sre/backend/internal/repo"
)

func repoPaths(t *testing.T) repo.Paths {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	p, err := repo.Find(wd)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

// Every project-owned row in docs/FOOTPRINT.md must be covered by a cleanup
// target, and every target must be documented.
func TestFootprintCoverage(t *testing.T) {
	rows, err := footprint.ParseFile(repoPaths(t).Footprint())
	if err != nil {
		t.Fatal(err)
	}
	targets := map[string]bool{}
	for _, tg := range Targets() {
		if targets[tg.ID] {
			t.Errorf("duplicate cleanup target %q", tg.ID)
		}
		targets[tg.ID] = true
	}
	documented := map[string]bool{}
	for _, r := range footprint.Filter(rows, footprint.ProjectOwned) {
		documented[r.ID] = true
		if !targets[r.ID] {
			t.Errorf("FOOTPRINT.md row %q is project-owned but labctl cleanup has no target for it", r.ID)
		}
	}
	for id := range targets {
		if !documented[id] {
			t.Errorf("cleanup target %q has no project-owned row in FOOTPRINT.md", id)
		}
	}
	if len(documented) == 0 {
		t.Fatal("no project-owned rows parsed from FOOTPRINT.md")
	}
	for _, r := range rows {
		if r.Ownership != footprint.ProjectOwned && r.Removal == "" {
			t.Errorf("row %q needs a removal command", r.ID)
		}
	}
}

var testLab = config.Lab{
	ClusterName: "airgap-sre",
	NodeImage:   "kindest/node:v1@sha256:aaa",
	Registry: config.Registry{
		Image:         "registry:3@sha256:bbb",
		ContainerName: "airgap-sre-registry",
		VolumeName:    "airgap-sre-registry-data",
		HostPort:      5001,
	},
}

func fakeRepo(t *testing.T) repo.Paths {
	t.Helper()
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "go.mod"), "module "+repo.ModulePath+"\n")
	mustWrite(t, filepath.Join(root, "docs", "FOOTPRINT.md"),
		"| ID | Item | Location | Approx. size | Ownership | Removal |\n| --- | --- | --- | --- | --- | --- |\n"+
			"| `go-mod-cache` | Go module cache | somewhere | 1 MB | shared | `go clean -modcache` |\n")
	mustWrite(t, filepath.Join(root, ".tools", "kind.exe"), "binary")
	mustWrite(t, filepath.Join(root, ".lab", "host-images.txt"), "kindest/node:v0@sha256:old\n")
	return repo.Paths{Root: root}
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// fakeDocker simulates a Docker daemon holding every project-owned item plus
// one image and one network user that belong to someone else.
type fakeDocker struct {
	calls           []string
	foreignOnKindNW bool
}

func (f *fakeDocker) Run(_ context.Context, c execx.Cmd) (string, error) {
	if c.Name != "docker" {
		return "", errors.New("unexpected program " + c.Name)
	}
	line := strings.Join(c.Args, " ")
	f.calls = append(f.calls, line)
	switch {
	case strings.HasPrefix(line, "ps -a --filter label=io.x-k8s.kind.cluster=airgap-sre"):
		return "airgap-sre-control-plane", nil
	case strings.HasPrefix(line, "ps -a --filter name=^/airgap-sre-registry$"):
		return "airgap-sre-registry", nil
	case strings.HasPrefix(line, "volume ls"):
		return "airgap-sre-registry-data", nil
	case strings.HasPrefix(line, "network ls"):
		return "kind", nil
	case strings.HasPrefix(line, "network inspect"):
		if f.foreignOnKindNW {
			return "someone-elses-cluster", nil
		}
		return "", nil
	case strings.HasPrefix(line, "image inspect"):
		if strings.HasSuffix(line, "sha256:old") {
			return "", errors.New("No such image")
		}
		return "104857600", nil
	}
	return "", nil
}

func (f *fakeDocker) ran(prefix string) bool {
	for _, c := range f.calls {
		if strings.HasPrefix(c, prefix) {
			return true
		}
	}
	return false
}

func TestDryRunRemovesNothing(t *testing.T) {
	p := fakeRepo(t)
	d := &fakeDocker{}
	var out bytes.Buffer
	if err := Run(context.Background(), Env{Paths: p, Lab: testLab, Runner: d}, &out, true); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(p.ToolsDir()); err != nil {
		t.Fatalf("dry-run removed .tools: %v", err)
	}
	for _, c := range d.calls {
		for _, destructive := range []string{"rm ", "volume rm", "network rm", "image rm"} {
			if strings.HasPrefix(c, destructive) {
				t.Errorf("dry-run executed a destructive docker command: %s", c)
			}
		}
	}
	got := out.String()
	for _, want := range []string{
		"Would remove",
		"[kind-cluster] kind node container airgap-sre-control-plane",
		"[registry-container] registry container airgap-sre-registry",
		"[registry-volume] registry volume airgap-sre-registry-data",
		"[kind-network] docker network kind",
		"[docker-images] docker image kindest/node:v1@sha256:aaa (100 MB)",
		"[docker-images] docker image registry:3@sha256:bbb",
		"[lab-dir] directory",
		"[tools-dir] directory",
		"go clean -modcache",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("dry-run output missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "sha256:old") {
		t.Errorf("an image that is not present was listed:\n%s", got)
	}
}

func TestRunRemovesEverythingOwned(t *testing.T) {
	p := fakeRepo(t)
	d := &fakeDocker{}
	var out bytes.Buffer
	if err := Run(context.Background(), Env{Paths: p, Lab: testLab, Runner: d}, &out, false); err != nil {
		t.Fatalf("%v\n%s", err, out.String())
	}
	for _, want := range []string{
		"rm -f -v airgap-sre-control-plane",
		"rm -f -v airgap-sre-registry",
		"volume rm airgap-sre-registry-data",
		"network rm kind",
		"image rm kindest/node:v1@sha256:aaa",
		"image rm registry:3@sha256:bbb",
	} {
		if !d.ran(want) {
			t.Errorf("expected docker %s; calls: %v", want, d.calls)
		}
	}
	for _, dir := range []string{p.ToolsDir(), p.LabDir()} {
		if _, err := os.Stat(dir); !os.IsNotExist(err) {
			t.Errorf("%s still exists (err=%v)", dir, err)
		}
	}
	if !strings.Contains(out.String(), "go clean -modcache") {
		t.Errorf("shared items not printed:\n%s", out.String())
	}
	if d.ran("clean") {
		t.Error("cleanup must never run commands for shared items")
	}
}

func TestKindNetworkKeptWhenOthersUseIt(t *testing.T) {
	p := fakeRepo(t)
	d := &fakeDocker{foreignOnKindNW: true}
	var out bytes.Buffer
	err := Run(context.Background(), Env{Paths: p, Lab: testLab, Runner: d}, &out, false)
	if err == nil {
		t.Fatal("expected an error because the kind network is still in use")
	}
	if d.ran("network rm") {
		t.Error("removed the kind network while a foreign container was attached")
	}
	if !strings.Contains(out.String(), "someone-elses-cluster") {
		t.Errorf("output should name the foreign container:\n%s", out.String())
	}
}

func TestPlanDirRefusesOutsideRepo(t *testing.T) {
	p := fakeRepo(t)
	plan := planDir(func(repo.Paths) string { return filepath.Dir(p.Root) })
	if _, err := plan(context.Background(), Env{Paths: p}); err == nil {
		t.Fatal("planDir must refuse a directory outside the repository")
	}
}
