# NookClaw

NookClaw is a local-first personal AI assistant written in Go.

It can run as:

- a direct CLI agent
- a gateway for chat and device integrations
- a web launcher and management console

NookClaw is designed to be usable on a single machine first, then extended only when you explicitly enable remote providers, channels, or web-facing features.

## Highlights

- local-first defaults with `~/.nookclaw` and `NOOKCLAW_*`
- default local model alias: `private-local -> ollama/qwen3.5:latest`
- direct agent chat from the terminal with tools, memory, and skills
- scheduled jobs with `cron`
- optional web launcher for configuration, chat, logs, models, tools, and skills
- optional channel gateways for Telegram, Discord, Slack, Matrix, LINE, OneBot, QQ, WeCom, DingTalk, WhatsApp, Pico, and MaixCam
- migration support for older `*claw` installs

## Project Status

NookClaw is usable today, but it is still early-stage software. Treat it like an admin tool: run it on machines you trust, review the config you generate, and enable network integrations deliberately.

## Requirements

- Go `1.25.7+` to build from source
- Ollama if you want the default local-first setup
- Node.js `20+` with `pnpm` only if you want to build the web frontend assets yourself

## Quick Start

Clone and build:

```bash
git clone https://github.com/samnoadd/NookClaw.git
cd NookClaw
make deps
make build
```

Initialize a fresh workspace:

```bash
./build/nookclaw onboard
./build/nookclaw status
```

If you want to use the default local model, make sure Ollama is running and the model exists:

```bash
ollama serve
ollama pull qwen3.5:latest
./build/nookclaw agent -m "hello"
```

If you already use a different Ollama model, edit `~/.nookclaw/config.json` after onboarding and point `private-local` to the model you already have.

For isolated testing without touching your normal home directory:

```bash
NOOKCLAW_HOME=/tmp/nookclaw-test ./build/nookclaw onboard
NOOKCLAW_HOME=/tmp/nookclaw-test ./build/nookclaw status
NOOKCLAW_HOME=/tmp/nookclaw-test ./build/nookclaw agent -m "hello"
```

## Main Commands

```bash
./build/nookclaw agent
./build/nookclaw status
./build/nookclaw model
./build/nookclaw skills
./build/nookclaw cron
./build/nookclaw gateway
./build/nookclaw migrate
./build/nookclaw --help
```

## Web Launcher

Build the launcher:

```bash
make build-launcher
```

Then start it:

```bash
./build/nookclaw-launcher
```

By default the launcher binds to `127.0.0.1:18800`, opens a browser, and manages the NookClaw gateway for you.

If you want to work on the web UI itself, see [web/README.md](web/README.md).

## Configuration And Data

Default locations:

- config: `~/.nookclaw/config.json`
- workspace: `~/.nookclaw/workspace`
- auth store: `~/.nookclaw/auth.json`

Primary environment variables:

- `NOOKCLAW_HOME`
- `NOOKCLAW_CONFIG`
- `NOOKCLAW_BUILTIN_SKILLS`
- `NOOKCLAW_GATEWAY_HOST`
- `NOOKCLAW_BINARY`

NookClaw still detects legacy `.picoclaw` paths and `PICOCLAW_*` variables for compatibility during migration.

## Privacy And Network Behavior

Fresh installs are intentionally conservative:

- no chat channels enabled by default
- no heartbeat loop enabled by default
- no remote skill registry enabled by default
- no web search enabled by default
- local Ollama is the default model path

If you later enable remote providers, chat platforms, browser tools, or registries, your prompts and metadata will follow those integrations.

## Build And Test

Common development targets:

```bash
make deps
make build
make test
make build-launcher
```

Install the CLI into `~/.local/bin`:

```bash
make install
```

## Migration

If you already have a legacy install, NookClaw can detect or import it.

Examples:

```bash
./build/nookclaw migrate
./build/nookclaw migrate --from openclaw
./build/nookclaw migrate --dry-run
```

Fresh installs use `~/.nookclaw`, but NookClaw can still read older `.picoclaw` layouts when needed.

## Origins And License

NookClaw is an MIT-licensed fork derived from PicoClaw. The current repo, packaging, branding, and local-first defaults are maintained here, while upstream attribution is preserved in [LICENSE](LICENSE).
