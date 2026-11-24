// Package lab owns the lab's lifecycle: a plain registry container, a kind
// cluster, and the containerd configuration that lets the cluster pull only
// from that registry.
package lab

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	v1 "github.com/google/go-containerregistry/pkg/v1"

	"github.com/adirangel/airgap-sre/backend/internal/config"
	"github.com/adirangel/airgap-sre/backend/internal/kube"
	"github.com/adirangel/airgap-sre/backend/internal/repo"
	"github.com/adirangel/airgap-sre/backend/internal/toolchain"
)

// OwnedLabel marks Docker objects created by labctl.
const OwnedLabel = "io.airgap-sre.owned=true"

// KindNetwork is the Docker network kind creates for its nodes.
const KindNetwork = "kind"

// HostImagesFile records every image labctl made the host Docker pull, so
// `labctl cleanup` can remove them even after a version bump.
const HostImagesFile = "host-images.txt"

// Lab bundles what the lab operations need.
type Lab struct {
	Paths  repo.Paths
	Cfg    config.Lab
	Images *config.Images
	TC     *toolchain.Toolchain
	Out    io.Writer
}

func (l *Lab) logf(format string, args ...any) { _, _ = fmt.Fprintf(l.Out, format, args...) }

// Kube returns a client for the lab cluster.
func (l *Lab) Kube() kube.Client { return kube.Client{K: l.TC} }

// CheckDocker fails with an actionable message when Docker is not reachable.
func (l *Lab) CheckDocker(ctx context.Context) error {
	out, err := l.TC.Docker(ctx, "version", "--format", "{{.Server.Version}} {{.Server.Os}}")
	if err != nil {
		return fmt.Errorf("docker is not reachable — start Docker Desktop (WSL2 backend) and retry: %w", err)
	}
	if !strings.Contains(out, "linux") {
		return fmt.Errorf("docker is running %q containers; switch Docker Desktop to Linux containers (WSL2 backend)", out)
	}
	return nil
}

// Up creates (or repairs) the registry and the cluster. It is idempotent.
func (l *Lab) Up(ctx context.Context) error {
	if err := l.CheckDocker(ctx); err != nil {
		return err
	}
	if _, err := l.TC.InstallAll(ctx); err != nil {
		return err
	}
	if err := l.ensureRegistry(ctx); err != nil {
		return err
	}
	if err := l.ensureCluster(ctx); err != nil {
		return err
	}
	if err := l.connectRegistry(ctx); err != nil {
		return err
	}
	if err := l.ConfigureNodes(ctx); err != nil {
		return err
	}
	l.logf("lab is up\n  cluster:    %s\n  kubeconfig: %s\n  registry:   %s (in-cluster: %s)\n",
		l.Cfg.ClusterName, l.Paths.Kubeconfig(), l.Cfg.Registry.HostAddr(), l.Cfg.Registry.ClusterEndpoint())
	return nil
}

func (l *Lab) containerRunning(ctx context.Context, name string) (exists, running bool) {
	out, err := l.TC.Docker(ctx, "container", "inspect", "-f", "{{.State.Running}}", name)
	if err != nil {
		return false, false
	}
	return true, strings.TrimSpace(out) == "true"
}

func (l *Lab) ensureRegistry(ctx context.Context) error {
	r := l.Cfg.Registry
	exists, running := l.containerRunning(ctx, r.ContainerName)
	switch {
	case exists && running:
		l.logf("registry %s is already running\n", r.ContainerName)
	case exists:
		l.logf("starting existing registry container %s\n", r.ContainerName)
		if _, err := l.TC.Docker(ctx, "start", r.ContainerName); err != nil {
			return err
		}
	default:
		l.logf("starting registry %s (%s) on %s\n", r.ContainerName, r.Image, r.HostAddr())
		if _, err := l.TC.Docker(ctx, "volume", "create", "--label", OwnedLabel, r.VolumeName); err != nil {
			return err
		}
		if err := l.recordHostImage(r.Image); err != nil {
			return err
		}
		_, err := l.TC.Docker(ctx, "run", "-d",
			"--restart", "unless-stopped",
			"--name", r.ContainerName,
			"--label", OwnedLabel,
			"-p", fmt.Sprintf("127.0.0.1:%d:5000", r.HostPort),
			"-v", r.VolumeName+":/var/lib/registry",
			r.Image)
		if err != nil {
			return fmt.Errorf("start registry: %w", err)
		}
	}
	return l.waitRegistry(ctx)
}

