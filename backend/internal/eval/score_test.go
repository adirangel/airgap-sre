package eval

import (
	"os"
	"testing"

	"github.com/adirangel/airgap-sre/backend/internal/repo"
	"github.com/adirangel/airgap-sre/backend/internal/scenario"
)

func demo() scenario.Scenario {
	return scenario.Scenario{
		ID: "service-selector-mismatch",
		Expected: scenario.Expected{
			RootCauseCategory: "service_selector_mismatch",
			MustName: []scenario.Resource{
				{Kind: "Service", Name: "storefront"},
				{Kind: "Deployment", Name: "shop-frontend"},
			},
		},
	}
}

const perfect = "The Service `storefront` selects `app=shop-front`, but the pods of Deployment " +
	"`shop-frontend` are labelled `app=shop-frontend`, so the Service has no endpoints.\n\n" +
	"```json\n" +
	`{"root_cause_category": "service_selector_mismatch", "resources": [{"kind": "Service", "name": "storefront"}, {"kind": "Deployment", "name": "shop-frontend"}], "summary": "selector mismatch"}` +
	"\n```\n"

func TestScorePerfectAnswer(t *testing.T) {
	sc := ScoreAnswer(demo(), perfect)
	if !sc.Pass || sc.Points != 1 || !sc.CategoryCorrect || sc.NamedCount != 2 || sc.InBlockCount != 2 || !sc.Structured {
		t.Fatalf("perfect answer scored %+v", sc)
	}
}

// A name that is only mentioned in the prose still passes, but the stricter
// in-block count shows it.
func TestScoreInBlockIsStricter(t *testing.T) {
	answer := "The storefront Service selects app=shop-front but the pods carry app=shop-frontend.\n" +
		"```json\n{\"root_cause_category\": \"service_selector_mismatch\", \"resources\": [{\"kind\": \"Service\", \"name\": \"storefront\"}, {\"kind\": \"Pod\", \"name\": \"shop-frontend-69df64b78f-qpjpf\"}]}\n```"
	sc := ScoreAnswer(demo(), answer)
	if !sc.Pass || sc.NamedCount != 2 || sc.InBlockCount != 1 {
		t.Fatalf("got %+v", sc)
	}
	if !sc.Names[0].InBlock || sc.Names[1].InBlock {
		t.Fatalf("names = %+v", sc.Names)
	}
}

func TestScoreIsDeterministic(t *testing.T) {
	first := ScoreAnswer(demo(), perfect)
	for i := 0; i < 50; i++ {
		if got := ScoreAnswer(demo(), perfect); got.Points != first.Points || got.Pass != first.Pass || got.GotCategory != first.GotCategory {
			t.Fatalf("run %d differs: %+v vs %+v", i, got, first)
		}
	}
}

func TestScoreWrongCategory(t *testing.T) {
	answer := "Service storefront and Deployment shop-frontend.\n" +
		"```json\n{\"root_cause_category\": \"network_policy\", \"resources\": []}\n```"
	sc := ScoreAnswer(demo(), answer)
	if sc.Pass || sc.CategoryCorrect || sc.Points != 0.5 || sc.GotCategory != "network_policy" {
		t.Fatalf("got %+v", sc)
	}
}

func TestScoreMissingResource(t *testing.T) {
	answer := "The storefront Service selects nothing.\n" +
		"```json\n{\"root_cause_category\": \"service_selector_mismatch\"}\n```"
	sc := ScoreAnswer(demo(), answer)
	if sc.Pass || !sc.CategoryCorrect || sc.NamedCount != 1 || sc.Points != 0.75 {
		t.Fatalf("got %+v", sc)
	}
	if sc.Names[0].Named != true || sc.Names[1].Named != false {
		t.Fatalf("names = %+v", sc.Names)
	}
}

// Free text must never earn the category: an answer that mentions every
// category would otherwise always win.
func TestScoreNoStructuredBlock(t *testing.T) {
	answer := "It is a service_selector_mismatch between storefront and shop-frontend, or maybe oom_killed, or bad_config."
	sc := ScoreAnswer(demo(), answer)
	if sc.Structured || sc.CategoryCorrect || sc.Pass || sc.GotCategory != "" || sc.Points != 0.5 {
		t.Fatalf("got %+v", sc)
	}
}

