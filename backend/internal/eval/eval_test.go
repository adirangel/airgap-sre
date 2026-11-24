package eval

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/adirangel/airgap-sre/backend/internal/kube"
	"github.com/adirangel/airgap-sre/backend/internal/scenario"
)

// A trimmed copy of what kagent 0.10.7 returns for message/send.
const sampleResponse = `{
  "jsonrpc": "2.0", "id": "1",
  "result": {
    "kind": "task", "id": "t1", "contextId": "ctx-1",
    "status": {"state": "completed"},
    "artifacts": [{"artifactId": "a1", "parts": [{"kind": "text", "text": "Final answer.\n` + "```json\\n{\\\"root_cause_category\\\": \\\"oom_killed\\\"}\\n```" + `"}]}],
    "history": [
      {"role": "user", "parts": [{"kind": "text", "text": "Why is it down?"}]},
      {"role": "user", "parts": [{"kind": "text", "text": "Why is it down?"}]},
      {"role": "agent", "metadata": {"adk_usage_metadata": {"totalTokenCount": 100}},
       "parts": [{"kind": "data", "data": {"id": "c1", "name": "k8s_get_resources", "args": {"namespace": "scn-x", "resource_type": "pod"}}, "metadata": {"adk_type": "function_call"}}]},
      {"role": "agent",
       "parts": [{"kind": "data", "data": {"id": "c1", "name": "k8s_get_resources", "response": {"output": {"output": "NAME READY\nweb 0/1"}}}, "metadata": {"adk_type": "function_response"}}]},
      {"role": "agent", "metadata": {"adk_usage_metadata": {"totalTokenCount": 250}},
       "parts": [{"kind": "text", "text": "Final answer."}]}
    ]
  }
}`

func TestParseReply(t *testing.T) {
	r, err := ParseReply([]byte(sampleResponse))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(r.Answer, "Final answer.") || !strings.Contains(r.Answer, "oom_killed") {
		t.Errorf("answer = %q", r.Answer)
	}
	if r.State != "completed" || r.ContextID != "ctx-1" || r.ToolCalls != 1 || r.Tokens != 350 {
		t.Errorf("reply = %+v", r)
	}
	// user (deduplicated), tool_call, tool_result, agent text
	if len(r.Steps) != 4 {
		t.Fatalf("want 4 steps, got %d: %+v", len(r.Steps), r.Steps)
	}
	if r.Steps[1].Kind != "tool_call" || r.Steps[1].Tool != "k8s_get_resources" || !strings.Contains(string(r.Steps[1].Args), "scn-x") {
		t.Errorf("tool call step = %+v", r.Steps[1])
	}
	if r.Steps[2].Kind != "tool_result" || r.Steps[2].Result != "NAME READY\nweb 0/1" {
		t.Errorf("tool result step = %+v", r.Steps[2])
	}
}

func TestParseReplyErrors(t *testing.T) {
	cases := map[string]string{
		"rpc error":  `{"jsonrpc":"2.0","error":{"code":-32600,"message":"invalid request"}}`,
		"no result":  `{"jsonrpc":"2.0"}`,
		"not json":   `<html>`,
		"task fails": `{"result":{"kind":"task","status":{"state":"failed"},"artifacts":[]}}`,
	}
	for name, body := range cases {
		if _, err := ParseReply([]byte(body)); err == nil {
			t.Errorf("%s: expected an error", name)
		}
	}
}

func TestA2AClientSend(t *testing.T) {
	var gotPath string
	var gotBody rpcRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &gotBody)
		_, _ = w.Write([]byte(sampleResponse))
	}))
	defer srv.Close()

	reply, err := A2AClient{BaseURL: srv.URL}.Send(context.Background(), "kagent", "sre-agent", "Why is it down?")
	if err != nil {
		t.Fatal(err)
	}
	if gotPath != "/api/a2a/kagent/sre-agent/" {
		t.Errorf("path = %s", gotPath)
	}
	m := gotBody.Params.Message
	if gotBody.Method != "message/send" || gotBody.JSONRPC != "2.0" || m.Role != "user" || m.MessageID == "" ||
		len(m.Parts) != 1 || m.Parts[0].Text != "Why is it down?" {
		t.Errorf("request = %+v", gotBody)
	}
	if reply.ToolCalls != 1 {
		t.Errorf("reply = %+v", reply)
	}
}

func TestA2AClientHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "boom", http.StatusBadGateway)
	}))
	defer srv.Close()
	if _, err := (A2AClient{BaseURL: srv.URL}).Send(context.Background(), "kagent", "x", "q"); err == nil {
		t.Fatal("expected an error for HTTP 502")
	}
}

