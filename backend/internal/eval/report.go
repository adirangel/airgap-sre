package eval

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Meta describes one eval run.
type Meta struct {
	Date          string   `json:"date"` // YYYY-MM-DD
	Started       string   `json:"started"`
	Provider      string   `json:"provider"`
	Model         string   `json:"model"`
	KagentVersion string   `json:"kagentVersion"`
	Agent         string   `json:"agent"`
	Tools         []string `json:"tools"`
	Commit        string   `json:"commit,omitempty"`
}

// Report is everything written to docs/results/<date>/.
type Report struct {
	Meta    Meta     `json:"meta"`
	Passed  int      `json:"passed"`
	Total   int      `json:"total"`
	Points  float64  `json:"points"` // sum of per-scenario points
	Results []Result `json:"results"`
}

// NewReport totals the results.
func NewReport(meta Meta, results []Result) Report {
	rep := Report{Meta: meta, Results: results, Total: len(results)}
	for i := range rep.Results {
		r := &rep.Results[i]
		r.Transcript = fmt.Sprintf("transcripts/%02d-%s.md", r.Order, r.ID)
		if r.Score.Pass {
			rep.Passed++
		}
		rep.Points += r.Score.Points
	}
	return rep
}

// ResultsDir picks docs/results/<date>, or <date>-2, <date>-3, ... when a run
// from the same day is already there, so earlier results are never overwritten.
func ResultsDir(base string, now time.Time) string {
	date := now.Format("2006-01-02")
	dir := filepath.Join(base, date)
	for n := 2; ; n++ {
		if _, err := os.Stat(filepath.Join(dir, "results.json")); err != nil {
			return dir
		}
		dir = filepath.Join(base, fmt.Sprintf("%s-%d", date, n))
	}
}

// Write stores results.json, results.md and one transcript (markdown plus the
// raw agent response) per scenario under dir.
func (rep Report) Write(dir string) error {
	if err := os.MkdirAll(filepath.Join(dir, "transcripts"), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(rep, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, "results.json"), append(data, '\n'), 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, "results.md"), []byte(rep.Markdown()), 0o644); err != nil {
		return err
	}
	for _, r := range rep.Results {
		path := filepath.Join(dir, filepath.FromSlash(r.Transcript))
		if err := os.WriteFile(path, []byte(transcriptMarkdown(rep.Meta, r)), 0o644); err != nil {
			return err
		}
		if len(r.Reply.Raw) > 0 {
			raw := r.Reply.Raw
			var pretty bytes.Buffer
			if json.Indent(&pretty, raw, "", "  ") == nil {
				raw = append(pretty.Bytes(), '\n')
			}
			if err := os.WriteFile(strings.TrimSuffix(path, ".md")+".raw.json", raw, 0o644); err != nil {
				return err
			}
		}
	}
	return nil
}

// Table is the markdown results table (also pasted into the README).
func (rep Report) Table() string {
	var b strings.Builder
	b.WriteString("| # | Scenario | Expected category | Agent's category | Resources named | Points | Result | Time | Tool calls |\n")
	b.WriteString("| --- | --- | --- | --- | --- | --- | --- | --- | --- |\n")
	for _, r := range rep.Results {
		verdict := "FAIL"
		switch {
		case r.Score.Pass:
			verdict = "PASS"
		case r.Error != "":
			verdict = "ERROR"
		}
		fmt.Fprintf(&b, "| %d | [`%s`](%s) | `%s` | %s | %d/%d | %.2f | %s | %.0fs | %d |\n",
			r.Order, r.ID, r.Transcript, r.Score.ExpectedCategory, codeOrDash(r.Score.GotCategory),
			r.Score.NamedCount, len(r.Score.Names), r.Score.Points, verdict, r.Seconds, r.ToolCalls)
	}
	fmt.Fprintf(&b, "\n**%d/%d passed**, %.2f/%d points.\n", rep.Passed, rep.Total, rep.Points, rep.Total)
	return b.String()
}

