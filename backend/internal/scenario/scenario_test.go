package scenario

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/adirangel/airgap-sre/backend/internal/config"
	"github.com/adirangel/airgap-sre/backend/internal/kube"
	"github.com/adirangel/airgap-sre/backend/internal/repo"
)

func repoPaths(t *testing.T) repo.Paths {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	p, err := repo.Find(wd)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func loadReal(t *testing.T) []Scenario {
	t.Helper()
	all, err := LoadAll(repoPaths(t).Infra("scenarios"))
	if err != nil {
		t.Fatal(err)
	}
	return all
}

// The five scenarios required by M2 must exist, load and be well-formed.
func TestRealScenariosLoad(t *testing.T) {
	all := loadReal(t)
	want := map[string]string{
		"crashloop-bad-config":      "bad_config",
		"oomkilled":                 "oom_killed",
		"imagepull-not-mirrored":    "image_pull_failure",
		"pending-unschedulable":     "unschedulable",
		"service-selector-mismatch": "service_selector_mismatch",
	}
	if len(all) != len(want) {
		t.Fatalf("want %d scenarios, got %d", len(want), len(all))
	}
	for i, s := range all {
		if s.Order != i+1 {
			t.Errorf("%s: order %d, want %d (scenarios must be numbered 1..n)", s.ID, s.Order, i+1)
		}
		if want[s.ID] != s.Expected.RootCauseCategory {
			t.Errorf("%s: category %q, want %q", s.ID, s.Expected.RootCauseCategory, want[s.ID])
		}
		if !strings.Contains(s.Symptom, s.Namespace) {
			t.Errorf("%s: the symptom must tell the agent which namespace to look at", s.ID)
		}
		// The symptom is all the agent is told; it must not give the answer away.
		if leaks(s.Symptom, s.ID) || leaks(s.Symptom, s.Expected.RootCauseCategory) {
			t.Errorf("%s: the symptom leaks words from the id or the root-cause category: %q", s.ID, s.Symptom)
		}
	}
}

// The air gap makes this a correctness rule: every image a scenario uses must
// be mirrored, unless the scenario declares it as deliberately un-mirrored.
func TestScenarioImagesAreMirroredOrDeclared(t *testing.T) {
	p := repoPaths(t)
	images, err := config.LoadImages(p.Infra("images.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	mirrored := map[string]bool{}
	for _, i := range images.Group("base") {
		mirrored[i.Ref] = true
	}
	for _, s := range loadReal(t) {
		used, err := s.ManifestImages()
		if err != nil {
			t.Fatal(err)
		}
		if len(used) == 0 {
			t.Errorf("%s: no images found in manifests.yaml", s.ID)
		}
		declared := map[string]bool{}
		for _, u := range s.UnmirroredImages {
			declared[u] = true
			if mirrored[u] {
				t.Errorf("%s: %s is declared un-mirrored but is listed in images.yaml", s.ID, u)
			}
		}
		for _, img := range used {
			if !mirrored[img] && !declared[img] {
				t.Errorf("%s: image %s is neither in images.yaml (group base) nor declared in unmirroredImages", s.ID, img)
			}
			delete(declared, img)
		}
		for u := range declared {
			t.Errorf("%s: unmirroredImages lists %s but manifests.yaml does not use it", s.ID, u)
		}
	}
}

func writeScenario(t *testing.T, id, scenarioYAML, expectedYAML string) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), id)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for name, content := range map[string]string{"scenario.yaml": scenarioYAML, "expected.yaml": expectedYAML, "manifests.yaml": "kind: ConfigMap\n"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

const goodScenario = `id: demo
order: 1
title: Demo
description: A demo.
namespace: scn-app
symptom: Something in scn-app is broken.
verify:
  timeoutSeconds: 10
  checks:
    - type: podUnschedulable
      selector: app=demo
`

const goodExpected = `rootCauseCategory: unschedulable
mustName:
  - kind: Deployment
    name: demo
`

func TestLoadValidation(t *testing.T) {
	if _, err := Load(writeScenario(t, "demo", goodScenario, goodExpected)); err != nil {
		t.Fatalf("valid scenario rejected: %v", err)
	}
	cases := map[string][2]string{
		"wrong namespace":   {strings.Replace(goodScenario, "namespace: scn-app", "namespace: default", 1), goodExpected},
		"leaky namespace":   {strings.Replace(goodScenario, "namespace: scn-app", "namespace: scn-unschedulable", 1), goodExpected},
		"unknown category":  {goodScenario, strings.Replace(goodExpected, "unschedulable", "gremlins", 1)},
		"category unknown":  {goodScenario, strings.Replace(goodExpected, "unschedulable", "unknown", 1)},
		"no mustName":       {goodScenario, "rootCauseCategory: unschedulable\n"},
		"unknown check":     {strings.Replace(goodScenario, "podUnschedulable", "podHaunted", 1), goodExpected},
		"check missing arg": {strings.Replace(goodScenario, "      selector: app=demo\n", "", 1), goodExpected},
		"unknown field":     {goodScenario + "surprise: true\n", goodExpected},
		"id != dir":         {strings.Replace(goodScenario, "id: demo", "id: other", 1), goodExpected},
	}
	for name, c := range cases {
		if _, err := Load(writeScenario(t, "demo", c[0], c[1])); err == nil {
			t.Errorf("%s: expected a validation error", name)
		}
	}
}

// fakeCluster serves canned pods and endpoints and records mutations.
type fakeCluster struct {
	pods      map[string][]kube.Pod // by selector
	endpoints map[string][]string   // by service
	applied   []string
	deleted   []string
	labels    map[string]string
}

func (f *fakeCluster) Pods(_ context.Context, _, selector string) ([]kube.Pod, error) {
	return f.pods[selector], nil
}

func (f *fakeCluster) ServiceEndpointAddresses(_ context.Context, _, service string) ([]string, error) {
	return f.endpoints[service], nil
}

func (f *fakeCluster) EnsureNamespace(_ context.Context, name string, labels map[string]string) error {
	f.applied = append(f.applied, "ns:"+name)
	f.labels = labels
	return nil
}

func (f *fakeCluster) ApplyPath(_ context.Context, namespace, path string) error {
	f.applied = append(f.applied, namespace+":"+filepath.Base(path))
	return nil
}

func (f *fakeCluster) DeleteNamespace(_ context.Context, name string, _ time.Duration) error {
	f.deleted = append(f.deleted, name)
	return nil
}

func pod(name, phase string, mods ...func(*kube.Pod)) kube.Pod {
	p := kube.Pod{Metadata: kube.Meta{Name: name}, Status: kube.PodStatus{Phase: phase}}
	for _, m := range mods {
		m(&p)
	}
	return p
}

func waiting(reason string) func(*kube.Pod) {
	return func(p *kube.Pod) {
		p.Status.ContainerStatuses = append(p.Status.ContainerStatuses,
			kube.ContainerStatus{Name: "c", State: kube.ContainerState{Waiting: &kube.StateDetail{Reason: reason}}})
	}
}

func lastTerminated(reason string) func(*kube.Pod) {
	return func(p *kube.Pod) {
		p.Status.ContainerStatuses = append(p.Status.ContainerStatuses,
			kube.ContainerStatus{Name: "c", LastState: kube.ContainerState{Terminated: &kube.StateDetail{Reason: reason, ExitCode: 137}}})
	}
}

func condition(typ, status, reason string) func(*kube.Pod) {
	return func(p *kube.Pod) {
		p.Status.Conditions = append(p.Status.Conditions, kube.Condition{Type: typ, Status: status, Reason: reason})
	}
}

func TestEvalChecks(t *testing.T) {
	cases := []struct {
		name    string
		check   Check
		cluster *fakeCluster
		want    bool
	}{
		{"waiting matches", Check{Type: CheckContainerWaiting, Selector: "a", Reasons: []string{"CrashLoopBackOff"}},
			&fakeCluster{pods: map[string][]kube.Pod{"a": {pod("p", "Running", waiting("CrashLoopBackOff"))}}}, true},
		{"waiting other reason", Check{Type: CheckContainerWaiting, Selector: "a", Reasons: []string{"CrashLoopBackOff"}},
			&fakeCluster{pods: map[string][]kube.Pod{"a": {pod("p", "Pending", waiting("ContainerCreating"))}}}, false},
		{"waiting no pods", Check{Type: CheckContainerWaiting, Selector: "a", Reasons: []string{"CrashLoopBackOff"}},
			&fakeCluster{}, false},
		{"terminated via lastState", Check{Type: CheckContainerTerminated, Selector: "a", Reasons: []string{"OOMKilled"}},
			&fakeCluster{pods: map[string][]kube.Pod{"a": {pod("p", "Running", lastTerminated("OOMKilled"))}}}, true},
		{"terminated wrong reason", Check{Type: CheckContainerTerminated, Selector: "a", Reasons: []string{"OOMKilled"}},
			&fakeCluster{pods: map[string][]kube.Pod{"a": {pod("p", "Running", lastTerminated("Error"))}}}, false},
		{"unschedulable", Check{Type: CheckPodUnschedulable, Selector: "a"},
			&fakeCluster{pods: map[string][]kube.Pod{"a": {pod("p", "Pending", condition("PodScheduled", "False", "Unschedulable"))}}}, true},
		{"scheduled", Check{Type: CheckPodUnschedulable, Selector: "a"},
			&fakeCluster{pods: map[string][]kube.Pod{"a": {pod("p", "Running", condition("PodScheduled", "True", ""))}}}, false},
		{"pods ready", Check{Type: CheckPodsReady, Selector: "a"},
			&fakeCluster{pods: map[string][]kube.Pod{"a": {pod("p", "Running", condition("Ready", "True", "")), pod("q", "Running", condition("Ready", "True", ""))}}}, true},
		{"one pod not ready", Check{Type: CheckPodsReady, Selector: "a"},
			&fakeCluster{pods: map[string][]kube.Pod{"a": {pod("p", "Running", condition("Ready", "True", "")), pod("q", "Running", condition("Ready", "False", ""))}}}, false},
		{"service without endpoints", Check{Type: CheckServiceNoEndpoints, Service: "s"},
			&fakeCluster{}, true},
		{"service with endpoints", Check{Type: CheckServiceNoEndpoints, Service: "s"},
			&fakeCluster{endpoints: map[string][]string{"s": {"10.0.0.1"}}}, false},
	}
	for _, c := range cases {
		got, detail, err := evalCheck(context.Background(), c.cluster, "ns", c.check)
		if err != nil {
			t.Errorf("%s: %v", c.name, err)
			continue
		}
		if got != c.want {
			t.Errorf("%s: got %v want %v (%s)", c.name, got, c.want, detail)
		}
		if detail == "" {
			t.Errorf("%s: empty detail", c.name)
		}
	}
}

func TestInjectVerifyReset(t *testing.T) {
	s, err := Load(writeScenario(t, "demo", goodScenario, goodExpected))
	if err != nil {
		t.Fatal(err)
	}
	c := &fakeCluster{pods: map[string][]kube.Pod{"app=demo": {pod("p", "Pending", condition("PodScheduled", "False", "Unschedulable"))}}}

	if err := Inject(context.Background(), c, s); err != nil {
		t.Fatal(err)
	}
	if strings.Join(c.applied, ",") != "ns:scn-app,scn-app:manifests.yaml" {
		t.Errorf("inject did: %v", c.applied)
	}
	if c.labels[ScenarioLabel] != "true" {
		t.Errorf("namespace labels = %v", c.labels)
	}

	results, err := VerifyFailure(context.Background(), c, s, time.Millisecond)
	if err != nil || len(results) != 1 || !results[0].OK {
		t.Fatalf("verify: err=%v results=%+v", err, results)
	}

	if err := Reset(context.Background(), c, s); err != nil {
		t.Fatal(err)
	}
	if len(c.deleted) != 1 || c.deleted[0] != "scn-app" {
		t.Errorf("reset deleted %v", c.deleted)
	}

	// Reset must never delete a namespace outside the scenario prefix.
	s.Namespace = "kube-system"
	if err := Reset(context.Background(), c, s); err == nil {
		t.Error("reset of a non-scenario namespace must be refused")
	}
}

func TestVerifyTimesOutWhenFailureAbsent(t *testing.T) {
	s, err := Load(writeScenario(t, "demo", strings.Replace(goodScenario, "timeoutSeconds: 10", "timeoutSeconds: 1", 1), goodExpected))
	if err != nil {
		t.Fatal(err)
	}
	c := &fakeCluster{pods: map[string][]kube.Pod{"app=demo": {pod("p", "Running", condition("PodScheduled", "True", ""))}}}
	results, err := VerifyFailure(context.Background(), c, s, 100*time.Millisecond)
	if !errors.Is(err, kube.ErrTimeout) {
		t.Fatalf("want timeout, got %v", err)
	}
	if len(results) != 1 || results[0].OK {
		t.Errorf("results should show the failing check: %+v", results)
	}
}