// fakeCluster always shows the failure as present and records what happened.
type fakeCluster struct {
	log        []string
	unhealthy  bool // when false, verify can never succeed
	failInject bool
}

func (f *fakeCluster) Pods(context.Context, string, string) ([]kube.Pod, error) {
	if !f.unhealthy {
		return []kube.Pod{{Status: kube.PodStatus{Phase: "Running"}}}, nil
	}
	return []kube.Pod{{Status: kube.PodStatus{Phase: "Pending",
		Conditions: []kube.Condition{{Type: "PodScheduled", Status: "False", Reason: "Unschedulable"}}}}}, nil
}

func (f *fakeCluster) ServiceEndpointAddresses(context.Context, string, string) ([]string, error) {
	return nil, nil
}

func (f *fakeCluster) EnsureNamespace(_ context.Context, name string, _ map[string]string) error {
	f.log = append(f.log, "ns "+name)
	return nil
}

func (f *fakeCluster) ApplyPath(_ context.Context, ns, _ string) error {
	if f.failInject {
		return errors.New("apply refused")
	}
	f.log = append(f.log, "apply "+ns)
	return nil
}

func (f *fakeCluster) DeleteNamespace(_ context.Context, name string, _ time.Duration) error {
	f.log = append(f.log, "delete "+name)
	return nil
}

func testScenario(t *testing.T, timeoutSeconds int) scenario.Scenario {
	t.Helper()
	return scenario.Scenario{
		ID: "pending-unschedulable", Order: 4, Title: "Pending", Namespace: "scn-batch",
		Symptom: "The batch-worker pods in namespace scn-batch never start.",
		Verify: scenario.Verify{TimeoutSeconds: timeoutSeconds,
			Checks: []scenario.Check{{Type: scenario.CheckPodUnschedulable, Selector: "app=batch-worker"}}},
		Dir: t.TempDir(),
		Expected: scenario.Expected{RootCauseCategory: "unschedulable",
			MustName: []scenario.Resource{{Kind: "Deployment", Name: "batch-worker"}, {Kind: "NodeLabel", Name: "disktype"}}},
	}
}

const goodAnswer = "Deployment batch-worker needs a node with label disktype=ssd; no node has it.\n" +
	"```json\n{\"root_cause_category\": \"unschedulable\"}\n```"

func TestRunnerHappyPath(t *testing.T) {
	c := &fakeCluster{unhealthy: true}
	var asked string
	r := Runner{Cluster: c, Out: io.Discard, Timeout: time.Minute, Interval: time.Millisecond,
		Ask: func(_ context.Context, symptom string) (Reply, error) {
			asked = symptom
			return Reply{Answer: goodAnswer, ToolCalls: 3, Tokens: 900}, nil
		}}
	s := testScenario(t, 5)
	results := r.Run(context.Background(), []scenario.Scenario{s})
	if len(results) != 1 {
		t.Fatalf("results = %+v", results)
	}
	res := results[0]
	if !res.Score.Pass || res.Error != "" || res.ToolCalls != 3 || res.Tokens != 900 {
		t.Errorf("result = %+v", res)
	}
	// The agent must be told the symptom and nothing else.
	if asked != s.Symptom {
		t.Errorf("agent was asked %q, want only the symptom", asked)
	}
	// reset, inject (ns + apply), then reset again afterwards.
	want := "delete scn-batch,ns scn-batch,apply scn-batch,delete scn-batch"
	if got := strings.Join(c.log, ","); got != want {
		t.Errorf("cluster calls = %s, want %s", got, want)
	}
}

func TestRunnerKeepSkipsFinalReset(t *testing.T) {
	c := &fakeCluster{unhealthy: true}
	r := Runner{Cluster: c, Out: io.Discard, Timeout: time.Minute, Interval: time.Millisecond, Keep: true,
		Ask: func(context.Context, string) (Reply, error) { return Reply{Answer: goodAnswer}, nil }}
	r.Run(context.Background(), []scenario.Scenario{testScenario(t, 5)})
	if got := strings.Join(c.log, ","); got != "delete scn-batch,ns scn-batch,apply scn-batch" {
		t.Errorf("cluster calls = %s", got)
	}
}

