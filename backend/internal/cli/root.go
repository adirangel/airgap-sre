// Package cli wires the labctl command tree.
package cli

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/adirangel/airgap-sre/backend/internal/execx"
	"github.com/adirangel/airgap-sre/backend/internal/repo"
)

// Version is overridden at build time with -ldflags "-X ...cli.Version=...".
var Version = "dev"

type app struct {
	repoRoot string
	runner   execx.Runner
}

// paths resolves the repository root (flag, env var, or walking up from cwd).
func (a *app) paths() (repo.Paths, error) {
	if a.repoRoot != "" {
		if err := os.Setenv(repo.EnvRoot, a.repoRoot); err != nil {
			return repo.Paths{}, err
		}
	}
	wd, err := os.Getwd()
	if err != nil {
		return repo.Paths{}, err
	}
	return repo.Find(wd)
}

// NewRoot builds the labctl root command.
func NewRoot() *cobra.Command {
	a := &app{runner: execx.Local{}}
	root := &cobra.Command{
		Use:           "labctl",
		Short:         "Manage the airgap-sre lab: tools, cluster, registry, scenarios, evals, cleanup",
		Version:       Version,
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.PersistentFlags().StringVar(&a.repoRoot, "repo-root", "", "repository root (default: found by walking up from the current directory)")
	root.AddCommand(newToolsCmd(a), newLabCmd(a), newCleanupCmd(a))
	return root
}