func (l *Lab) waitRegistry(ctx context.Context) error {
	url := "http://" + l.Cfg.Registry.HostAddr() + "/v2/"
	return kube.Poll(ctx, 60*time.Second, time.Second, func(ctx context.Context) (bool, string, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return false, "", err
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return false, "registry not answering at " + url, nil
		}
		_ = resp.Body.Close()
		return resp.StatusCode == http.StatusOK, "registry answered " + resp.Status, nil
	})
}

// ClusterExists reports whether the kind cluster is present.
func (l *Lab) ClusterExists(ctx context.Context) (bool, error) {
	out, err := l.TC.Kind(ctx, nil, "get", "clusters")
	if err != nil {
		return false, err
	}
	for _, line := range strings.Split(out, "\n") {
		if strings.TrimSpace(line) == l.Cfg.ClusterName {
			return true, nil
		}
	}
	return false, nil
}

func (l *Lab) ensureCluster(ctx context.Context) error {
	exists, err := l.ClusterExists(ctx)
	if err != nil {
		return err
	}
	if exists {
		l.logf("kind cluster %s already exists\n", l.Cfg.ClusterName)
	} else {
		l.logf("creating kind cluster %s (%s)\n", l.Cfg.ClusterName, l.Cfg.NodeImage)
		if err := l.recordHostImage(l.Cfg.NodeImage); err != nil {
			return err
		}
		_, err := l.TC.Kind(ctx, l.Out, "create", "cluster",
			"--name", l.Cfg.ClusterName,
			"--image", l.Cfg.NodeImage,
			"--config", l.Paths.Infra(filepath.FromSlash(l.Cfg.KindConfig)),
			"--kubeconfig", l.Paths.Kubeconfig(),
			"--wait", "180s")
		if err != nil {
			return fmt.Errorf("kind create cluster: %w", err)
		}
	}
	// Always refresh the repo-local kubeconfig (never ~/.kube/config).
	_, err = l.TC.Kind(ctx, nil, "export", "kubeconfig", "--name", l.Cfg.ClusterName, "--kubeconfig", l.Paths.Kubeconfig())
	return err
}

func (l *Lab) connectRegistry(ctx context.Context) error {
	out, err := l.TC.Docker(ctx, "container", "inspect", "-f", "{{json .NetworkSettings.Networks}}", l.Cfg.Registry.ContainerName)
	if err != nil {
		return err
	}
	var nets map[string]json.RawMessage
	if err := json.Unmarshal([]byte(out), &nets); err != nil {
		return fmt.Errorf("parse docker inspect output: %w", err)
	}
	if _, ok := nets[KindNetwork]; ok {
		return nil
	}
	_, err = l.TC.Docker(ctx, "network", "connect", KindNetwork, l.Cfg.Registry.ContainerName)
	return err
}

// Nodes returns the kind node container names.
func (l *Lab) Nodes(ctx context.Context) ([]string, error) {
	out, err := l.TC.Kind(ctx, nil, "get", "nodes", "--name", l.Cfg.ClusterName)
	if err != nil {
		return nil, err
	}
	var nodes []string
	for _, line := range strings.Split(out, "\n") {
		if n := strings.TrimSpace(line); n != "" {
			nodes = append(nodes, n)
		}
	}
	return nodes, nil
}

// ConfigureNodes writes the containerd hosts.toml files that implement the
// image-level air gap. containerd picks them up without a restart.
func (l *Lab) ConfigureNodes(ctx context.Context) error {
	refs := make([]string, 0, len(l.Images.Images))
	for _, i := range l.Images.Images {
		refs = append(refs, i.Ref)
	}
	registries, err := Registries(refs)
	if err != nil {
		return err
	}
	nodes, err := l.Nodes(ctx)
	if err != nil {
		return err
	}
	endpoint := l.Cfg.Registry.ClusterEndpoint()
	files := map[string]string{DefaultRegistryDir: DefaultHostsTOML(endpoint)}
	for _, r := range registries {
		files[r] = HostsTOML(endpoint, r)
	}
	for _, node := range nodes {
		for dir, content := range files {
			target := CertsDir + "/" + dir
			if _, err := l.TC.Docker(ctx, "exec", node, "mkdir", "-p", target); err != nil {
				return err
			}
			if _, err := l.TC.DockerStdin(ctx, content, "exec", "-i", node, "cp", "/dev/stdin", target+"/hosts.toml"); err != nil {
				return err
			}
		}
	}
	l.logf("containerd on %d node(s) now pulls only from %s (%d registries redirected + catch-all)\n", len(nodes), endpoint, len(registries))
	return nil
}

