// Package cleanup removes everything this project owns on the machine and
// prints (never runs) the removal commands for shared items.
//
// Every row in docs/FOOTPRINT.md marked "project-owned" must have a Target
// here with the same ID; cleanup_test.go enforces that.
package cleanup

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/adirangel/airgap-sre/backend/internal/config"
	"github.com/adirangel/airgap-sre/backend/internal/execx"
	"github.com/adirangel/airgap-sre/backend/internal/footprint"
	"github.com/adirangel/airgap-sre/backend/internal/lab"
	"github.com/adirangel/airgap-sre/backend/internal/repo"
)

// Env is what targets need to discover and remove their items.
type Env struct {
	Paths  repo.Paths
	Lab    config.Lab
	Runner execx.Runner
}

func (e Env) docker(ctx context.Context, args ...string) (string, error) {
	return e.Runner.Run(ctx, execx.Cmd{Name: "docker", Args: args})
}

// Step is one concrete removal.
type Step struct {
	Description string
	Do          func(ctx context.Context) error
}

// Target is one project-owned FOOTPRINT.md row.
type Target struct {
	ID   string // must equal the row ID in docs/FOOTPRINT.md
	Plan func(ctx context.Context, env Env) ([]Step, error)
}

// Targets returns all cleanup targets in removal order: containers first,
// then what they used, then the directories inside the repository.
func Targets() []Target {
	return []Target{
		{ID: "kind-cluster", Plan: planKindCluster},
		{ID: "registry-container", Plan: planRegistryContainer},
		{ID: "registry-volume", Plan: planRegistryVolume},
		{ID: "kind-network", Plan: planKindNetwork},
		{ID: "docker-images", Plan: planHostImages},
		{ID: "lab-dir", Plan: planDir(func(p repo.Paths) string { return p.LabDir() })},
		{ID: "tools-dir", Plan: planDir(func(p repo.Paths) string { return p.ToolsDir() })},
	}
}

