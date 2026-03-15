# NookClaw Profile

NookClaw defines a local-first personal profile for this fork.

## What Changed

- Default model is now `private-local`, mapped to `ollama/qwen3.5:latest`.
- Heartbeat is disabled by default.
- Web search and web fetch are disabled by default.
- Remote skill registry lookup and install are disabled by default.
- Remote exec targets are disabled by default.
- Workspace bootstrap files are created with private permissions.
- Session logs and heartbeat logs are written with private permissions.
- Embedded workspace identity and behavior text now describe a private local-first assistant.

## Files To Adjust For Your Machine

- `pkg/config/defaults.go`
  Change the `private-local` model alias if your Ollama model name differs.
- `workspace/AGENTS.md`
  Tune behavior rules for your preferred interaction style.
- `workspace/IDENTITY.md`
  Rename the assistant and describe your own fork.

## Build

For general users, install from public releases:

```bash
curl -fsSL https://raw.githubusercontent.com/samnoadd/NookClaw/main/install.sh | bash
```

This path only works when the repository and release assets are publicly reachable,
and after a GitHub release has been published.
For a private fork or unreleased branch, install Go and build from source instead:

```bash
make deps
make install
```

NookClaw now uses the `nookclaw` command name.
Fresh installs default to `~/.nookclaw`, and the primary environment-variable namespace is `NOOKCLAW_*`.
For compatibility, NookClaw still detects legacy `~/.picoclaw` data and `PICOCLAW_*` environment variables.

## Install Locally

After installation, run your fork directly:

```bash
nookclaw onboard
nookclaw agent -m "Hello"
```

If you are developing from the repo without installing, use `./build/nookclaw`.
