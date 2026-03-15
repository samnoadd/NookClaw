package onboard

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/samnoadd/NookClaw/cmd/nookclaw/internal"
	"github.com/samnoadd/NookClaw/pkg/config"
	"github.com/samnoadd/NookClaw/web/backend/launcherconfig"
)

func onboard(opts onboardOptions) error {
	if err := validateOnboardOptions(opts); err != nil {
		return err
	}

	configPath := internal.GetDefaultConfigPath()
	interactive := !opts.NonInteractive && isInteractiveTerminal()
	updated, proceed := handleExistingSetup(configPath, interactive, opts.Force, os.Stdin, os.Stdout)
	if !proceed {
		return nil
	}

	cfg := config.DefaultConfig()
	state, err := newSetupWizard(os.Stdin, os.Stdout, interactive, opts).run(cfg)
	if err != nil {
		if err == errOnboardingAborted {
			return nil
		}
		return err
	}
	if err := config.SaveConfig(configPath, cfg); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}

	workspace := cfg.WorkspacePath()
	createWorkspaceTemplates(workspace)
	launcherPath := launcherconfig.PathForAppConfig(configPath)
	if err := launcherconfig.Save(launcherPath, state.LauncherConfig); err != nil {
		return fmt.Errorf("saving launcher config: %w", err)
	}
	if state.GatewayAutostart {
		state.GatewayService = installGatewayUserService(configPath)
	}

	fmt.Print(buildOnboardingMessage(cfg, configPath, launcherPath, updated, state))
	return nil
}

