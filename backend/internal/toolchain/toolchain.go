// Package toolchain runs the repo-local kind, kubectl and helm binaries (and
// the docker CLI) so that nothing they write lands outside the repository:
// the kubeconfig, the kubectl discovery cache and every Helm home directory
// live under .lab/.
package toolchain

import (
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/adirangel/airgap-sre/backend/internal/config"
	"github.com/adirangel/airgap-sre/backend/internal/execx"
	"github.com/adirangel/airgap-sre/backend/internal/repo"
	"github.com/adirangel/airgap-sre/backend/internal/tools"
)

// Toolchain resolves and runs the pinned tools.
type Toolchain struct {
	Paths    repo.Paths
	Versions *config.Versions
	Runner   execx.Runner
	Log      io.Writer
	Platform tools.Platform
	Client   *http.Client
}

// New builds a Toolchain for the host platform.
func New(p repo.Paths, v *config.Versions, r execx.Runner, log io.Writer) *Toolchain {
	return &Toolchain{Paths: p, Versions: v, Runner: r, Log: log, Platform: tools.Host()}
}

// Env is added to every child process. It keeps kind, kubectl and helm away
// from ~/.kube and the user's Helm directories.
func (t *Toolchain) Env() []string {
	lab := t.Paths.LabDir()
	return []string{
		"KUBECONFIG=" + t.Paths.Kubeconfig(),
		"KUBECACHEDIR=" + filepath.Join(lab, "kube-cache"),
		"HELM_CACHE_HOME=" + filepath.Join(lab, "helm", "cache"),
		"HELM_CONFIG_HOME=" + filepath.Join(lab, "helm", "config"),
		"HELM_DATA_HOME=" + filepath.Join(lab, "helm", "data"),
	}
}

// Bin returns the path of a pinned tool, downloading it on first use.
func (t *Toolchain) Bin(ctx context.Context, name string) (string, error) {
	tool, err := t.Versions.Tool(name)
	if err != nil {
		return "", err
	}
	in := tools.Installer{ToolsDir: t.Paths.ToolsDir(), Platform: t.Platform, Client: t.Client, Log: t.Log}
	return in.Ensure(ctx, tool)
}

// InstallAll downloads every pinned tool and returns name -> path.
func (t *Toolchain) InstallAll(ctx context.Context) (map[string]string, error) {
	out := map[string]string{}
	for _, tool := range t.Versions.Tools {
		p, err := t.Bin(ctx, tool.Name)
		if err != nil {
			return nil, err
		}
		out[tool.Name] = p
	}
	return out, nil
}

func (t *Toolchain) run(ctx context.Context, tool string, stream io.Writer, stdin string, args []string) (string, error) {
	bin, err := t.Bin(ctx, tool)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(t.Paths.LabDir(), 0o755); err != nil {
		return "", err
	}
	return t.Runner.Run(ctx, execx.Cmd{Name: bin, Args: args, Env: t.Env(), Stdin: stdin, Stream: stream})
}

// Kind runs kind, streaming its progress output to stream (may be nil).
func (t *Toolchain) Kind(ctx context.Context, stream io.Writer, args ...string) (string, error) {
	return t.run(ctx, "kind", stream, "", args)
}

// Kubectl runs kubectl against the lab kubeconfig.
func (t *Toolchain) Kubectl(ctx context.Context, args ...string) (string, error) {
	return t.run(ctx, "kubectl", nil, "", append([]string{"--kubeconfig", t.Paths.Kubeconfig()}, args...))
}

// KubectlStdin runs kubectl with stdin content (for `apply -f -`).
func (t *Toolchain) KubectlStdin(ctx context.Context, stdin string, args ...string) (string, error) {
	return t.run(ctx, "kubectl", nil, stdin, append([]string{"--kubeconfig", t.Paths.Kubeconfig()}, args...))
}

// Helm runs helm against the lab kubeconfig.
func (t *Toolchain) Helm(ctx context.Context, stream io.Writer, args ...string) (string, error) {
	return t.run(ctx, "helm", stream, "", append([]string{"--kubeconfig", t.Paths.Kubeconfig()}, args...))
}

// Docker runs the docker CLI from PATH (Docker Desktop is a prerequisite).
func (t *Toolchain) Docker(ctx context.Context, args ...string) (string, error) {
	return t.Runner.Run(ctx, execx.Cmd{Name: "docker", Args: args})
}

// DockerStdin runs docker with stdin content.
func (t *Toolchain) DockerStdin(ctx context.Context, stdin string, args ...string) (string, error) {
	return t.Runner.Run(ctx, execx.Cmd{Name: "docker", Args: args, Stdin: stdin})
}

// DockerStream runs docker, streaming its output.
func (t *Toolchain) DockerStream(ctx context.Context, stream io.Writer, args ...string) (string, error) {
	return t.Runner.Run(ctx, execx.Cmd{Name: "docker", Args: args, Stream: stream})
}
