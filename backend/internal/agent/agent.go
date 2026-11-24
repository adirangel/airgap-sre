// Package agent installs kagent into the lab and applies the SRE agent
// defined under agents/.
package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"sigs.k8s.io/yaml"

	"github.com/adirangel/airgap-sre/backend/internal/config"
	"github.com/adirangel/airgap-sre/backend/internal/lab"
)

// EnvAPIKey is the only way an API key enters the system: a session
// environment variable, turned into a Kubernetes Secret at install time.
const EnvAPIKey = "AIRGAP_SRE_MODEL_API_KEY"

// Files under agents/.
const (
	ModelConfigFile = "modelconfig.yaml"
	AgentFile       = "sre-agent.yaml"
	PromptFile      = "prompts/sre-agent.md"
)

// Model is the part of agents/modelconfig.yaml labctl needs.
type Model struct {
	Name            string
	Namespace       string
	Provider        string
	Model           string
	OllamaHost      string
	APIKeySecret    string
	APIKeySecretKey string
}

// Definition is the part of agents/sre-agent.yaml labctl needs.
type Definition struct {
	Name            string
	Namespace       string
	ModelConfig     string
	PromptConfigMap string
	PromptKey       string
	Tools           []string
}

type modelConfigYAML struct {
	Kind     string `json:"kind"`
	Metadata struct {
		Name      string `json:"name"`
		Namespace string `json:"namespace"`
	} `json:"metadata"`
	Spec struct {
		Provider        string `json:"provider"`
		Model           string `json:"model"`
		APIKeySecret    string `json:"apiKeySecret"`
		APIKeySecretKey string `json:"apiKeySecretKey"`
		Ollama          struct {
			Host string `json:"host"`
		} `json:"ollama"`
	} `json:"spec"`
}

type agentYAML struct {
	Kind     string `json:"kind"`
	Metadata struct {
		Name      string `json:"name"`
		Namespace string `json:"namespace"`
	} `json:"metadata"`
	Spec struct {
		Declarative struct {
			ModelConfig       string `json:"modelConfig"`
			SystemMessageFrom struct {
				Type string `json:"type"`
				Name string `json:"name"`
				Key  string `json:"key"`
			} `json:"systemMessageFrom"`
			Tools []struct {
				McpServer struct {
					ToolNames []string `json:"toolNames"`
				} `json:"mcpServer"`
			} `json:"tools"`
		} `json:"declarative"`
	} `json:"spec"`
}

// LoadModel reads agents/modelconfig.yaml.
func LoadModel(path string) (Model, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Model{}, err
	}
	var y modelConfigYAML
	if err := yaml.Unmarshal(data, &y); err != nil {
		return Model{}, fmt.Errorf("%s: %w", path, err)
	}
	m := Model{
		Name: y.Metadata.Name, Namespace: y.Metadata.Namespace,
		Provider: y.Spec.Provider, Model: y.Spec.Model, OllamaHost: y.Spec.Ollama.Host,
		APIKeySecret: y.Spec.APIKeySecret, APIKeySecretKey: y.Spec.APIKeySecretKey,
	}
	if y.Kind != "ModelConfig" || m.Name == "" || m.Namespace == "" || m.Provider == "" || m.Model == "" {
		return Model{}, fmt.Errorf("%s: need kind ModelConfig with metadata.name, metadata.namespace, spec.provider and spec.model", path)
	}
	if (m.APIKeySecret == "") != (m.APIKeySecretKey == "") {
		return Model{}, fmt.Errorf("%s: set both apiKeySecret and apiKeySecretKey, or neither", path)
	}
	return m, nil
}

// LoadDefinition reads agents/sre-agent.yaml.
func LoadDefinition(path string) (Definition, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Definition{}, err
	}
	var y agentYAML
	if err := yaml.Unmarshal(data, &y); err != nil {
		return Definition{}, fmt.Errorf("%s: %w", path, err)
	}
	d := Definition{
		Name: y.Metadata.Name, Namespace: y.Metadata.Namespace,
		ModelConfig:     y.Spec.Declarative.ModelConfig,
		PromptConfigMap: y.Spec.Declarative.SystemMessageFrom.Name,
		PromptKey:       y.Spec.Declarative.SystemMessageFrom.Key,
	}
	for _, t := range y.Spec.Declarative.Tools {
		d.Tools = append(d.Tools, t.McpServer.ToolNames...)
	}
	if y.Kind != "Agent" || d.Name == "" || d.Namespace == "" || d.ModelConfig == "" {
		return Definition{}, fmt.Errorf("%s: need kind Agent with metadata.name, metadata.namespace and spec.declarative.modelConfig", path)
	}
	if y.Spec.Declarative.SystemMessageFrom.Type != "ConfigMap" || d.PromptConfigMap == "" || d.PromptKey == "" {
		return Definition{}, fmt.Errorf("%s: spec.declarative.systemMessageFrom must reference a ConfigMap (labctl creates it from %s)", path, PromptFile)
	}
	return d, nil
}