func buildOnboardingMessage(
	cfg *config.Config,
	configPath string,
	launcherPath string,
	updated bool,
	state onboardingState,
) string {
	title := "Setup Complete"
	if updated {
		title = "Setup Updated"
	}

	modelAlias := cfg.Agents.Defaults.GetModelName()
	modelTarget := "(not set)"
	if modelCfg, err := cfg.GetModelConfig(modelAlias); err == nil && modelCfg != nil && modelCfg.Model != "" {
		modelTarget = modelCfg.Model
	}

	safety := buildSafetyNotes(cfg, state)
	var b strings.Builder
	b.WriteString(renderBanner(
		title,
		"NookClaw workspace, launcher, and runtime profile are ready.",
	))
	b.WriteString("\n")

	b.WriteString(renderSummarySection("Configuration"))
	fmt.Fprintf(&b, "  Config:         %s\n", configPath)
	fmt.Fprintf(&b, "  Workspace:      %s\n", cfg.WorkspacePath())
	fmt.Fprintf(&b, "  Launcher:       %s (%s)\n", launcherPath, launcherAccessLabel(state.LauncherConfig))
	fmt.Fprintln(&b, "  Note:           Launcher access is stored separately from config.json")
	fmt.Fprintf(&b, "  Gateway:        %s:%d\n", cfg.Gateway.Host, cfg.Gateway.Port)

	fmt.Fprintln(&b)
	b.WriteString(renderSummarySection("Runtime Profile"))
	fmt.Fprintf(&b, "  Setup mode:     %s\n", valueOrFallback(state.SetupMode, quickStartMode))
	fmt.Fprintf(&b, "  Provider:       %s\n", valueOrFallback(cfg.Agents.Defaults.Provider, "(not set)"))
	fmt.Fprintf(&b, "  Model alias:    %s\n", valueOrFallback(modelAlias, "(not set)"))
	fmt.Fprintf(&b, "  Model target:   %s\n", modelTarget)
	fmt.Fprintf(&b, "  Channels:       %s\n", enabledChannelsSummary(cfg))
	fmt.Fprintf(&b, "  Web tools:      %s\n", statusLabel(cfg.Tools.Web.Enabled))
	fmt.Fprintf(&b, "  Scheduler:      %s\n", statusLabel(cfg.Tools.Cron.Enabled))
	fmt.Fprintf(&b, "  Heartbeat:      %s\n", statusLabel(cfg.Heartbeat.Enabled))
	fmt.Fprintf(&b, "  Remote exec:    %s\n", statusLabel(cfg.Tools.Exec.AllowRemote))

	if hasInboundChannels(cfg) || state.GatewayAutostart {
		fmt.Fprintln(&b)
		b.WriteString(renderSummarySection("Gateway Runtime"))
		if state.GatewayAutostart {
			if state.GatewayService.Enabled {
				fmt.Fprintln(&b, "  Mode:           systemd user service")
				fmt.Fprintf(&b, "  Unit:           %s\n", state.GatewayService.UnitPath)
				fmt.Fprintln(&b, "  Status:         enabled and started")
				if state.GatewayService.LingerHint != "" {
					fmt.Fprintf(&b, "  Boot note:      %s\n", state.GatewayService.LingerHint)
				}
			} else {
				fmt.Fprintln(&b, "  Mode:           systemd user service requested")
				if state.GatewayService.UnitPath != "" {
					fmt.Fprintf(&b, "  Unit:           %s\n", state.GatewayService.UnitPath)
				}
				if state.GatewayService.Error != "" {
					fmt.Fprintf(&b, "  Status:         %s\n", state.GatewayService.Error)
				}
				if state.GatewayService.LingerHint != "" {
					fmt.Fprintf(&b, "  Boot note:      %s\n", state.GatewayService.LingerHint)
				}
			}
		} else {
			fmt.Fprintln(&b, "  Mode:           manual")
			fmt.Fprintln(&b, "  Command:        nookclaw gateway")
			fmt.Fprintln(&b, "  Note:           Inbound channels only receive messages while the gateway is running.")
		}
	}

	if len(safety) > 0 {
		fmt.Fprintln(&b)
		b.WriteString(renderSummarySection("Safety Notes"))
		for _, note := range safety {
			fmt.Fprintf(&b, "  - %s\n", note)
		}
	}

	fmt.Fprintln(&b)
	b.WriteString(renderSummarySection("Next Steps"))
	fmt.Fprintln(&b, "  1. Review the generated setup")
	fmt.Fprintln(&b, "     nookclaw status")
	fmt.Fprintln(&b, "     nookclaw model")
	fmt.Fprintln(&b)
	nextStep := 2
	if hasInboundChannels(cfg) || state.GatewayAutostart {
		if state.GatewayAutostart && state.GatewayService.Enabled {
			fmt.Fprintf(&b, "  %d. Verify the background gateway service\n", nextStep)
			fmt.Fprintln(&b, "     systemctl --user status nookclaw-gateway")
		} else if state.GatewayAutostart {
			fmt.Fprintf(&b, "  %d. Finish enabling the background gateway service\n", nextStep)
			fmt.Fprintln(&b, "     systemctl --user daemon-reload")
			fmt.Fprintln(&b, "     systemctl --user enable --now nookclaw-gateway")
		} else {
			fmt.Fprintf(&b, "  %d. Start the gateway for inbound channels\n", nextStep)
			fmt.Fprintln(&b, "     nookclaw gateway")
		}
		fmt.Fprintln(&b)
		nextStep++
	}
	fmt.Fprintf(&b, "  %d. Start a first chat\n", nextStep)
	fmt.Fprintln(&b, "     nookclaw agent -m \"hello\"")
	fmt.Fprintln(&b)
	nextStep++
	fmt.Fprintf(&b, "  %d. Open the web launcher\n", nextStep)
	fmt.Fprintln(&b, "     make build-launcher && ./build/nookclaw-launcher")

	if state.DetectedOpenClaw != "" {
		fmt.Fprintln(&b)
		b.WriteString(renderSummarySection("Migration"))
		fmt.Fprintf(&b, "  OpenClaw setup detected at %s\n", state.DetectedOpenClaw)
		fmt.Fprintln(&b, "  Import it later with:")
		fmt.Fprintln(&b, "    nookclaw migrate --from openclaw")
	}

	if state.CredentialHint != "" {
		fmt.Fprintln(&b)
		b.WriteString(renderSummarySection("Credentials"))
		fmt.Fprintf(&b, "  %s\n", state.CredentialHint)
	}

	return b.String()
}

