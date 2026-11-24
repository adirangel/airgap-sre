package eval

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/adirangel/airgap-sre/backend/internal/scenario"
)

// AskFunc sends the symptom to the agent and returns its reply.
type AskFunc func(ctx context.Context, symptom string) (Reply, error)

// Runner evaluates an agent against scenarios, one at a time.
type Runner struct {
	Cluster  scenario.Cluster
	Ask      AskFunc
	Out      io.Writer
	Timeout  time.Duration // per agent call
	Interval time.Duration // verify poll interval
	Keep     bool          // leave the scenario in the cluster afterwards
}

// Result is the outcome for one scenario.
type Result struct {
	Order      int     `json:"order"`
	ID         string  `json:"id"`
	Title      string  `json:"title"`
	Symptom    string  `json:"symptom"`
	Score      Score   `json:"score"`
	Answer     string  `json:"answer"`
	Error      string  `json:"error,omitempty"` // set when setup or the agent call failed
	Seconds    float64 `json:"seconds"`         // time the agent took
	ToolCalls  int     `json:"toolCalls"`
	Tokens     int     `json:"tokens"`
	Transcript string  `json:"transcript"` // path relative to the results directory

	Reply Reply `json:"-"`
}

func (r *Runner) logf(format string, args ...any) { _, _ = fmt.Fprintf(r.Out, format, args...) }

// Run evaluates every scenario. A failure in one scenario is recorded in its
// Result and does not stop the others.
func (r *Runner) Run(ctx context.Context, scenarios []scenario.Scenario) []Result {
	results := make([]Result, 0, len(scenarios))
	for _, s := range scenarios {
		if ctx.Err() != nil {
			break
		}
		results = append(results, r.one(ctx, s))
	}
	return results
}

func (r *Runner) one(ctx context.Context, s scenario.Scenario) Result {
	res := Result{Order: s.Order, ID: s.ID, Title: s.Title, Symptom: s.Symptom, Score: ScoreAnswer(s, "")}
	r.logf("[%d] %s\n", s.Order, s.ID)
	fail := func(stage string, err error) Result {
		res.Error = stage + ": " + err.Error()
		r.logf("    ERROR %s\n", res.Error)
		return res
	}

	// A clean start: an old copy of the scenario must not help or confuse.
	if err := scenario.Reset(ctx, r.Cluster, s); err != nil {
		return fail("reset before inject", err)
	}
	if err := scenario.Inject(ctx, r.Cluster, s); err != nil {
		return fail("inject", err)
	}
	if !r.Keep {
		defer func() {
			// A fresh context so the cluster is cleaned up even after Ctrl+C.
			cctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
			defer cancel()
			if err := scenario.Reset(cctx, r.Cluster, s); err != nil {
				r.logf("    WARNING reset failed: %v\n", err)
			}
		}()
	}
	if _, err := scenario.VerifyFailure(ctx, r.Cluster, s, r.Interval); err != nil {
		return fail("verify (the failure never showed up, so the agent was not asked)", err)
	}
	r.logf("    failure is present; asking the agent\n")

	askCtx, cancel := context.WithTimeout(ctx, r.Timeout)
	start := time.Now()
	reply, err := r.Ask(askCtx, s.Symptom)
	cancel()
	res.Seconds = time.Since(start).Seconds()
	res.Reply = reply
	res.Answer = reply.Answer
	res.ToolCalls = reply.ToolCalls
	res.Tokens = reply.Tokens
	if err != nil {
		// Score whatever text came back, but record the error.
		res.Score = ScoreAnswer(s, reply.Answer)
		return fail("agent", err)
	}
	res.Score = ScoreAnswer(s, reply.Answer)
	verdict := "FAIL"
	if res.Score.Pass {
		verdict = "PASS"
	}
	r.logf("    %s  category %s (expected %s), named %d/%d, %.0fs, %d tool calls\n",
		verdict, orDash(res.Score.GotCategory), res.Score.ExpectedCategory,
		res.Score.NamedCount, len(res.Score.Names), res.Seconds, res.ToolCalls)
	return res
}

func orDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}
