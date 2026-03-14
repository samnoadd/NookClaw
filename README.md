# NookClaw

NookClaw is a personal AI assistant written in Go.

It can run as:

- a direct CLI agent
- a gateway for chat and device integrations
- a web launcher and management console

## Highlights

- terminal-first agent workflow
- pluggable model providers and model aliases
- tools, memory, and installable skills
- scheduled jobs with `cron`
- optional web UI for chat, configuration, logs, models, tools, and skills
- optional channel integrations for Telegram, Discord, Slack, Matrix, LINE, OneBot, QQ, WeCom, DingTalk, WhatsApp, Pico, and MaixCam

## Project Status

NookClaw is usable today, but it is still early-stage software. Review the generated configuration, choose your model/provider setup intentionally, and enable external integrations deliberately.

## Requirements

- Go `1.25.7+` to build from source
- at least one model backend you want to use
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

Then open your config, choose the model/provider you want to use, and run the agent:

```bash
./build/nookclaw agent -m "hello"
```

For isolated testing:

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

Start it:

```bash
./build/nookclaw-launcher
```

By default the launcher listens on `127.0.0.1:18800`, opens a browser, and manages the NookClaw gateway for you.

If you want to work on the web UI itself, see [web/README.md](web/README.md).

## Configuration

Default files:

- config: `~/.nookclaw/config.json`
- workspace: `~/.nookclaw/workspace`
- auth store: `~/.nookclaw/auth.json`

Main environment variables:

- `NOOKCLAW_HOME`
- `NOOKCLAW_CONFIG`
- `NOOKCLAW_BUILTIN_SKILLS`
- `NOOKCLAW_GATEWAY_HOST`
- `NOOKCLAW_BINARY`

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

NookClaw includes migration commands for importing configuration and workspace data from earlier claw-based installs.

Examples:

```bash
./build/nookclaw migrate
./build/nookclaw migrate --from openclaw
./build/nookclaw migrate --dry-run
```

## License

NookClaw is distributed under the MIT license. Upstream attribution is preserved in [LICENSE](LICENSE).
