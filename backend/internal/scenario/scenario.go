// Package scenario loads the failure scenarios under
// infrastructure/scenarios/ and runs inject / verify / reset against the lab.
package scenario

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"sigs.k8s.io/yaml"

	"github.com/adirangel/airgap-sre/backend/internal/kube"
)

// Root-cause categories. The set is closed: expected.yaml must use one of
// these, and the agent is asked to pick exactly one. It includes categories no
// scenario uses, so the answer cannot be guessed by elimination.
var Categories = []string{
	"bad_config",
	"oom_killed",
	"image_pull_failure",
	"unschedulable",
	"service_selector_mismatch",
	"probe_failure",
	"missing_dependency",
	"rbac_denied",
	"network_policy",
	"storage",
	"node_failure",
	"quota_exceeded",
	"unknown",
}

// IsCategory reports whether c is a known root-cause category.
func IsCategory(c string) bool {
	for _, k := range Categories {
		if k == c {
			return true
		}
	}
	return false
}

// Check types understood by verify.
const (
	CheckContainerWaiting    = "containerWaiting"
	CheckContainerTerminated = "containerTerminated"
	CheckPodUnschedulable    = "podUnschedulable"
	CheckPodsReady           = "podsReady"
	CheckServiceNoEndpoints  = "serviceNoEndpoints"
)

// NamespacePrefix is required for every scenario namespace, so reset can
// never delete a namespace that is not a scenario's.
const NamespacePrefix = "scn-"

// ScenarioLabel marks scenario namespaces. Its value is always "true": the
// agent can read namespace labels, so the scenario id must not appear there.
const ScenarioLabel = "airgap-sre.io/scenario"

// Scenario is one reproducible failure.
type Scenario struct {
	ID               string   `json:"id"`
	Order            int      `json:"order"`
	Title            string   `json:"title"`
	Description      string   `json:"description"`
	Namespace        string   `json:"namespace"`
	Symptom          string   `json:"symptom"`
	UnmirroredImages []string `json:"unmirroredImages,omitempty"`
	Verify           Verify   `json:"verify"`

	// Filled by Load, not read from scenario.yaml.
	Dir      string   `json:"-"`
	Expected Expected `json:"-"`
}

// Verify describes how to confirm the failure really happened.
type Verify struct {
	TimeoutSeconds int     `json:"timeoutSeconds"`
	Checks         []Check `json:"checks"`
}

// Check is one observable condition. All checks must hold at the same time.
type Check struct {
	Type     string   `json:"type"`
	Selector string   `json:"selector,omitempty"`
	Reasons  []string `json:"reasons,omitempty"`
	Service  string   `json:"service,omitempty"`
}

// Expected is expected.yaml: what a correct diagnosis must contain.
type Expected struct {
	RootCauseCategory string     `json:"rootCauseCategory"`
	MustName          []Resource `json:"mustName"`
}

// Resource is something the diagnosis has to name.
type Resource struct {
	Kind string `json:"kind"`
	Name string `json:"name"`
}

// ManifestsPath is the file applied by inject.
func (s Scenario) ManifestsPath() string { return filepath.Join(s.Dir, "manifests.yaml") }

var idRe = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// Load reads one scenario directory and validates it.
func Load(dir string) (Scenario, error) {
	var s Scenario
	if err := readStrict(filepath.Join(dir, "scenario.yaml"), &s); err != nil {
		return s, err
	}
	if err := readStrict(filepath.Join(dir, "expected.yaml"), &s.Expected); err != nil {
		return s, err
	}
	s.Dir = dir
	s.Symptom = strings.Join(strings.Fields(s.Symptom), " ")
	s.Description = strings.Join(strings.Fields(s.Description), " ")
	if err := s.validate(); err != nil {
		return s, fmt.Errorf("scenario %s: %w", filepath.Base(dir), err)
	}
	return s, nil
}

