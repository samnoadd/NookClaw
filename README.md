# NookClaw

NookClaw is a local-first personal AI assistant written in Go.

It is a privacy-hardened fork of PicoClaw with a cleaner personal/self-hosted focus:

- `nookclaw` is the primary CLI
- fresh installs default to `~/.nookclaw`
- `NOOKCLAW_*` is the primary environment-variable namespace
- the default model alias is `private-local` over Ollama
- heartbeat, remote skill registry, web search, and remote exec targets are disabled by default
- existing `~/.picoclaw` data and `PICOCLAW_*` env vars are still detected for compatibility

## Status

NookClaw is suitable for local use, internal distribution, and publishing as a standalone fork.

Current public identity:

- module path: `github.com/samnoadd/NookClaw`
- CLI binary: `nookclaw`
- launcher binaries: `nookclaw-launcher`, `nookclaw-launcher-tui`
- web backend binary: `nookclaw-web`

This project is still early-stage software. Keep it on trusted machines and trusted networks.

## Build

```bash
git clone https://github.com/samnoadd/NookClaw.git
cd nookclaw
make deps
make build
make test
```

Main build outputs:

- `build/nookclaw`
- `build/nookclaw-launcher`

## Quick Start

```bash
./build/nookclaw onboard
./build/nookclaw status
./build/nookclaw agent -m "hello"
```

For an isolated test run:

```bash
NOOKCLAW_HOME=/tmp/nookclaw-test ./build/nookclaw onboard
NOOKCLAW_HOME=/tmp/nookclaw-test ./build/nookclaw status
```

## Paths And Env

Primary paths:

- config: `~/.nookclaw/config.json`
- workspace: `~/.nookclaw/workspace`
- auth store: `~/.nookclaw/auth.json`

Primary environment variables:

- `NOOKCLAW_HOME`
- `NOOKCLAW_CONFIG`
- `NOOKCLAW_BUILTIN_SKILLS`
- `NOOKCLAW_GATEWAY_HOST`
- `NOOKCLAW_BINARY`

Legacy PicoClaw paths and env vars are still accepted so existing installs can transition without breaking.

## Privacy Defaults

NookClaw ships with conservative defaults intended for local use:

- local Ollama model alias by default
- no enabled chat channels by default
- no heartbeat background loop by default
- no remote web tools enabled by default
- no remote skill registry enabled by default

If you enable remote providers, chat channels, or web tools, your data will follow those integrations.

## Migration

If you already have a PicoClaw install, NookClaw will detect legacy data automatically when the new `~/.nookclaw` home does not exist yet.

You can also migrate explicitly:

```bash
nookclaw migrate --from openclaw
```

## Publishing Notes

The repo is prepared for publishing as NookClaw, but you may still want to customize:

- GitHub repository owner/path if you publish somewhere other than `shayea/nookclaw`
- container image names
- localized READMEs and long-form docs
- logos, screenshots, and package metadata

## License

NookClaw is distributed under the MIT license. Upstream copyright and license notices from PicoClaw are preserved in [`LICENSE`](LICENSE).
