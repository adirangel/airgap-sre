package agent

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/adirangel/airgap-sre/backend/internal/config"
	"github.com/adirangel/airgap-sre/backend/internal/lab"
	"github.com/adirangel/airgap-sre/backend/internal/repo"
	"github.com/adirangel/airgap-sre/backend/internal/scenario"
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

// The files under agents/ must load and agree with each other and with the
// pinned kagent settings.
func TestRealAgentFiles(t *testing.T) {
	p := repoPaths(t)
	versions, err := config.LoadVersions(p.Infra("versions.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	m := &Manager{Lab: &lab.Lab{Paths: p}, Kagent: versions.Kagent}
	model, def, prompt, err := m.Load()
	if err != nil {
		t.Fatal(err)
	}
	if model.Provider == "" || model.Model == "" {
		t.Errorf("model = %+v", model)
	}
	if len(def.Tools) == 0 {
		t.Error("the agent has no tools")
	}
	// The baseline agent is read-only: no tool that changes the cluster.
	for _, tool := range def.Tools {
		for _, verb := range []string{"apply", "create", "delete", "patch", "scale", "rollout", "execute", "label", "annotate", "generate"} {
			if strings.Contains(tool, verb) {
				t.Errorf("tool %s is not read-only", tool)
			}
		}
	}
	// The prompt must offer exactly the scorer's closed category list.
	for _, c := range scenario.Categories {
		if !strings.Contains(prompt, "`"+c+"`") {
			t.Errorf("system prompt does not list category %q", c)
		}
	}
	if !strings.Contains(prompt, `"root_cause_category"`) {
		t.Error("system prompt does not describe the JSON diagnosis block")
	}
}

// The prompt must not contain anything specific to a scenario.
func TestPromptDoesNotLeakScenarios(t *testing.T) {
	p := repoPaths(t)
	data, err := os.ReadFile(p.Agents(filepath.FromSlash(PromptFile)))
	if err != nil {
		t.Fatal(err)
	}
	prompt := strings.ToLower(string(data))
	all, err := scenario.LoadAll(p.Infra("scenarios"))
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range all {
		banned := []string{s.Namespace, s.ID}
		for _, r := range s.Expected.MustName {
			banned = append(banned, r.Name)
		}
		for _, b := range banned {
			if strings.Contains(prompt, strings.ToLower(b)) {
				t.Errorf("system prompt mentions %q from scenario %s", b, s.ID)
			}
		}
	}
}

// No API key may ever be committed under agents/.
func TestNoSecretsInAgentFiles(t *testing.T) {
	p := repoPaths(t)
	model, err := LoadModel(p.Agents(ModelConfigFile))
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(p.Agents(ModelConfigFile))
	if err != nil {
		t.Fatal(err)
	}
	for _, marker := range []string{"sk-", "AIza", "apiKey:"} {
		if strings.Contains(string(data), marker) {
			t.Errorf("%s contains %q — keys must come from %s only", ModelConfigFile, marker, EnvAPIKey)
		}
	}
	if (model.APIKeySecret == "") != (model.APIKeySecretKey == "") {
		t.Error("apiKeySecret and apiKeySecretKey must be set together")
	}
}

func write(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "f.yaml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoadValidation(t *testing.T) {
	if _, err := LoadModel(write(t, "kind: ModelConfig\nmetadata: {name: m, namespace: kagent}\nspec: {provider: Ollama}\n")); err == nil {
		t.Error("ModelConfig without a model must be rejected")
	}
	if _, err := LoadModel(write(t, "kind: ModelConfig\nmetadata: {name: m, namespace: kagent}\nspec: {provider: Anthropic, model: x, apiKeySecret: s}\n")); err == nil {
		t.Error("apiKeySecret without apiKeySecretKey must be rejected")
	}
	if _, err := LoadDefinition(write(t, "kind: Agent\nmetadata: {name: a, namespace: kagent}\nspec: {declarative: {modelConfig: m, systemMessage: hi}}\n")); err == nil {
		t.Error("an Agent with an inline prompt must be rejected (the prompt lives in a file)")
	}
}

func TestCheckModelEndpoint(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"models":[{"name":"present:1b"},{"name":"tagless:latest"}]}`))
	}))
	defer srv.Close()
	port := srv.URL[strings.LastIndex(srv.URL, ":")+1:]
	host := "http://host.docker.internal:" + port
	ctx := context.Background()

	if err := CheckModelEndpoint(ctx, Model{Provider: "Ollama", Model: "present:1b", OllamaHost: host}); err != nil {
		t.Errorf("present model: %v", err)
	}
	if err := CheckModelEndpoint(ctx, Model{Provider: "Ollama", Model: "tagless", OllamaHost: "host.docker.internal:" + port}); err != nil {
		t.Errorf("implicit :latest and scheme-less host: %v", err)
	}
	err := CheckModelEndpoint(ctx, Model{Provider: "Ollama", Model: "absent:7b", OllamaHost: host})
	if err == nil || !strings.Contains(err.Error(), "ollama pull absent:7b") {
		t.Errorf("missing model should explain how to pull it, got %v", err)
	}
	// Nothing to check for cloud providers or a remote Ollama.
	if err := CheckModelEndpoint(ctx, Model{Provider: "Anthropic", Model: "x"}); err != nil {
		t.Errorf("cloud provider: %v", err)
	}
	if err := CheckModelEndpoint(ctx, Model{Provider: "Ollama", Model: "x", OllamaHost: "http://ollama.example.com:11434"}); err != nil {
		t.Errorf("remote ollama: %v", err)
	}
}