func (s Scenario) validate() error {
	switch {
	case !idRe.MatchString(s.ID):
		return fmt.Errorf("id %q must be lowercase words joined by dashes", s.ID)
	case s.ID != filepath.Base(s.Dir):
		return fmt.Errorf("id %q must equal the directory name", s.ID)
	case !strings.HasPrefix(s.Namespace, NamespacePrefix) || len(s.Namespace) == len(NamespacePrefix):
		return fmt.Errorf("namespace must start with %q, got %q", NamespacePrefix, s.Namespace)
	case leaks(s.Namespace, s.ID) || leaks(s.Namespace, s.Expected.RootCauseCategory):
		return fmt.Errorf("namespace %q gives the answer away: it must not contain words from the id or the category", s.Namespace)
	case s.Title == "" || s.Description == "" || s.Symptom == "":
		return fmt.Errorf("title, description and symptom are required")
	case s.Verify.TimeoutSeconds <= 0:
		return fmt.Errorf("verify.timeoutSeconds must be positive")
	case len(s.Verify.Checks) == 0:
		return fmt.Errorf("verify needs at least one check")
	case !IsCategory(s.Expected.RootCauseCategory):
		return fmt.Errorf("expected.yaml: unknown rootCauseCategory %q", s.Expected.RootCauseCategory)
	case s.Expected.RootCauseCategory == "unknown":
		return fmt.Errorf("expected.yaml: rootCauseCategory must not be \"unknown\"")
	case len(s.Expected.MustName) == 0:
		return fmt.Errorf("expected.yaml: mustName needs at least one resource")
	}
	for _, r := range s.Expected.MustName {
		if r.Kind == "" || r.Name == "" {
			return fmt.Errorf("expected.yaml: every mustName entry needs kind and name")
		}
	}
	for i, c := range s.Verify.Checks {
		if err := c.validate(); err != nil {
			return fmt.Errorf("verify.checks[%d]: %w", i, err)
		}
	}
	if _, err := os.Stat(s.ManifestsPath()); err != nil {
		return fmt.Errorf("manifests.yaml: %w", err)
	}
	return nil
}

func (c Check) validate() error {
	switch c.Type {
	case CheckContainerWaiting, CheckContainerTerminated:
		if c.Selector == "" || len(c.Reasons) == 0 {
			return fmt.Errorf("%s needs selector and reasons", c.Type)
		}
	case CheckPodUnschedulable, CheckPodsReady:
		if c.Selector == "" {
			return fmt.Errorf("%s needs selector", c.Type)
		}
	case CheckServiceNoEndpoints:
		if c.Service == "" {
			return fmt.Errorf("%s needs service", c.Type)
		}
	default:
		return fmt.Errorf("unknown check type %q", c.Type)
	}
	return nil
}

// LoadAll loads every scenario under dir, sorted by order then id.
func LoadAll(dir string) ([]Scenario, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var out []Scenario
	seenNS := map[string]string{}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		s, err := Load(filepath.Join(dir, e.Name()))
		if err != nil {
			return nil, err
		}
		if other, dup := seenNS[s.Namespace]; dup {
			return nil, fmt.Errorf("scenarios %s and %s share namespace %s", other, s.ID, s.Namespace)
		}
		seenNS[s.Namespace] = s.ID
		out = append(out, s)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Order != out[j].Order {
			return out[i].Order < out[j].Order
		}
		return out[i].ID < out[j].ID
	})
	return out, nil
}

// Find returns the scenario with the given id.
func Find(all []Scenario, id string) (Scenario, error) {
	for _, s := range all {
		if s.ID == id {
			return s, nil
		}
	}
	ids := make([]string, 0, len(all))
	for _, s := range all {
		ids = append(ids, s.ID)
	}
	return Scenario{}, fmt.Errorf("unknown scenario %q (known: %s)", id, strings.Join(ids, ", "))
}

var imageLineRe = regexp.MustCompile(`(?m)^\s*(?:-\s+)?image:\s*["']?([^\s"'#]+)`)

// ManifestImages returns every container image referenced in manifests.yaml.
func (s Scenario) ManifestImages() ([]string, error) {
	data, err := os.ReadFile(s.ManifestsPath())
	if err != nil {
		return nil, err
	}
	var images []string
	for _, m := range imageLineRe.FindAllStringSubmatch(string(data), -1) {
		images = append(images, m[1])
	}
	return images, nil
}

// Cluster is what inject / verify / reset need from the lab cluster.
type Cluster interface {
	Pods(ctx context.Context, namespace, selector string) ([]kube.Pod, error)
	ServiceEndpointAddresses(ctx context.Context, namespace, service string) ([]string, error)
	EnsureNamespace(ctx context.Context, name string, labels map[string]string) error
	ApplyPath(ctx context.Context, namespace, path string) error
	DeleteNamespace(ctx context.Context, name string, timeout time.Duration) error
}

// Inject creates the scenario namespace and applies its manifests.
func Inject(ctx context.Context, c Cluster, s Scenario) error {
	if err := c.EnsureNamespace(ctx, s.Namespace, map[string]string{ScenarioLabel: "true"}); err != nil {
		return err
	}
	return c.ApplyPath(ctx, s.Namespace, s.ManifestsPath())
}

// Reset deletes the scenario namespace and waits until it is gone.
func Reset(ctx context.Context, c Cluster, s Scenario) error {
	if !strings.HasPrefix(s.Namespace, NamespacePrefix) {
		return fmt.Errorf("refusing to delete namespace %q: not a scenario namespace", s.Namespace)
	}
	return c.DeleteNamespace(ctx, s.Namespace, 3*time.Minute)
}

// CheckResult is the outcome of one check at one point in time.
type CheckResult struct {
	Check  Check
	OK     bool
	Detail string
}

