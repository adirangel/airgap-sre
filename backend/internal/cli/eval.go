package cli

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/adirangel/airgap-sre/backend/internal/agent"
	"github.com/adirangel/airgap-sre/backend/internal/eval"
	"github.com/adirangel/airgap-sre/backend/internal/execx"
)

func newEvalCmd(a *app) *cobra.Command {
	cmd := &cobra.Command{Use: "eval", Short: "Score the agent against the failure scenarios"}

	var (
		timeout time.Duration
		keep    bool
		outDir  string
	)
	run := &cobra.Command{
		Use:   "run [scenario-id...]",
		Short: "Inject each scenario, ask the agent for a diagnosis, score it, write results and transcripts",
		Long: "For each scenario: reset, inject, wait until the failure is present, send only the symptom to the\n" +
			"agent, score the answer against expected.yaml, reset. Results and full transcripts are written to\n" +
			"docs/results/<date>/. Scoring is deterministic; there is no LLM judge.",
		RunE: func(cmd *cobra.Command, args []string) error {
			m, err := a.newAgentManager(cmd)
			if err != nil {
				return err
			}
			model, def, _, err := m.Load()
			if err != nil {
				return err
			}
			scenarios, err := pickScenarios(m.Lab, args, len(args) == 0)
			if err != nil {
				return err
			}
			ctx := cmd.Context()
			out := cmd.OutOrStdout()

			if err := agent.CheckModelEndpoint(ctx, model); err != nil {
				return err
			}
			if _, err := m.Lab.TC.Kubectl(ctx, "wait", "--for=condition=Ready", "agent/"+def.Name, "-n", def.Namespace, "--timeout=60s"); err != nil {
				return fmt.Errorf("agent %s is not ready — run `labctl agent install` first: %w", def.Name, err)
			}

			now := time.Now()
			if outDir == "" {
				outDir = eval.ResultsDir(m.Lab.Paths.Docs("results"), now)
			}
			meta := eval.Meta{
				Date:          now.Format("2006-01-02"),
				Started:       now.Format("2006-01-02 15:04 MST"),
				Provider:      model.Provider,
				Model:         model.Model,
				KagentVersion: m.Kagent.Version,
				Agent:         def.Name,
				Tools:         def.Tools,
			}
			if commit, err := a.runner.Run(ctx, execx.Cmd{Name: "git", Args: []string{"-C", m.Lab.Paths.Root, "rev-parse", "--short", "HEAD"}}); err == nil {
				meta.Commit = commit
			}

			// One port-forward per question: a dropped tunnel then costs one
			// scenario, not the rest of the run.
			ask := func(ctx context.Context, symptom string) (eval.Reply, error) {
				addr, stop, err := m.Lab.TC.PortForward(ctx, m.Kagent.Namespace, "svc/"+m.Kagent.ControllerService, m.Kagent.ControllerPort)
				if err != nil {
					return eval.Reply{}, err
				}
				defer stop()
				client := eval.A2AClient{BaseURL: "http://" + addr, HTTP: &http.Client{}}
				return client.Send(ctx, def.Namespace, def.Name, symptom)
			}

			_, _ = fmt.Fprintf(out, "eval: %d scenario(s), agent %s, model %s (%s)\n", len(scenarios), def.Name, model.Model, model.Provider)
			runner := eval.Runner{Cluster: m.Lab.Kube(), Ask: ask, Out: out, Timeout: timeout, Interval: 2 * time.Second, Keep: keep}
			results := runner.Run(ctx, scenarios)
			if len(results) == 0 {
				return errors.New("no scenario was evaluated")
			}
			report := eval.NewReport(meta, results)
			if err := report.Write(outDir); err != nil {
				return fmt.Errorf("write results: %w", err)
			}
			_, _ = fmt.Fprintf(out, "\n%s\nresults and transcripts: %s\n", strings.TrimRight(report.Table(), "\n"), outDir)
			return nil
		},
	}
	run.Flags().DurationVar(&timeout, "timeout", 15*time.Minute, "time limit for the agent per scenario")
	run.Flags().BoolVar(&keep, "keep", false, "leave each scenario in the cluster after it was scored")
	run.Flags().StringVar(&outDir, "out", "", "results directory (default: docs/results/<date>)")
	cmd.AddCommand(run)
	return cmd
}
