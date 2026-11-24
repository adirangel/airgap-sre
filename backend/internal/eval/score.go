// Package eval asks an agent to diagnose each scenario and scores the answer
// against expected.yaml. Scoring is fully deterministic: no LLM judge.
package eval

import (
	"encoding/json"
	"regexp"
	"strings"

	"github.com/adirangel/airgap-sre/backend/internal/scenario"
)

// Diagnosis is the structured block the agent is asked to end its answer with.
type Diagnosis struct {
	RootCauseCategory string            `json:"root_cause_category"`
	Resources         []DiagnosedObject `json:"resources"`
	Summary           string            `json:"summary"`

	Unparsed bool   `json:"-"` // true when no structured block was found
	Raw      string `json:"-"` // the JSON text that was parsed
}

// DiagnosedObject is one resource the agent blames.
type DiagnosedObject struct {
	Kind string `json:"kind"`
	Name string `json:"name"`
}

// NameResult says whether one expected resource was named.
type NameResult struct {
	Kind  string `json:"kind"`
	Name  string `json:"name"`
	Named bool   `json:"named"`
}

// Score is the result for one scenario.
type Score struct {
	ScenarioID       string       `json:"scenario"`
	ExpectedCategory string       `json:"expectedCategory"`
	GotCategory      string       `json:"gotCategory"` // "" when the agent gave no structured answer
	CategoryCorrect  bool         `json:"categoryCorrect"`
	Names            []NameResult `json:"names"`
	NamedCount       int          `json:"namedCount"`
	Structured       bool         `json:"structured"` // the answer ended with a parseable diagnosis block
	Points           float64      `json:"points"`     // 0..1: half for the category, half for the named resources
	Pass             bool         `json:"pass"`       // category correct AND every resource named
}

// ScoreAnswer scores one answer. The rules:
//
//   - Category: taken only from the structured diagnosis block and compared
//     exactly (after normalising case, spaces and dashes). Free text never
//     earns the category, so an answer that lists several causes cannot win.
//   - Resources: each mustName entry counts when its name appears anywhere in
//     the answer as a whole token (case-insensitive). "shop" does not match
//     inside "shop-frontend".
//   - Points = 0.5*category + 0.5*(named/total). Pass needs both in full.
func ScoreAnswer(s scenario.Scenario, answer string) Score {
	d := ExtractDiagnosis(answer)
	sc := Score{
		ScenarioID:       s.ID,
		ExpectedCategory: s.Expected.RootCauseCategory,
		GotCategory:      NormalizeCategory(d.RootCauseCategory),
		Structured:       !d.Unparsed,
	}
	sc.CategoryCorrect = sc.Structured && sc.GotCategory == s.Expected.RootCauseCategory
	for _, r := range s.Expected.MustName {
		named := MentionsName(answer, r.Name)
		if named {
			sc.NamedCount++
		}
		sc.Names = append(sc.Names, NameResult{Kind: r.Kind, Name: r.Name, Named: named})
	}
	if sc.CategoryCorrect {
		sc.Points += 0.5
	}
	if n := len(s.Expected.MustName); n > 0 {
		sc.Points += 0.5 * float64(sc.NamedCount) / float64(n)
	}
	sc.Pass = sc.CategoryCorrect && sc.NamedCount == len(s.Expected.MustName)
	return sc
}

// NormalizeCategory lower-cases and turns spaces and dashes into underscores.
func NormalizeCategory(c string) string {
	c = strings.ToLower(strings.TrimSpace(c))
	c = strings.NewReplacer(" ", "_", "-", "_").Replace(c)
	return c
}

func isNameChar(b byte) bool {
	return b >= 'a' && b <= 'z' || b >= '0' && b <= '9' || b == '-' || b == '_'
}

// MentionsName reports whether name occurs in text as a whole token: the
// characters right before and after it must not be letters, digits, '-' or
// '_'. Matching is case-insensitive.
func MentionsName(text, name string) bool {
	text, name = strings.ToLower(text), strings.ToLower(name)
	if name == "" {
		return false
	}
	for from := 0; ; {
		i := strings.Index(text[from:], name)
		if i < 0 {
			return false
		}
		start := from + i
		end := start + len(name)
		beforeOK := start == 0 || !isNameChar(text[start-1])
		afterOK := end == len(text) || !isNameChar(text[end])
		if beforeOK && afterOK {
			return true
		}
		from = start + 1
	}
}

var fencedRe = regexp.MustCompile("(?s)```[a-zA-Z]*\\s*(\\{.*?\\})\\s*```")

// ExtractDiagnosis finds the last JSON object in the answer that has a
// root_cause_category field. Fenced code blocks are tried first, then any
// balanced {...} object in the text.
func ExtractDiagnosis(answer string) Diagnosis {
	var candidates []string
	for _, m := range fencedRe.FindAllStringSubmatch(answer, -1) {
		candidates = append(candidates, m[1])
	}
	candidates = append(candidates, balancedObjects(answer)...)
	// Prefer the last one in the text: that is the agent's final word.
	best := Diagnosis{Unparsed: true}
	bestPos := -1
	for _, c := range candidates {
		var d Diagnosis
		if err := json.Unmarshal([]byte(c), &d); err != nil || strings.TrimSpace(d.RootCauseCategory) == "" {
			continue
		}
		if pos := strings.LastIndex(answer, c); pos > bestPos {
			d.Raw = c
			best, bestPos = d, pos
		}
	}
	return best
}

// balancedObjects returns every top-level {...} substring, honouring strings.
func balancedObjects(s string) []string {
	var out []string
	depth, start := 0, -1
	inString, escaped := false, false
	for i := 0; i < len(s); i++ {
		ch := s[i]
		if inString {
			switch {
			case escaped:
				escaped = false
			case ch == '\\':
				escaped = true
			case ch == '"':
				inString = false
			}
			continue
		}
		switch ch {
		case '"':
			if depth > 0 {
				inString = true
			}
		case '{':
			if depth == 0 {
				start = i
			}
			depth++
		case '}':
			if depth > 0 {
				depth--
				if depth == 0 && start >= 0 {
					out = append(out, s[start:i+1])
					start = -1
				}
			}
		}
	}
	return out
}