// Manager installs and removes kagent and the agent.
type Manager struct {
	Lab    *lab.Lab
	Kagent config.Kagent
	Out    io.Writer
	Getenv func(string) string // defaults to os.Getenv
}

func (m *Manager) logf(format string, args ...any) { _, _ = fmt.Fprintf(m.Out, format, args...) }

func (m *Manager) getenv(k string) string {
	if m.Getenv != nil {
		return m.Getenv(k)
	}
	return os.Getenv(k)
}

// Load reads and cross-checks the files under agents/.
func (m *Manager) Load() (Model, Definition, string, error) {
	model, err := LoadModel(m.Lab.Paths.Agents(ModelConfigFile))
	if err != nil {
		return Model{}, Definition{}, "", err
	}
	def, err := LoadDefinition(m.Lab.Paths.Agents(AgentFile))
	if err != nil {
		return Model{}, Definition{}, "", err
	}
	prompt, err := os.ReadFile(m.Lab.Paths.Agents(filepath.FromSlash(PromptFile)))
	if err != nil {
		return Model{}, Definition{}, "", err
	}
	switch {
	case model.Namespace != m.Kagent.Namespace || def.Namespace != m.Kagent.Namespace:
		return Model{}, Definition{}, "", fmt.Errorf("agents/: ModelConfig and Agent must be in namespace %q (versions.yaml)", m.Kagent.Namespace)
	case def.ModelConfig != model.Name:
		return Model{}, Definition{}, "", fmt.Errorf("agents/: Agent uses modelConfig %q but %s defines %q", def.ModelConfig, ModelConfigFile, model.Name)
	}
	return model, def, string(prompt), nil
}

// MissingImages returns the kagent images that are not in the lab registry.
func (m *Manager) MissingImages(ctx context.Context) ([]string, error) {
	repos, err := m.Lab.Catalog(ctx)
	if err != nil {
		return nil, fmt.Errorf("lab registry is not reachable — run `labctl lab up`: %w", err)
	}
	have := map[string]bool{}
	for _, r := range repos {
		have[r] = true
	}
	var missing []string
	for _, img := range m.Lab.Images.Group("kagent") {
		registry, repository, _, err := lab.SplitRef(img.Ref)
		if err != nil {
			return nil, err
		}
		if !have[lab.Namespace(registry)+"/"+repository] {
			missing = append(missing, img.Ref)
		}
	}
	return missing, nil
}

// Install installs the kagent charts and applies the model, prompt and agent.
// It is idempotent: run it again after editing anything under agents/.
func (m *Manager) Install(ctx context.Context) error {
	model, def, prompt, err := m.Load()
	if err != nil {
		return err
	}
	missing, err := m.MissingImages(ctx)
	if err != nil {
		return err
	}
	if len(missing) > 0 {
		return fmt.Errorf("these images are not in the lab registry yet, so kagent could not start:\n  %s\nrun `labctl lab mirror --group kagent` first", strings.Join(missing, "\n  "))
	}
	if err := m.Lab.ConfigureNodes(ctx); err != nil {
		return err
	}

	k := m.Kagent
	tc := m.Lab.TC
	m.logf("installing kagent CRDs %s\n", k.Version)
	if _, err := tc.Helm(ctx, nil, "upgrade", "--install", "kagent-crds", k.CRDsChart,
		"--version", k.Version, "--namespace", k.Namespace, "--create-namespace",
		"--set", "kmcp.enabled=false", "--wait", "--timeout", "5m"); err != nil {
		return fmt.Errorf("helm install kagent-crds: %w", err)
	}
	m.logf("installing kagent %s (values: infrastructure/%s)\n", k.Version, k.Values)
	if _, err := tc.Helm(ctx, nil, "upgrade", "--install", "kagent", k.Chart,
		"--version", k.Version, "--namespace", k.Namespace,
		"--values", m.Lab.Paths.Infra(filepath.FromSlash(k.Values)),
		"--wait", "--timeout", "10m"); err != nil {
		return fmt.Errorf("helm install kagent: %w", err)
	}

	if err := m.applyAPIKey(ctx, model); err != nil {
		return err
	}
	cm := map[string]any{
		"apiVersion": "v1", "kind": "ConfigMap",
		"metadata": map[string]any{"name": def.PromptConfigMap, "namespace": def.Namespace},
		"data":     map[string]string{def.PromptKey: prompt},
	}
	if err := m.applyJSON(ctx, cm); err != nil {
		return fmt.Errorf("apply prompt ConfigMap: %w", err)
	}
	for _, f := range []string{ModelConfigFile, AgentFile} {
		if _, err := tc.Kubectl(ctx, "apply", "-f", m.Lab.Paths.Agents(f)); err != nil {
			return fmt.Errorf("apply agents/%s: %w", f, err)
		}
	}
	m.logf("waiting for agent %s/%s to become ready\n", def.Namespace, def.Name)
	if _, err := tc.Kubectl(ctx, "wait", "--for=condition=Ready", "agent/"+def.Name, "-n", def.Namespace, "--timeout=300s"); err != nil {
		return fmt.Errorf("agent did not become Ready: %w", err)
	}
	// A changed prompt lives in a ConfigMap the pod has already read.
	if _, err := tc.Kubectl(ctx, "rollout", "restart", "deployment/"+def.Name, "-n", def.Namespace); err != nil {
		return err
	}
	if _, err := tc.Kubectl(ctx, "rollout", "status", "deployment/"+def.Name, "-n", def.Namespace, "--timeout=300s"); err != nil {
		return fmt.Errorf("agent deployment did not roll out: %w", err)
	}
	m.logf("agent %s is ready (provider %s, model %s, %d tools)\n", def.Name, model.Provider, model.Model, len(def.Tools))
	if err := CheckModelEndpoint(ctx, model); err != nil {
		m.logf("WARNING: %v\n", err)
	}
	return nil
}

