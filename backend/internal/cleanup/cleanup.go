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
	"strings"

	"github.com/adirangel/airgap-sre/backend/internal/execx"
	"github.com/adirangel/airgap-sre/backend/internal/footprint"
	"github.com/adirangel/airgap-sre/backend/internal/repo"
)

// Env is what targets need to discover and remove their items.
type Env struct {
	Paths  repo.Paths
	Runner execx.Runner
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

// Targets returns all cleanup targets in removal order.
func Targets() []Target {
	return []Target{
		{ID: "tools-dir", Plan: planDir(func(p repo.Paths) string { return p.ToolsDir() })},
		{ID: "lab-dir", Plan: planDir(func(p repo.Paths) string { return p.LabDir() })},
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
