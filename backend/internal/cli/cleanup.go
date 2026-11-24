package cli

import (
	"github.com/spf13/cobra"

	"github.com/adirangel/airgap-sre/backend/internal/cleanup"
	"github.com/adirangel/airgap-sre/backend/internal/config"
)

func newCleanupCmd(a *app) *cobra.Command {
	var dryRun bool
	cmd := &cobra.Command{
		Use:   "cleanup",
		Short: "Remove everything this project owns on the machine (see docs/FOOTPRINT.md)",
		Long: "Removes all project-owned items listed in docs/FOOTPRINT.md: the kind cluster, the registry\n" +
			"container and volume, the kind Docker network, project Docker images, .lab/ and .tools/.\n" +
			"Shared items (Go caches, Ollama models, ...) are never removed; their removal commands are printed instead.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			p, err := a.paths()
			if err != nil {
				return err
			}
			versions, err := config.LoadVersions(p.Infra("versions.yaml"))
			if err != nil {
				return err
			}
			env := cleanup.Env{Paths: p, Lab: versions.Lab, Runner: a.runner}
			return cleanup.Run(cmd.Context(), env, cmd.OutOrStdout(), dryRun)
		},
	}
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "print what would be removed without removing anything")
	return cmd
}