func (m *Manager) applyJSON(ctx context.Context, obj any) error {
	data, err := json.Marshal(obj)
	if err != nil {
		return err
	}
	_, err = m.Lab.TC.KubectlStdin(ctx, string(data), "apply", "-f", "-")
	return err
}

// applyAPIKey creates the API-key Secret from the session environment
// variable. The key travels over kubectl's stdin: never on a command line,
// never to disk.
func (m *Manager) applyAPIKey(ctx context.Context, model Model) error {
	if model.APIKeySecret == "" {
		return nil
	}
	key := m.getenv(EnvAPIKey)
	if key == "" {
		exists, err := m.Lab.Kube().Exists(ctx, "secret", model.APIKeySecret, "-n", model.Namespace)
		if err != nil {
			return err
		}
		if exists {
			m.logf("%s is not set; keeping the existing Secret %s\n", EnvAPIKey, model.APIKeySecret)
			return nil
		}
		return fmt.Errorf("%s uses an API key, but %s is not set in this session and the Secret %s does not exist.\nSet it for this session only (no setx):  $env:%s = '<key>'",
			ModelConfigFile, EnvAPIKey, model.APIKeySecret, EnvAPIKey)
	}
	secret := map[string]any{
		"apiVersion": "v1", "kind": "Secret", "type": "Opaque",
		"metadata":   map[string]any{"name": model.APIKeySecret, "namespace": model.Namespace},
		"stringData": map[string]string{model.APIKeySecretKey: key},
	}
	if err := m.applyJSON(ctx, secret); err != nil {
		return errors.New("could not create the API-key Secret (details withheld so the key is not printed)")
	}
	m.logf("API key from %s stored as Secret %s/%s\n", EnvAPIKey, model.Namespace, model.APIKeySecret)
	return nil
}

// Uninstall removes the agent and kagent from the cluster.
func (m *Manager) Uninstall(ctx context.Context) error {
	tc := m.Lab.TC
	k := m.Kagent
	for _, f := range []string{AgentFile, ModelConfigFile} {
		_, _ = tc.Kubectl(ctx, "delete", "-f", m.Lab.Paths.Agents(f), "--ignore-not-found", "--timeout=120s")
	}
	for _, release := range []string{"kagent", "kagent-crds"} {
		if _, err := tc.Helm(ctx, nil, "uninstall", release, "--namespace", k.Namespace, "--ignore-not-found", "--wait", "--timeout", "5m"); err != nil {
			return err
		}
	}
	if err := m.Lab.Kube().DeleteNamespace(ctx, k.Namespace, 5*time.Minute); err != nil {
		return err
	}
	m.logf("kagent removed (namespace %s deleted)\n", k.Namespace)
	return nil
}

// CheckModelEndpoint verifies, from the host, that a host-local Ollama is up
// and has the model. For other providers there is nothing to check locally.
func CheckModelEndpoint(ctx context.Context, model Model) error {
	if !strings.EqualFold(model.Provider, "Ollama") {
		return nil
	}
	u, err := url.Parse(model.OllamaHost)
	if err != nil || u.Host == "" {
		u, err = url.Parse("http://" + model.OllamaHost)
		if err != nil {
			return fmt.Errorf("cannot parse ollama host %q", model.OllamaHost)
		}
	}
	if u.Hostname() != "host.docker.internal" {
		return nil // not on this host; cannot check from here
	}
	port := u.Port()
	if port == "" {
		port = "11434"
	}
	endpoint := "http://" + net.JoinHostPort("127.0.0.1", port) + "/api/tags"
	cctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(cctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("ollama is not answering on %s — start Ollama on the host: %w", endpoint, err)
	}
	defer func() { _ = resp.Body.Close() }()
	var tags struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tags); err != nil {
		return fmt.Errorf("unexpected answer from %s: %w", endpoint, err)
	}
	var names []string
	for _, t := range tags.Models {
		if t.Name == model.Model || t.Name == model.Model+":latest" {
			return nil
		}
		names = append(names, t.Name)
	}
	return fmt.Errorf("ollama on the host does not have model %q (it has: %s). Pull it yourself with `ollama pull %s` — labctl never downloads models",
		model.Model, strings.Join(names, ", "), model.Model)
}