func buildSafetyNotes(cfg *config.Config, state onboardingState) []string {
	var notes []string
	if state.LauncherConfig.Public {
		notes = append(notes, "Launcher access is open to the local network. Review your host firewall and channel allowlists before wider exposure.")
	}
	if cfg.Tools.Exec.AllowRemote {
		notes = append(notes, "Remote exec is enabled. Keep this setup limited to trusted operators and trusted prompts.")
	}
	if enabledChannelsSummary(cfg) != "none enabled" {
		notes = append(notes, "At least one inbound channel is enabled. Confirm the token scope and message allowlists before daily use.")
	}
	if cfg.Channels.Telegram.Enabled && len(cfg.Channels.Telegram.AllowFrom) == 0 {
		notes = append(notes, "Telegram allow_from is empty. Add specific IDs or usernames if you do not want the bot reachable by any Telegram user.")
	}
	if cfg.Tools.Web.Enabled {
		notes = append(notes, "Web tools are enabled. Prompts may cause outbound fetches or searches when the runtime decides they are useful.")
	}
	return notes
}

func hasInboundChannels(cfg *config.Config) bool {
	return enabledChannelsSummary(cfg) != "none enabled"
}

func enabledChannelsSummary(cfg *config.Config) string {
	var enabled []string
	if cfg.Channels.Telegram.Enabled {
		enabled = append(enabled, "Telegram")
	}
	if cfg.Channels.Discord.Enabled {
		enabled = append(enabled, "Discord")
	}
	if cfg.Channels.Slack.Enabled {
		enabled = append(enabled, "Slack")
	}
	if cfg.Channels.Matrix.Enabled {
		enabled = append(enabled, "Matrix")
	}
	if cfg.Channels.LINE.Enabled {
		enabled = append(enabled, "LINE")
	}
	if cfg.Channels.OneBot.Enabled {
		enabled = append(enabled, "OneBot")
	}
	if cfg.Channels.QQ.Enabled {
		enabled = append(enabled, "QQ")
	}
	if cfg.Channels.WeCom.Enabled || cfg.Channels.WeComApp.Enabled || cfg.Channels.WeComAIBot.Enabled {
		enabled = append(enabled, "WeCom")
	}
	if cfg.Channels.DingTalk.Enabled {
		enabled = append(enabled, "DingTalk")
	}
	if cfg.Channels.WhatsApp.Enabled {
		enabled = append(enabled, "WhatsApp")
	}
	if cfg.Channels.Pico.Enabled {
		enabled = append(enabled, "Pico")
	}
	if cfg.Channels.MaixCam.Enabled {
		enabled = append(enabled, "MaixCam")
	}
	if cfg.Channels.Feishu.Enabled {
		enabled = append(enabled, "Feishu")
	}
	if cfg.Channels.IRC.Enabled {
		enabled = append(enabled, "IRC")
	}
	if len(enabled) == 0 {
		return "none enabled"
	}
	return strings.Join(enabled, ", ")
}

func statusLabel(enabled bool) string {
	if enabled {
		return "enabled"
	}
	return "disabled"
}

func valueOrFallback(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func createWorkspaceTemplates(workspace string) {
	err := copyEmbeddedToTarget(workspace)
	if err != nil {
		fmt.Printf("Error copying workspace templates: %v\n", err)
	}
}

func copyEmbeddedToTarget(targetDir string) error {
	// Ensure target directory exists
	if err := os.MkdirAll(targetDir, 0o700); err != nil {
		return fmt.Errorf("Failed to create target directory: %w", err)
	}

	// Walk through all files in embed.FS
	err := fs.WalkDir(embeddedFiles, "workspace", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if d.IsDir() {
			return nil
		}

		// Read embedded file
		data, err := embeddedFiles.ReadFile(path)
		if err != nil {
			return fmt.Errorf("Failed to read embedded file %s: %w", path, err)
		}

		new_path, err := filepath.Rel("workspace", path)
		if err != nil {
			return fmt.Errorf("Failed to get relative path for %s: %v\n", path, err)
		}

		// Build target file path
		targetPath := filepath.Join(targetDir, new_path)

		// Ensure target file's directory exists
		if err := os.MkdirAll(filepath.Dir(targetPath), 0o700); err != nil {
			return fmt.Errorf("Failed to create directory %s: %w", filepath.Dir(targetPath), err)
		}

		// Write file
		if err := os.WriteFile(targetPath, data, 0o600); err != nil {
			return fmt.Errorf("Failed to write file %s: %w", targetPath, err)
		}

		return nil
	})

	return err
}
