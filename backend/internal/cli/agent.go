package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/adirangel/airgap-sre/backend/internal/agent"
	"github.com/adirangel/airgap-sre/backend/internal/config"
)

func (a *app) newAgentManager(cmd *cobra.Command) (*agent.Manager, error) {
	l, err := a.newLab(cmd)
	if err != nil {
		return nil, err
	}
	versions, err := config.LoadVersions(l.Paths.Infra("versions.yaml"))
	if err != nil {
		return nil, err
	}
	return &agent.Manager{Lab: l, Kagent: versions.Kagent, Out: cmd.OutOrStdout()}, nil
}

func newAgentCmd(a *app) *cobra.Command {
	cmd := &cobra.Command{Use: "agent", Short: "Install kagent and the SRE agent defined under agents/"}

	cmd.AddCommand(&cobra.Command{
		Use:   "install",
		Short: "Install kagent from the lab registry and apply the model, prompt and agent (idempotent)",
		Long: "Installs the pinned kagent charts, then applies agents/modelconfig.yaml, the system prompt and\n" +
			"agents/sre-agent.yaml. Run it again after editing anything under agents/.\n\n" +
			"If the ModelConfig references an API key, the key is read from the session environment variable\n" +
			agent.EnvAPIKey + " and stored only as a Kubernetes Secret.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			m, err := a.newAgentManager(cmd)
			if err != nil {
				return err
			}
			return m.Install(cmd.Context())
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "uninstall",
		Short: "Remove the agent and kagent from the cluster",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			m, err := a.newAgentManager(cmd)
			if err != nil {
				return err
			}
			return m.Uninstall(cmd.Context())
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "status",
		Short: "Show the configured model and whether the agent is ready",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			m, err := a.newAgentManager(cmd)
			if err != nil {
				return err
			}
			model, def, _, err := m.Load()
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			_, _ = fmt.Fprintf(out, "kagent    %s (namespace %s)\n", m.Kagent.Version, m.Kagent.Namespace)
			_, _ = fmt.Fprintf(out, "model     %s / %s (ModelConfig %s)\n", model.Provider, model.Model, model.Name)
			_, _ = fmt.Fprintf(out, "agent     %s, %d tools\n", def.Name, len(def.Tools))
			if missing, err := m.MissingImages(cmd.Context()); err != nil {
				_, _ = fmt.Fprintf(out, "images    unknown: %v\n", err)
			} else if len(missing) > 0 {
				_, _ = fmt.Fprintf(out, "images    %d not mirrored yet (run `labctl lab mirror --group kagent`)\n", len(missing))
			} else {
				_, _ = fmt.Fprintln(out, "images    all mirrored")
			}
			ready, err := m.Lab.TC.Kubectl(cmd.Context(), "get", "agent", def.Name, "-n", def.Namespace,
				"-o", `jsonpath={range .status.conditions[*]}{.type}={.status} {end}`)
			if err != nil {
				_, _ = fmt.Fprintln(out, "state     not installed (run `labctl agent install`)")
			} else {
				_, _ = fmt.Fprintf(out, "state     %s\n", ready)
			}
			if err := agent.CheckModelEndpoint(cmd.Context(), model); err != nil {
				_, _ = fmt.Fprintf(out, "endpoint  %v\n", err)
			} else {
				_, _ = fmt.Fprintln(out, "endpoint  ok")
			}
			return nil
		},
	})
	return cmd
}
