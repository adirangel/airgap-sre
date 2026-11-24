// Package repo locates the repository root and names every path labctl uses
// inside it. Nothing here may point outside the repository folder.
package repo

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ModulePath is the Go module path; it doubles as the marker used to
// recognise the repository root.
const ModulePath = "github.com/adirangel/airgap-sre"

// EnvRoot overrides root discovery when set.
const EnvRoot = "AIRGAP_SRE_ROOT"

// Paths gives access to the well-known locations inside the repository.
type Paths struct {
	Root string
}

// Find walks up from start until it finds the go.mod of this module.
func Find(start string) (Paths, error) {
	if v := os.Getenv(EnvRoot); v != "" {
		abs, err := filepath.Abs(v)
		if err != nil {
			return Paths{}, err
		}
		if !isRoot(abs) {
			return Paths{}, fmt.Errorf("%s=%q is not the airgap-sre repository root", EnvRoot, v)
		}
		return Paths{Root: abs}, nil
	}
	dir, err := filepath.Abs(start)
	if err != nil {
		return Paths{}, err
	}
	for {
		if isRoot(dir) {
			return Paths{Root: dir}, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return Paths{}, errors.New("airgap-sre repository root not found: run labctl from inside the repo or pass --repo-root")
		}
		dir = parent
	}
}

func isRoot(dir string) bool {
	f, err := os.Open(filepath.Join(dir, "go.mod"))
	if err != nil {
		return false
	}
	defer func() { _ = f.Close() }()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module ")) == ModulePath
		}
	}
	return false
}

// ToolsDir holds the pinned kind/kubectl/helm binaries (gitignored).
func (p Paths) ToolsDir() string { return filepath.Join(p.Root, ".tools") }

// LabDir holds lab state: kubeconfig, Helm homes, rendered configs (gitignored).
func (p Paths) LabDir() string { return filepath.Join(p.Root, ".lab") }

// Kubeconfig is the only kubeconfig labctl ever reads or writes.
func (p Paths) Kubeconfig() string { return filepath.Join(p.LabDir(), "kubeconfig") }

// Footprint is docs/FOOTPRINT.md.
func (p Paths) Footprint() string { return filepath.Join(p.Root, "docs", "FOOTPRINT.md") }

// Infra joins a path under infrastructure/.
func (p Paths) Infra(elem ...string) string {
	return filepath.Join(append([]string{p.Root, "infrastructure"}, elem...)...)
}

// Agents joins a path under agents/.
func (p Paths) Agents(elem ...string) string {
	return filepath.Join(append([]string{p.Root, "agents"}, elem...)...)
}

// Docs joins a path under docs/.
func (p Paths) Docs(elem ...string) string {
	return filepath.Join(append([]string{p.Root, "docs"}, elem...)...)
}