func TestScoreEmptyAnswer(t *testing.T) {
	sc := ScoreAnswer(demo(), "")
	if sc.Pass || sc.Points != 0 || sc.Structured {
		t.Fatalf("got %+v", sc)
	}
}

func TestNormalizeCategory(t *testing.T) {
	for in, want := range map[string]string{
		"OOM_Killed":                 "oom_killed",
		" oom-killed ":               "oom_killed",
		"Service Selector Mismatch":  "service_selector_mismatch",
		"service_selector_mismatch ": "service_selector_mismatch",
	} {
		if got := NormalizeCategory(in); got != want {
			t.Errorf("NormalizeCategory(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestMentionsName(t *testing.T) {
	yes := [][2]string{
		{"the Service `shop` has no endpoints", "shop"},
		{"service/shop has no endpoints", "shop"},
		{"shop.scn-shop.svc.cluster.local", "shop"},
		{"Look at shop.", "shop"},
		{"(shop)", "shop"},
		{"SHOP is broken", "shop"},
		{"image docker.io/library/redis:7.4-alpine was not found", "redis:7.4-alpine"},
		{"key DB_PORT=5432a", "DB_PORT"},
		{"the ConfigMap checkout-config", "checkout-config"},
		{"shop-frontend and then shop", "shop"},
	}
	for _, c := range yes {
		if !MentionsName(c[0], c[1]) {
			t.Errorf("MentionsName(%q, %q) = false, want true", c[0], c[1])
		}
	}
	no := [][2]string{
		{"the Deployment shop-frontend is healthy", "shop"},
		{"workshop is closed", "shop"},
		{"my_shop", "shop"},
		{"shops", "shop"},
		{"redis:7.4-alpine3.20", "redis:7.4-alpine"},
		{"checkout-config-v2", "checkout-config"},
		{"anything", ""},
	}
	for _, c := range no {
		if MentionsName(c[0], c[1]) {
			t.Errorf("MentionsName(%q, %q) = true, want false", c[0], c[1])
		}
	}
}

func TestExtractDiagnosis(t *testing.T) {
	// The last block wins, also without a code fence, and braces inside
	// strings do not confuse the parser.
	answer := "First guess:\n```json\n{\"root_cause_category\": \"bad_config\"}\n```\n" +
		"After checking {not json} more:\n" +
		`{"root_cause_category": "oom_killed", "summary": "limit {32Mi} too low", "resources": [{"kind": "Deployment", "name": "x"}]}`
	d := ExtractDiagnosis(answer)
	if d.Unparsed || d.RootCauseCategory != "oom_killed" || len(d.Resources) != 1 || d.Resources[0].Name != "x" {
		t.Fatalf("got %+v", d)
	}
	for _, bad := range []string{"", "no json here", "{\"other\": 1}", "```json\n{broken\n```"} {
		if d := ExtractDiagnosis(bad); !d.Unparsed {
			t.Errorf("ExtractDiagnosis(%q) should be unparsed, got %+v", bad, d)
		}
	}
}

// A model answer written from each real expected.yaml must pass, and the same
// answer must fail against every other scenario. This guards both the scorer
// and the scenario definitions.
func TestRealScenariosAreScorable(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	p, err := repo.Find(wd)
	if err != nil {
		t.Fatal(err)
	}
	all, err := scenario.LoadAll(p.Infra("scenarios"))
	if err != nil {
		t.Fatal(err)
	}
	answers := map[string]string{}
	for _, s := range all {
		text := "Root cause found.\n"
		for _, r := range s.Expected.MustName {
			text += "- " + r.Kind + " " + r.Name + "\n"
		}
		text += "```json\n{\"root_cause_category\": \"" + s.Expected.RootCauseCategory + "\"}\n```\n"
		answers[s.ID] = text
	}
	for _, s := range all {
		for id, answer := range answers {
			sc := ScoreAnswer(s, answer)
			if id == s.ID && !sc.Pass {
				t.Errorf("%s: its own model answer does not pass: %+v", s.ID, sc)
			}
			if id != s.ID && sc.Pass {
				t.Errorf("%s: the model answer of %s passes too — expected.yaml is not specific enough", s.ID, id)
			}
		}
	}
}