func TestRunnerRecordsFailures(t *testing.T) {
	never := func(context.Context, string) (Reply, error) {
		t.Error("the agent must not be asked when setup failed")
		return Reply{}, nil
	}

	// Inject fails.
	r := Runner{Cluster: &fakeCluster{failInject: true}, Out: io.Discard, Timeout: time.Minute, Interval: time.Millisecond, Ask: never}
	res := r.Run(context.Background(), []scenario.Scenario{testScenario(t, 5)})[0]
	if !strings.HasPrefix(res.Error, "inject:") || res.Score.Pass || res.Score.Points != 0 {
		t.Errorf("inject failure: %+v", res)
	}

	// The failure never shows up.
	r = Runner{Cluster: &fakeCluster{unhealthy: false}, Out: io.Discard, Timeout: time.Minute, Interval: 50 * time.Millisecond, Ask: never}
	res = r.Run(context.Background(), []scenario.Scenario{testScenario(t, 1)})[0]
	if !strings.HasPrefix(res.Error, "verify") || res.Score.Pass {
		t.Errorf("verify failure: %+v", res)
	}

	// The agent call fails: recorded, scored as a miss, and the run goes on.
	r = Runner{Cluster: &fakeCluster{unhealthy: true}, Out: io.Discard, Timeout: time.Minute, Interval: time.Millisecond,
		Ask: func(context.Context, string) (Reply, error) { return Reply{}, errors.New("model timed out") }}
	results := r.Run(context.Background(), []scenario.Scenario{testScenario(t, 5), testScenario(t, 5)})
	if len(results) != 2 || !strings.Contains(results[0].Error, "model timed out") || results[0].Score.Pass {
		t.Errorf("agent failure: %+v", results)
	}
}

func TestReportWrite(t *testing.T) {
	s := testScenario(t, 5)
	reply, err := ParseReply([]byte(sampleResponse))
	if err != nil {
		t.Fatal(err)
	}
	pass := Result{Order: 4, ID: s.ID, Title: s.Title, Symptom: s.Symptom, Answer: goodAnswer, Score: ScoreAnswer(s, goodAnswer), Seconds: 12, ToolCalls: 1, Reply: reply}
	missAnswer := "Something with batch-worker.\n```json\n{\"root_cause_category\": \"node_failure\"}\n```"
	miss := Result{Order: 5, ID: "other", Title: "Other", Symptom: "x", Answer: missAnswer, Score: ScoreAnswer(s, missAnswer)}
	rep := NewReport(Meta{Date: "2026-01-02", Started: "2026-01-02 10:00 UTC", Provider: "Ollama", Model: "m:1", KagentVersion: "0.0.1", Agent: "sre-agent", Tools: []string{"k8s_get_resources"}}, []Result{pass, miss})

	if rep.Passed != 1 || rep.Total != 2 || rep.Points != 1.25 {
		t.Fatalf("totals: passed=%d total=%d points=%v", rep.Passed, rep.Total, rep.Points)
	}
	dir := filepath.Join(t.TempDir(), "2026-01-02")
	if err := rep.Write(dir); err != nil {
		t.Fatal(err)
	}
	read := func(rel string) string {
		data, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(rel)))
		if err != nil {
			t.Fatal(err)
		}
		return string(data)
	}
	md := read("results.md")
	for _, want := range []string{"**1/2 passed**", "`m:1` (Ollama)", "| 4 | [`pending-unschedulable`](transcripts/04-pending-unschedulable.md)", "In JSON block", "PASS", "## Misses",
		"category `node_failure`, expected `unschedulable`", "did not name NodeLabel `disktype`"} {
		if !strings.Contains(md, want) {
			t.Errorf("results.md missing %q:\n%s", want, md)
		}
	}
	tr := read("transcripts/04-pending-unschedulable.md")
	for _, want := range []string{"## Symptom given to the agent", s.Symptom, "## Final answer", "tool call: `k8s_get_resources`", "tool result: `k8s_get_resources`", "web 0/1"} {
		if !strings.Contains(tr, want) {
			t.Errorf("transcript missing %q:\n%s", want, tr)
		}
	}
	if !json.Valid([]byte(read("results.json"))) || !json.Valid([]byte(read("transcripts/04-pending-unschedulable.raw.json"))) {
		t.Error("results.json and the raw transcript must be valid JSON")
	}

	// A second run on the same day must not overwrite the first.
	base := filepath.Dir(dir)
	day := time.Date(2026, 1, 2, 12, 0, 0, 0, time.UTC)
	if got := ResultsDir(base, day); got != filepath.Join(base, "2026-01-02-2") {
		t.Errorf("ResultsDir = %s", got)
	}
	if got := ResultsDir(base, day.AddDate(0, 0, 1)); got != filepath.Join(base, "2026-01-03") {
		t.Errorf("ResultsDir = %s", got)
	}
}

func TestFenceCannotBeBrokenByToolOutput(t *testing.T) {
	out := fence("text", "before\n```\ninside\n```\nafter")
	if !strings.HasPrefix(out, "````text\n") || !strings.HasSuffix(out, "\n````\n") {
		t.Errorf("fence = %q", out)
	}
}
