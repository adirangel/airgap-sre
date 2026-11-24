# agents/

The baseline SRE agent and its model.

| File | What it is |
| --- | --- |
| [`modelconfig.yaml`](modelconfig.yaml) | The **one** place where the model is configured (kagent `ModelConfig` named `sre-model`). Swap provider or model here |
| [`sre-agent.yaml`](sre-agent.yaml) | The kagent `Agent`: declarative, read-only, built-in Kubernetes tools only |
| [`prompts/sre-agent.md`](prompts/sre-agent.md) | The system prompt. `labctl agent install` loads it into the ConfigMap `sre-agent-prompt` |

Apply any change with:

```powershell
go run ./backend/cmd/labctl agent install
go run ./backend/cmd/labctl agent status
```

## Swapping the model

Edit `modelconfig.yaml` only.

- **Another local model**: change `spec.model`. Pull it yourself first
  (`ollama pull <model>`); `labctl` never downloads models. The model must
  support tool calling (`ollama show <model>` lists `tools`).
- **A cloud provider**: set `provider`, `model`, `apiKeySecret` and
  `apiKeySecretKey` as shown in the comment at the top of the file. Then, in
  the same PowerShell window (session only, never `setx`):

  ```powershell
  $env:AIRGAP_SRE_MODEL_API_KEY = '<key>'
  go run ./backend/cmd/labctl agent install
  ```

  `labctl` sends the key to the cluster as a Kubernetes Secret over stdin. It
  is never written to disk, never put on a command line and never committed.

## Rules

- The agent is read-only. `sre-agent.yaml` lists only read tools, and the
  tool server runs with read-only RBAC and no access to Secrets
  (`infrastructure/helm/kagent-values.yaml`). A unit test rejects write tools.
- The prompt lists exactly the root-cause categories the scorer knows
  (`backend/internal/scenario/scenario.go`) and must not mention anything
  specific to a scenario. Unit tests check both.