// Evaluate runs every check once.
func Evaluate(ctx context.Context, c Cluster, s Scenario) ([]CheckResult, error) {
	results := make([]CheckResult, 0, len(s.Verify.Checks))
	for _, chk := range s.Verify.Checks {
		ok, detail, err := evalCheck(ctx, c, s.Namespace, chk)
		if err != nil {
			return nil, err
		}
		results = append(results, CheckResult{Check: chk, OK: ok, Detail: detail})
	}
	return results, nil
}

// VerifyFailure polls until every check holds at the same time, which proves
// the scenario's failure is really present in the cluster.
func VerifyFailure(ctx context.Context, c Cluster, s Scenario, interval time.Duration) ([]CheckResult, error) {
	var last []CheckResult
	err := kube.Poll(ctx, time.Duration(s.Verify.TimeoutSeconds)*time.Second, interval, func(ctx context.Context) (bool, string, error) {
		results, err := Evaluate(ctx, c, s)
		if err != nil {
			return false, "", err
		}
		last = results
		all := true
		var pending []string
		for _, r := range results {
			if !r.OK {
				all = false
				pending = append(pending, r.Check.Type+": "+r.Detail)
			}
		}
		return all, strings.Join(pending, "; "), nil
	})
	return last, err
}

func evalCheck(ctx context.Context, c Cluster, ns string, chk Check) (bool, string, error) {
	switch chk.Type {
	case CheckServiceNoEndpoints:
		addrs, err := c.ServiceEndpointAddresses(ctx, ns, chk.Service)
		if err != nil {
			return false, "", err
		}
		if len(addrs) == 0 {
			return true, fmt.Sprintf("service %s has no endpoints", chk.Service), nil
		}
		return false, fmt.Sprintf("service %s has %d endpoint(s)", chk.Service, len(addrs)), nil
	}

	pods, err := c.Pods(ctx, ns, chk.Selector)
	if err != nil {
		return false, "", err
	}
	if len(pods) == 0 {
		return false, fmt.Sprintf("no pods match %s yet", chk.Selector), nil
	}
	switch chk.Type {
	case CheckContainerWaiting:
		for _, p := range pods {
			for _, cs := range p.Status.ContainerStatuses {
				if w := cs.State.Waiting; w != nil && contains(chk.Reasons, w.Reason) {
					return true, fmt.Sprintf("pod %s container %s is waiting: %s", p.Metadata.Name, cs.Name, w.Reason), nil
				}
			}
		}
		return false, fmt.Sprintf("no container of %s is waiting with %v yet (%s)", chk.Selector, chk.Reasons, summarize(pods)), nil
	case CheckContainerTerminated:
		for _, p := range pods {
			for _, cs := range p.Status.ContainerStatuses {
				for _, t := range []*kube.StateDetail{cs.State.Terminated, cs.LastState.Terminated} {
					if t != nil && contains(chk.Reasons, t.Reason) {
						return true, fmt.Sprintf("pod %s container %s was terminated: %s (exit code %d)", p.Metadata.Name, cs.Name, t.Reason, t.ExitCode), nil
					}
				}
			}
		}
		return false, fmt.Sprintf("no container of %s was terminated with %v yet (%s)", chk.Selector, chk.Reasons, summarize(pods)), nil
	case CheckPodUnschedulable:
		for _, p := range pods {
			if cond, ok := p.Condition("PodScheduled"); ok && cond.Status == "False" && cond.Reason == "Unschedulable" {
				return true, fmt.Sprintf("pod %s is unschedulable: %s", p.Metadata.Name, cond.Message), nil
			}
		}
		return false, fmt.Sprintf("no pod of %s is unschedulable yet (%s)", chk.Selector, summarize(pods)), nil
	case CheckPodsReady:
		for _, p := range pods {
			if !p.Ready() {
				return false, fmt.Sprintf("pod %s is not Ready yet (%s)", p.Metadata.Name, summarize(pods)), nil
			}
		}
		return true, fmt.Sprintf("%d pod(s) of %s are Ready", len(pods), chk.Selector), nil
	}
	return false, "", fmt.Errorf("unknown check type %q", chk.Type)
}

func summarize(pods []kube.Pod) string {
	parts := make([]string, 0, len(pods))
	for _, p := range pods {
		parts = append(parts, p.Metadata.Name+"="+p.Status.Phase)
	}
	return strings.Join(parts, ", ")
}

// leaks reports whether name contains any word (4+ letters) of hint, where
// hint is a dash- or underscore-separated id or category.
func leaks(name, hint string) bool {
	name = strings.ToLower(name)
	for _, w := range strings.FieldsFunc(strings.ToLower(hint), func(r rune) bool { return r == '-' || r == '_' }) {
		if len(w) >= 4 && strings.Contains(name, w) {
			return true
		}
	}
	return false
}

func contains(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}

func readStrict(path string, into any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if err := yaml.UnmarshalStrict(data, into); err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	return nil
}
