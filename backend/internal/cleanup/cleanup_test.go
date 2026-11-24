package cleanup

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

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

func fakeRepo(t *testing.T) repo.Paths {
	t.Helper()
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "go.mod"), "module "+repo.ModulePath+"\n")
	mustWrite(t, filepath.Join(root, "docs", "FOOTPRINT.md"),
		"| ID | Item | Location | Approx. size | Ownership | Removal |\n| --- | --- | --- | --- | --- | --- |\n"+
			"| `go-mod-cache` | Go module cache | somewhere | 1 MB | shared | `go clean -modcache` |\n")
	mustWrite(t, filepath.Join(root, ".tools", "kind.exe"), "binary")
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

// noDocker answers every docker query with "nothing found".
var noDocker = execx.RunnerFunc(func(context.Context, execx.Cmd) (string, error) { return "", nil })

func TestDryRunRemovesNothing(t *testing.T) {
	p := fakeRepo(t)
	var out bytes.Buffer
	if err := Run(context.Background(), Env{Paths: p, Runner: noDocker}, &out, true); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(p.ToolsDir()); err != nil {
		t.Fatalf("dry-run removed .tools: %v", err)
	}
	got := out.String()
	for _, want := range []string{"Would remove", "[tools-dir] directory", "go clean -modcache"} {
		if !strings.Contains(got, want) {
			t.Errorf("dry-run output missing %q:\n%s", want, got)
		}
	}
}

func TestRunRemovesProjectDirs(t *testing.T) {
	p := fakeRepo(t)
	var out bytes.Buffer
	if err := Run(context.Background(), Env{Paths: p, Runner: noDocker}, &out, false); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(p.ToolsDir()); !os.IsNotExist(err) {
		t.Fatalf(".tools still exists (err=%v)", err)
	}
	if !strings.Contains(out.String(), "go clean -modcache") {
		t.Errorf("shared items not printed:\n%s", out.String())
	}
}