// Run plans every target, then (unless dryRun) executes the steps. It always
// ends by printing shared and manual items from FOOTPRINT.md.
func Run(ctx context.Context, env Env, out io.Writer, dryRun bool) error {
	verb := "Removing"
	if dryRun {
		verb = "Would remove"
	}
	_, _ = fmt.Fprintf(out, "%s project-owned items:\n", verb)
	var firstErr error
	for _, t := range Targets() {
		steps, err := t.Plan(ctx, env)
		if err != nil {
			_, _ = fmt.Fprintf(out, "  [%s] could not inspect: %v\n", t.ID, err)
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		if len(steps) == 0 {
			_, _ = fmt.Fprintf(out, "  [%s] nothing to remove\n", t.ID)
			continue
		}
		for _, s := range steps {
			if dryRun {
				_, _ = fmt.Fprintf(out, "  [%s] %s\n", t.ID, s.Description)
				continue
			}
			if err := s.Do(ctx); err != nil {
				_, _ = fmt.Fprintf(out, "  [%s] FAILED %s: %v\n", t.ID, s.Description, err)
				if firstErr == nil {
					firstErr = err
				}
				continue
			}
			_, _ = fmt.Fprintf(out, "  [%s] removed: %s\n", t.ID, s.Description)
		}
	}

	rows, err := footprint.ParseFile(env.Paths.Footprint())
	if err != nil {
		return fmt.Errorf("read FOOTPRINT.md: %w", err)
	}
	printRows(out, "\nShared items (never removed by labctl; optional — affects other projects):", footprint.Filter(rows, footprint.Shared))
	printRows(out, "\nManual steps (labctl cannot do these for you):", footprint.Filter(rows, footprint.Manual))
	return firstErr
}

func printRows(out io.Writer, title string, rows []footprint.Row) {
	if len(rows) == 0 {
		return
	}
	_, _ = fmt.Fprintln(out, title)
	for _, r := range rows {
		_, _ = fmt.Fprintf(out, "  [%s] %s — %s (%s)\n      %s\n", r.ID, r.Item, r.Location, r.Size, r.Removal)
	}
}

func lines(s string) []string {
	var out []string
	for _, l := range strings.Split(s, "\n") {
		if l = strings.TrimSpace(l); l != "" {
			out = append(out, l)
		}
	}
	return out
}

// planKindCluster removes the kind node containers (and their anonymous
// volumes). It talks to Docker directly so it works even when .tools/ is gone.
func planKindCluster(ctx context.Context, env Env) ([]Step, error) {
	out, err := env.docker(ctx, "ps", "-a", "--filter", "label=io.x-k8s.kind.cluster="+env.Lab.ClusterName, "--format", "{{.Names}}")
	if err != nil {
		return nil, err
	}
	var steps []Step
	for _, name := range lines(out) {
		steps = append(steps, Step{
			Description: fmt.Sprintf("kind node container %s (cluster %s)", name, env.Lab.ClusterName),
			Do: func(ctx context.Context) error {
				_, err := env.docker(ctx, "rm", "-f", "-v", name)
				return err
			},
		})
	}
	return steps, nil
}

func planRegistryContainer(ctx context.Context, env Env) ([]Step, error) {
	name := env.Lab.Registry.ContainerName
	out, err := env.docker(ctx, "ps", "-a", "--filter", "name=^/"+name+"$", "--format", "{{.Names}}")
	if err != nil {
		return nil, err
	}
	if len(lines(out)) == 0 {
		return nil, nil
	}
	return []Step{{
		Description: "registry container " + name,
		Do: func(ctx context.Context) error {
			_, err := env.docker(ctx, "rm", "-f", "-v", name)
			return err
		},
	}}, nil
}

func planRegistryVolume(ctx context.Context, env Env) ([]Step, error) {
	name := env.Lab.Registry.VolumeName
	out, err := env.docker(ctx, "volume", "ls", "--filter", "name=^"+name+"$", "--format", "{{.Name}}")
	if err != nil {
		return nil, err
	}
	if len(lines(out)) == 0 {
		return nil, nil
	}
	return []Step{{
		Description: "registry volume " + name + " (all mirrored images)",
		Do: func(ctx context.Context) error {
			_, err := env.docker(ctx, "volume", "rm", name)
			return err
		},
	}}, nil
}

// planKindNetwork removes the "kind" Docker network, but only when nothing
// that is not ours is still attached to it.
func planKindNetwork(ctx context.Context, env Env) ([]Step, error) {
	out, err := env.docker(ctx, "network", "ls", "--filter", "name=^"+lab.KindNetwork+"$", "--format", "{{.Name}}")
	if err != nil {
		return nil, err
	}
	if len(lines(out)) == 0 {
		return nil, nil
	}
	return []Step{{
		Description: "docker network " + lab.KindNetwork,
		Do: func(ctx context.Context) error {
			attached, err := env.docker(ctx, "network", "inspect", lab.KindNetwork, "--format", "{{range .Containers}}{{.Name}} {{end}}")
			if err != nil {
				return err
			}
			if others := strings.Fields(attached); len(others) > 0 {
				return fmt.Errorf("still used by containers that are not part of this project: %s", strings.Join(others, ", "))
			}
			_, err = env.docker(ctx, "network", "rm", lab.KindNetwork)
			return err
		},
	}}, nil
}

// planHostImages removes the images labctl made the host Docker pull: the
// pinned node and registry images, plus anything recorded in
// .lab/host-images.txt from earlier pins.
func planHostImages(ctx context.Context, env Env) ([]Step, error) {
	refs := []string{env.Lab.NodeImage, env.Lab.Registry.Image}
	if data, err := os.ReadFile(filepath.Join(env.Paths.LabDir(), lab.HostImagesFile)); err == nil {
		refs = append(refs, lines(string(data))...)
	}
	// Fail early if Docker is unreachable, instead of reporting "nothing".
	if _, err := env.docker(ctx, "version", "--format", "{{.Server.Version}}"); err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	var steps []Step
	for _, ref := range refs {
		if ref == "" || seen[ref] {
			continue
		}
		seen[ref] = true
		size, err := env.docker(ctx, "image", "inspect", "--format", "{{.Size}}", ref)
		if err != nil {
			continue // not present
		}
		desc := "docker image " + ref
		if n, convErr := strconv.ParseInt(strings.TrimSpace(size), 10, 64); convErr == nil {
			desc += " (" + humanSize(n) + ")"
		}
		steps = append(steps, Step{
			Description: desc,
			Do: func(ctx context.Context) error {
				_, err := env.docker(ctx, "image", "rm", ref)
				return err
			},
		})
	}
	return steps, nil
}

// planDir removes a directory inside the repository.
func planDir(dir func(repo.Paths) string) func(context.Context, Env) ([]Step, error) {
	return func(_ context.Context, env Env) ([]Step, error) {
		d := dir(env.Paths)
		rel, err := filepath.Rel(env.Paths.Root, d)
		if err != nil || rel == "." || filepath.IsAbs(rel) || strings.HasPrefix(rel, "..") {
			return nil, fmt.Errorf("refusing to remove %q: not inside the repository", d)
		}
		if _, err := os.Stat(d); err != nil {
			if os.IsNotExist(err) {
				return nil, nil
			}
			return nil, err
		}
		return []Step{{
			Description: fmt.Sprintf("directory %s (%s)", d, humanSize(dirSize(d))),
			Do:          func(context.Context) error { return os.RemoveAll(d) },
		}}, nil
	}
}

func dirSize(dir string) int64 {
	var total int64
	_ = filepath.WalkDir(dir, func(_ string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if info, err := d.Info(); err == nil {
			total += info.Size()
		}
		return nil
	})
	return total
}

func humanSize(n int64) string {
	const mb = 1024 * 1024
	switch {
	case n >= 1024*mb:
		return fmt.Sprintf("%.1f GB", float64(n)/(1024*mb))
	case n >= mb:
		return fmt.Sprintf("%.0f MB", float64(n)/mb)
	default:
		return fmt.Sprintf("%.0f KB", float64(n)/1024)
	}
}
