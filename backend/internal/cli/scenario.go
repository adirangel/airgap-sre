package cli

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/adirangel/airgap-sre/backend/internal/lab"
	"github.com/adirangel/airgap-sre/backend/internal/scenario"
)

// pickScenarios resolves command arguments to scenarios: explicit ids, or
// every scenario with --all.
func pickScenarios(l *lab.Lab, args []string, all bool) ([]scenario.Scenario, error) {
	scenarios, err := scenario.LoadAll(l.Paths.Infra("scenarios"))
	if err != nil {
		return nil, err
	}
	switch {
	case all && len(args) > 0:
		return nil, errors.New("pass scenario ids or --all, not both")
	case all:
		return scenarios, nil
	case len(args) == 0:
		return nil, errors.New("pass at least one scenario id, or --all (see `labctl scenario list`)")
	}
	var picked []scenario.Scenario
	for _, id := range args {
		s, err := scenario.Find(scenarios, id)
		if err != nil {
			return nil, err
		}
		picked = append(picked, s)
	}
	return picked, nil
}

func printChecks(out io.Writer, results []scenario.CheckResult) {
	for _, r := range results {
		mark := "ok  "
		if !r.OK {
			mark = "FAIL"
		}
		_, _ = fmt.Fprintf(out, "    %s %-20s %s\n", mark, r.Check.Type, r.Detail)
	}
}

func newScenarioCmd(a *app) *cobra.Command {
	cmd := &cobra.Command{Use: "scenario", Short: "Inject, verify and reset reproducible failure scenarios"}

	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List the available scenarios",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			l, err := a.newLab(cmd)
			if err != nil {
				return err
			}
			scenarios, err := scenario.LoadAll(l.Paths.Infra("scenarios"))
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			for _, s := range scenarios {
				_, _ = fmt.Fprintf(out, "%d. %-28s %s\n   category: %-26s namespace: %s\n", s.Order, s.ID, s.Title, s.Expected.RootCauseCategory, s.Namespace)
			}
			return nil
		},
	})

	var injectAll bool
	inject := &cobra.Command{
		Use:   "inject [id...]",
		Short: "Create the scenario namespace and apply its broken manifests",
		RunE: func(cmd *cobra.Command, args []string) error {
			l, err := a.newLab(cmd)
			if err != nil {
				return err
			}
			picked, err := pickScenarios(l, args, injectAll)
			if err != nil {
				return err
			}
			for _, s := range picked {
				if err := scenario.Inject(cmd.Context(), l.Kube(), s); err != nil {
					return fmt.Errorf("inject %s: %w", s.ID, err)
				}
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "injected %s into namespace %s\n", s.ID, s.Namespace)
			}
			return nil
		},
	}
	inject.Flags().BoolVar(&injectAll, "all", false, "inject every scenario")
	cmd.AddCommand(inject)

	var verifyAll bool
	verify := &cobra.Command{
		Use:   "verify [id...]",
		Short: "Wait until the scenario's failure is really present in the cluster",
		RunE: func(cmd *cobra.Command, args []string) error {
			l, err := a.newLab(cmd)
			if err != nil {
				return err
			}
			picked, err := pickScenarios(l, args, verifyAll)
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			var failed []string
			for _, s := range picked {
				results, err := scenario.VerifyFailure(cmd.Context(), l.Kube(), s, 2*time.Second)
				if err != nil {
					_, _ = fmt.Fprintf(out, "FAIL  %s: %v\n", s.ID, err)
					failed = append(failed, s.ID)
				} else {
					_, _ = fmt.Fprintf(out, "PASS  %s: failure is present\n", s.ID)
				}
				printChecks(out, results)
			}
			if len(failed) > 0 {
				return fmt.Errorf("failure not observed for: %s", strings.Join(failed, ", "))
			}
			return nil
		},
	}
	verify.Flags().BoolVar(&verifyAll, "all", false, "verify every scenario")
	cmd.AddCommand(verify)

	var resetAll bool
	reset := &cobra.Command{
		Use:   "reset [id...]",
		Short: "Delete the scenario namespace and wait until it is gone",
		RunE: func(cmd *cobra.Command, args []string) error {
			l, err := a.newLab(cmd)
			if err != nil {
				return err
			}
			picked, err := pickScenarios(l, args, resetAll)
			if err != nil {
				return err
			}
			for _, s := range picked {
				if err := scenario.Reset(cmd.Context(), l.Kube(), s); err != nil {
					return fmt.Errorf("reset %s: %w", s.ID, err)
				}
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "reset %s (namespace %s removed)\n", s.ID, s.Namespace)
			}
			return nil
		},
	}
	reset.Flags().BoolVar(&resetAll, "all", false, "reset every scenario")
	cmd.AddCommand(reset)

	return cmd
}