// Markdown is results.md.
func (rep Report) Markdown() string {
	var b strings.Builder
	m := rep.Meta
	fmt.Fprintf(&b, "# Eval results — %s\n\n", m.Date)
	fmt.Fprintf(&b, "| | |\n| --- | --- |\n")
	fmt.Fprintf(&b, "| Started | %s |\n", m.Started)
	fmt.Fprintf(&b, "| Model | `%s` (%s) |\n", m.Model, m.Provider)
	fmt.Fprintf(&b, "| Agent | `%s` on kagent %s |\n", m.Agent, m.KagentVersion)
	fmt.Fprintf(&b, "| Tools | %s |\n", codeList(m.Tools))
	if m.Commit != "" {
		fmt.Fprintf(&b, "| Commit | `%s` |\n", m.Commit)
	}
	b.WriteString("\n")
	b.WriteString(rep.Table())
	b.WriteString("\nScoring is deterministic (no LLM judge): the category is read from the JSON block that ends the\n" +
		"agent's answer and must match exactly; each expected resource name must appear in the answer as a whole\n" +
		"token. Points = 0.5 for the category + 0.5 × (resources named / expected). PASS needs both in full.\n")

	var misses []Result
	for _, r := range rep.Results {
		if !r.Score.Pass {
			misses = append(misses, r)
		}
	}
	if len(misses) > 0 {
		b.WriteString("\n## Misses\n\n")
		for _, r := range misses {
			fmt.Fprintf(&b, "- **%s** — ", r.ID)
			var why []string
			if r.Error != "" {
				why = append(why, "error: "+r.Error)
			}
			if !r.Score.Structured {
				why = append(why, "no parseable diagnosis block at the end of the answer")
			} else if !r.Score.CategoryCorrect {
				why = append(why, fmt.Sprintf("category `%s`, expected `%s`", r.Score.GotCategory, r.Score.ExpectedCategory))
			}
			for _, n := range r.Score.Names {
				if !n.Named {
					why = append(why, fmt.Sprintf("did not name %s `%s`", n.Kind, n.Name))
				}
			}
			fmt.Fprintf(&b, "%s. [transcript](%s)\n", strings.Join(why, "; "), r.Transcript)
		}
	}
	return b.String()
}

func transcriptMarkdown(meta Meta, r Result) string {
	var b strings.Builder
	verdict := "FAIL"
	if r.Score.Pass {
		verdict = "PASS"
	}
	fmt.Fprintf(&b, "# %d. %s — %s\n\n", r.Order, r.ID, r.Title)
	fmt.Fprintf(&b, "- Run: %s, model `%s` (%s), agent `%s`\n", meta.Started, meta.Model, meta.Provider, meta.Agent)
	fmt.Fprintf(&b, "- Result: **%s**, %.2f points\n", verdict, r.Score.Points)
	fmt.Fprintf(&b, "- Category: expected `%s`, agent said %s\n", r.Score.ExpectedCategory, codeOrDash(r.Score.GotCategory))
	for _, n := range r.Score.Names {
		mark := "missing"
		if n.Named {
			mark = "named"
		}
		fmt.Fprintf(&b, "- Must name %s `%s`: %s\n", n.Kind, n.Name, mark)
	}
	fmt.Fprintf(&b, "- Agent time: %.0fs, tool calls: %d, tokens: %d\n", r.Seconds, r.ToolCalls, r.Tokens)
	if r.Error != "" {
		fmt.Fprintf(&b, "- Error: %s\n", r.Error)
	}
	fmt.Fprintf(&b, "\n## Symptom given to the agent\n\n> %s\n", r.Symptom)
	b.WriteString("\n## Final answer\n\n")
	if strings.TrimSpace(r.Answer) == "" {
		b.WriteString("_(empty)_\n")
	} else {
		b.WriteString(r.Answer + "\n")
	}
	b.WriteString("\n## Full transcript\n")
	if len(r.Reply.Steps) == 0 {
		b.WriteString("\n_(no steps recorded)_\n")
	}
	for i, s := range r.Reply.Steps {
		switch s.Kind {
		case "tool_call":
			fmt.Fprintf(&b, "\n### %d. tool call: `%s`\n\n", i+1, s.Tool)
			args := "{}"
			if len(s.Args) > 0 {
				args = string(s.Args)
			}
			b.WriteString(fence("json", args))
		case "tool_result":
			fmt.Fprintf(&b, "\n### %d. tool result: `%s`\n\n", i+1, s.Tool)
			b.WriteString(fence("text", s.Result))
		default:
			fmt.Fprintf(&b, "\n### %d. %s\n\n%s\n", i+1, s.Role, s.Text)
		}
	}
	return b.String()
}

// fence wraps text in a code fence that is longer than any backtick run
// inside it, so tool output cannot break out of the block.
func fence(lang, text string) string {
	ticks := "```"
	for strings.Contains(text, ticks) {
		ticks += "`"
	}
	return ticks + lang + "\n" + strings.TrimRight(text, "\n") + "\n" + ticks + "\n"
}

func codeOrDash(s string) string {
	if s == "" {
		return "—"
	}
	return "`" + s + "`"
}

func codeList(items []string) string {
	out := make([]string, len(items))
	for i, s := range items {
		out[i] = "`" + s + "`"
	}
	return strings.Join(out, ", ")
}
