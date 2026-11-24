// Package kube is a thin, typed layer over the repo-local kubectl binary.
// It only models the few fields labctl reads.
package kube

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// Kubectl is the part of toolchain.Toolchain this package needs.
type Kubectl interface {
	Kubectl(ctx context.Context, args ...string) (string, error)
	KubectlStdin(ctx context.Context, stdin string, args ...string) (string, error)
}

// Client talks to the lab cluster.
type Client struct {
	K Kubectl
}

// Meta is object metadata.
type Meta struct {
	Name      string            `json:"name"`
	Namespace string            `json:"namespace"`
	Labels    map[string]string `json:"labels"`
}

// Pod is a subset of v1.Pod.
type Pod struct {
	Metadata Meta      `json:"metadata"`
	Status   PodStatus `json:"status"`
}

// PodStatus is a subset of v1.PodStatus.
type PodStatus struct {
	Phase             string            `json:"phase"`
	Conditions        []Condition       `json:"conditions"`
	ContainerStatuses []ContainerStatus `json:"containerStatuses"`
}

// Condition is a pod or node condition.
type Condition struct {
	Type    string `json:"type"`
	Status  string `json:"status"`
	Reason  string `json:"reason"`
	Message string `json:"message"`
}

// ContainerStatus is a subset of v1.ContainerStatus.
type ContainerStatus struct {
	Name         string         `json:"name"`
	Ready        bool           `json:"ready"`
	RestartCount int            `json:"restartCount"`
	State        ContainerState `json:"state"`
	LastState    ContainerState `json:"lastState"`
}

// ContainerState is a subset of v1.ContainerState.
type ContainerState struct {
	Waiting    *StateDetail `json:"waiting"`
	Terminated *StateDetail `json:"terminated"`
	Running    *struct{}    `json:"running"`
}

// StateDetail carries the reason of a waiting or terminated container.
type StateDetail struct {
	Reason   string `json:"reason"`
	Message  string `json:"message"`
	ExitCode int    `json:"exitCode"`
}

// Ready reports whether the pod's Ready condition is True.
func (p Pod) Ready() bool {
	for _, c := range p.Status.Conditions {
		if c.Type == "Ready" {
			return c.Status == "True"
		}
	}
	return false
}

// Condition returns the condition of the given type, if present.
func (p Pod) Condition(typ string) (Condition, bool) {
	for _, c := range p.Status.Conditions {
		if c.Type == typ {
			return c, true
		}
	}
	return Condition{}, false
}

// EndpointSlice is a subset of discovery.k8s.io/v1 EndpointSlice.
type EndpointSlice struct {
	Metadata  Meta `json:"metadata"`
	Endpoints []struct {
		Addresses []string `json:"addresses"`
	} `json:"endpoints"`
}

// Node is a subset of v1.Node.
type Node struct {
	Metadata Meta `json:"metadata"`
	Status   struct {
		Conditions []Condition `json:"conditions"`
	} `json:"status"`
}

type list[T any] struct {
	Items []T `json:"items"`
}

func getList[T any](ctx context.Context, c Client, args ...string) ([]T, error) {
	out, err := c.K.Kubectl(ctx, append(args, "-o", "json")...)
	if err != nil {
		return nil, err
	}
	var l list[T]
	if err := json.Unmarshal([]byte(out), &l); err != nil {
		return nil, fmt.Errorf("parse kubectl output: %w", err)
	}
	return l.Items, nil
}

// Pods lists pods in a namespace, optionally filtered by a label selector.
func (c Client) Pods(ctx context.Context, namespace, selector string) ([]Pod, error) {
	args := []string{"get", "pods", "-n", namespace}
	if selector != "" {
		args = append(args, "-l", selector)
	}
	return getList[Pod](ctx, c, args...)
}

// Nodes lists cluster nodes.
func (c Client) Nodes(ctx context.Context) ([]Node, error) {
	return getList[Node](ctx, c, "get", "nodes")
}

// ServiceEndpointAddresses returns the ready-or-not endpoint addresses behind
// a Service, read from its EndpointSlices.
func (c Client) ServiceEndpointAddresses(ctx context.Context, namespace, service string) ([]string, error) {
	slices, err := getList[EndpointSlice](ctx, c, "get", "endpointslices", "-n", namespace, "-l", "kubernetes.io/service-name="+service)
	if err != nil {
		return nil, err
	}
	var addrs []string
	for _, s := range slices {
		for _, e := range s.Endpoints {
			addrs = append(addrs, e.Addresses...)
		}
	}
	return addrs, nil
}

// Exists reports whether `kubectl get <args>` finds the object.
func (c Client) Exists(ctx context.Context, args ...string) (bool, error) {
	_, err := c.K.Kubectl(ctx, append([]string{"get"}, args...)...)
	if err == nil {
		return true, nil
	}
	if strings.Contains(err.Error(), "NotFound") || strings.Contains(err.Error(), "not found") {
		return false, nil
	}
	return false, err
}

// Apply applies a manifest given as text.
func (c Client) Apply(ctx context.Context, manifest string) error {
	_, err := c.K.KubectlStdin(ctx, manifest, "apply", "-f", "-")
	return err
}

// ApplyPath applies a file or directory into a namespace.
func (c Client) ApplyPath(ctx context.Context, namespace, path string) error {
	_, err := c.K.Kubectl(ctx, "apply", "-n", namespace, "-f", path)
	return err
}

// EnsureNamespace creates the namespace (with labels) if it is missing.
func (c Client) EnsureNamespace(ctx context.Context, name string, labels map[string]string) error {
	var b strings.Builder
	fmt.Fprintf(&b, "apiVersion: v1\nkind: Namespace\nmetadata:\n  name: %s\n", name)
	if len(labels) > 0 {
		b.WriteString("  labels:\n")
		for k, v := range labels {
			fmt.Fprintf(&b, "    %s: %q\n", k, v)
		}
	}
	return c.Apply(ctx, b.String())
}

// DeleteNamespace deletes a namespace and waits until it is gone.
func (c Client) DeleteNamespace(ctx context.Context, name string, timeout time.Duration) error {
	_, err := c.K.Kubectl(ctx, "delete", "namespace", name, "--ignore-not-found", "--wait=true",
		fmt.Sprintf("--timeout=%ds", int(timeout.Seconds())))
	return err
}

// ErrTimeout is returned by Poll when the condition never became true.
var ErrTimeout = errors.New("timed out")

// Poll calls fn every interval until it reports done, returns an error, or
// the timeout passes. The last status text is included in the timeout error.
func Poll(ctx context.Context, timeout, interval time.Duration, fn func(context.Context) (done bool, status string, err error)) error {
	deadline := time.Now().Add(timeout)
	last := ""
	for {
		done, status, err := fn(ctx)
		if err != nil {
			return err
		}
		if done {
			return nil
		}
		if status != "" {
			last = status
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("%w after %s; last state: %s", ErrTimeout, timeout, last)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(interval):
		}
	}
}
