package onboard

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/samnoadd/NookClaw/cmd/nookclaw/internal"
	"github.com/samnoadd/NookClaw/pkg/config"
)

func onboard() {
	configPath := internal.GetDefaultConfigPath()
	updated := false

	if _, err := os.Stat(configPath); err == nil {
		fmt.Printf("A NookClaw config already exists at %s\n", configPath)
		fmt.Print("Replace it with a fresh onboarding setup? (y/N): ")
		var response string
		fmt.Scanln(&response)
		if !strings.EqualFold(strings.TrimSpace(response), "y") &&
			!strings.EqualFold(strings.TrimSpace(response), "yes") {
			fmt.Println("Aborted.")
			return
		}
		updated = true
	}

	cfg := config.DefaultConfig()
	if err := config.SaveConfig(configPath, cfg); err != nil {
		fmt.Printf("Error saving config: %v\n", err)
		os.Exit(1)
	}

	workspace := cfg.WorkspacePath()
	createWorkspaceTemplates(workspace)

	fmt.Print(buildOnboardingMessage(cfg, configPath, updated))
}

func buildOnboardingMessage(cfg *config.Config, configPath string, updated bool) string {
	title := "Setup Complete"
	if updated {
		title = "Setup Updated"
	}

	modelAlias := cfg.Agents.Defaults.GetModelName()
	modelTarget := "(not set)"
	if modelCfg, err := cfg.GetModelConfig(modelAlias); err == nil && modelCfg != nil && modelCfg.Model != "" {
		modelTarget = modelCfg.Model
	}

	var b strings.Builder
	fmt.Fprintf(&b, "%s NookClaw %s\n\n", internal.Logo, title)

	fmt.Fprintln(&b, "Created")
	fmt.Fprintf(&b, "  Config:       %s\n", configPath)
	fmt.Fprintf(&b, "  Workspace:    %s\n", cfg.WorkspacePath())
	fmt.Fprintf(&b, "  Gateway:      %s:%d\n", cfg.Gateway.Host, cfg.Gateway.Port)

	fmt.Fprintln(&b)
	fmt.Fprintln(&b, "Default Runtime")
	fmt.Fprintf(&b, "  Model alias:  %s\n", valueOrFallback(modelAlias, "(not set)"))
	fmt.Fprintf(&b, "  Model target: %s\n", modelTarget)
	fmt.Fprintf(&b, "  Channels:     %s\n", enabledChannelsSummary(cfg))
	fmt.Fprintf(&b, "  Web tools:    %s\n", statusLabel(cfg.Tools.Web.Enabled))
	fmt.Fprintf(&b, "  Scheduler:    %s\n", statusLabel(cfg.Tools.Cron.Enabled))
	fmt.Fprintf(&b, "  Heartbeat:    %s\n", statusLabel(cfg.Heartbeat.Enabled))
	fmt.Fprintf(&b, "  Remote exec:  %s\n", statusLabel(cfg.Tools.Exec.AllowRemote))

	fmt.Fprintln(&b)
	fmt.Fprintln(&b, "Recommended Next Steps")
	fmt.Fprintln(&b, "  1. Review the generated setup")
	fmt.Fprintln(&b, "     nookclaw status")
	fmt.Fprintln(&b, "     nookclaw model")
	fmt.Fprintln(&b)
	fmt.Fprintln(&b, "  2. Start a first chat")
	fmt.Fprintln(&b, "     nookclaw agent -m \"hello\"")
	fmt.Fprintln(&b)
	fmt.Fprintln(&b, "  3. Open the web launcher")
	fmt.Fprintln(&b, "     make build-launcher && ./build/nookclaw-launcher")
	fmt.Fprintln(&b)
	fmt.Fprintln(&b, "  4. Import an existing setup if needed")
	fmt.Fprintln(&b, "     nookclaw migrate --from openclaw")

	return b.String()
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
