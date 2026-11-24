package cli

import (
	"github.com/spf13/cobra"

	"github.com/adirangel/airgap-sre/backend/internal/cleanup"
)

func newCleanupCmd(a *app) *cobra.Command {
	var dryRun bool
	cmd := &cobra.Command{
		Use:   "cleanup",
		Short: "Remove everything this project owns on the machine (see docs/FOOTPRINT.md)",
		Long: "Removes all project-owned items listed in docs/FOOTPRINT.md.\n" +
			"Shared items (Go caches, Ollama models, ...) are never removed; their removal commands are printed instead.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			p, err := a.paths()
			if err != nil {
				return err
			}
			env := cleanup.Env{Paths: p, Runner: a.runner}
			return cleanup.Run(cmd.Context(), env, cmd.OutOrStdout(), dryRun)
		},
	}
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "print what would be removed without removing anything")
	return cmd
}