// Down deletes the cluster and the registry container. The registry volume is
// kept so mirrored images survive; `labctl cleanup` removes it.
func (l *Lab) Down(ctx context.Context) error {
	if err := l.CheckDocker(ctx); err != nil {
		return err
	}
	l.logf("deleting kind cluster %s\n", l.Cfg.ClusterName)
	if _, err := l.TC.Kind(ctx, l.Out, "delete", "cluster", "--name", l.Cfg.ClusterName, "--kubeconfig", l.Paths.Kubeconfig()); err != nil {
		return err
	}
	if exists, _ := l.containerRunning(ctx, l.Cfg.Registry.ContainerName); exists {
		l.logf("removing registry container %s (volume %s is kept)\n", l.Cfg.Registry.ContainerName, l.Cfg.Registry.VolumeName)
		if _, err := l.TC.Docker(ctx, "rm", "-f", l.Cfg.Registry.ContainerName); err != nil {
			return err
		}
	}
	return nil
}

// Status is a snapshot of the lab.
type Status struct {
	RegistryExists  bool
	RegistryRunning bool
	Repositories    []string
	ClusterExists   bool
	Nodes           []NodeStatus
}

// NodeStatus is one cluster node.
type NodeStatus struct {
	Name  string
	Ready bool
}

// Status inspects the registry and the cluster.
func (l *Lab) Status(ctx context.Context) (Status, error) {
	var st Status
	if err := l.CheckDocker(ctx); err != nil {
		return st, err
	}
	st.RegistryExists, st.RegistryRunning = l.containerRunning(ctx, l.Cfg.Registry.ContainerName)
	if st.RegistryRunning {
		repos, err := l.Catalog(ctx)
		if err == nil {
			st.Repositories = repos
		}
	}
	exists, err := l.ClusterExists(ctx)
	if err != nil {
		return st, err
	}
	st.ClusterExists = exists
	if exists {
		nodes, err := l.Kube().Nodes(ctx)
		if err != nil {
			return st, err
		}
		for _, n := range nodes {
			ready := false
			for _, c := range n.Status.Conditions {
				if c.Type == "Ready" && c.Status == "True" {
					ready = true
				}
			}
			st.Nodes = append(st.Nodes, NodeStatus{Name: n.Metadata.Name, Ready: ready})
		}
	}
	return st, nil
}

// Catalog lists the repositories in the lab registry.
func (l *Lab) Catalog(ctx context.Context) ([]string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://"+l.Cfg.Registry.HostAddr()+"/v2/_catalog?n=1000", nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("registry catalog: HTTP %s", resp.Status)
	}
	var body struct {
		Repositories []string `json:"repositories"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, err
	}
	return body.Repositories, nil
}

// MirrorGroup copies one group of images.yaml into the lab registry and makes
// sure the nodes redirect every registry those images come from.
func (l *Lab) MirrorGroup(ctx context.Context, group, arch string) error {
	images := l.Images.Group(group)
	if len(images) == 0 {
		return fmt.Errorf("no images in group %q", group)
	}
	if _, running := l.containerRunning(ctx, l.Cfg.Registry.ContainerName); !running {
		return errors.New("the lab registry is not running — run `labctl lab up` first")
	}
	if arch == "" {
		arch = runtime.GOARCH
	}
	results, err := Mirror(ctx, l.Out, l.Cfg.Registry.HostAddr(), images, v1.Platform{OS: "linux", Architecture: arch})
	if err != nil {
		return err
	}
	l.logf("mirrored %d image(s)\n", len(results))
	if exists, err := l.ClusterExists(ctx); err == nil && exists {
		return l.ConfigureNodes(ctx)
	}
	return nil
}

func (l *Lab) recordHostImage(ref string) error {
	if err := os.MkdirAll(l.Paths.LabDir(), 0o755); err != nil {
		return err
	}
	path := filepath.Join(l.Paths.LabDir(), HostImagesFile)
	existing, _ := os.ReadFile(path)
	for _, line := range strings.Split(string(existing), "\n") {
		if strings.TrimSpace(line) == ref {
			return nil
		}
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintln(f, ref); err != nil {
		_ = f.Close()
		return err
	}
	return f.Close()
}
