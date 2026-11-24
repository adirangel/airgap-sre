# agents/

kagent `Agent` and `ModelConfig` manifests and the system prompts they use.
Populated in milestone M3.

- The model is configured in exactly one swappable `ModelConfig` file.
- API keys are never stored here; they come from environment variables and
  are turned into a Kubernetes Secret at install time by `labctl`.
