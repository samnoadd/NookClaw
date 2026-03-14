# NookClaw Roadmap

This roadmap reflects the current direction of NookClaw as a local-first personal AI assistant.

## Priorities

### 1. Privacy And Safety

- keep local-first defaults strict
- improve secret handling and redaction
- strengthen tool sandboxing and network controls
- make remote features opt-in and explicit

### 2. Local-First Runtime

- better Ollama and local model workflows
- smoother onboarding for private local installs
- clearer migration from legacy `~/.picoclaw` setups to `~/.nookclaw`
- more reliable status, diagnostics, and upgrade paths

### 3. Packaging And Publishing

- stable release process for `nookclaw`
- clean Docker and launcher packaging
- consistent docs across CLI, web launcher, and releases
- repository metadata and screenshots aligned with the NookClaw brand

### 4. Web Console

- improve configuration UX
- reduce launcher confusion around paths and compatibility fallbacks
- harden OAuth and credential flows
- continue removing inherited upstream branding from the UI

### 5. Skills And Tooling

- clearer local skill management
- safer install and registry behavior
- better built-in skill discoverability
- stronger docs for custom skills and local automation

### 6. Documentation

- expand NookClaw-specific setup guides
- replace inherited upstream references with fork-specific guidance
- improve translated docs over time
- keep English docs as the source of truth until translations are refreshed

## Contribution Focus

The most useful contributions right now are:

- privacy and security hardening
- local-model quality of life improvements
- packaging and release polish
- documentation cleanup
- test reliability outside constrained environments
