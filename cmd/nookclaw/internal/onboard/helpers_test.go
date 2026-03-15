package onboard

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/samnoadd/NookClaw/pkg/config"
	"github.com/samnoadd/NookClaw/web/backend/launcherconfig"
)

func TestCopyEmbeddedToTargetUsesAgentsMarkdown(t *testing.T) {
	targetDir := t.TempDir()

	if err := copyEmbeddedToTarget(targetDir); err != nil {
		t.Fatalf("copyEmbeddedToTarget() error = %v", err)
	}

	agentsPath := filepath.Join(targetDir, "AGENTS.md")
	if _, err := os.Stat(agentsPath); err != nil {
		t.Fatalf("expected %s to exist: %v", agentsPath, err)
	}

	legacyPath := filepath.Join(targetDir, "AGENT.md")
	if _, err := os.Stat(legacyPath); !os.IsNotExist(err) {
		t.Fatalf("expected legacy file %s to be absent, got err=%v", legacyPath, err)
	}
}

func TestBuildOnboardingMessage_GuidedSummary(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Gateway.Host = "127.0.0.1"
	cfg.Gateway.Port = 18790
	cfg.Channels.Telegram.Enabled = true
	cfg.Channels.Telegram.Token = "telegram-token"

	msg := buildOnboardingMessage(
		cfg,
		"/tmp/nookclaw/config.json",
		"/tmp/nookclaw/launcher-config.json",
		false,
		onboardingState{
			SetupMode:      quickStartMode,
			LauncherConfig: launcherconfig.Default(),
		},
	)

	for _, snippet := range []string{
		"Setup Complete",
		"NookClaw workspace, launcher, and runtime profile are ready.",
		"Configuration",
		"Launcher access is stored separately from config.json",
		"Runtime Profile",
		"Gateway Runtime",
		"nookclaw gateway",
		"Next Steps",
		"Quick Start",
		"nookclaw status",
		"nookclaw agent -m \"hello\"",
	} {
		if !strings.Contains(msg, snippet) {
			t.Fatalf("expected onboarding message to contain %q\nmessage:\n%s", snippet, msg)
		}
	}

	for _, unwanted := range []string{
		"personal fork starts in local-first mode",
		"existing ~/.picoclaw installs are still detected automatically",
	} {
		if strings.Contains(msg, unwanted) {
			t.Fatalf("did not expect onboarding message to contain %q\nmessage:\n%s", unwanted, msg)
		}
	}
}

func TestBuildOnboardingMessage_IncludesMigrationAndCredentialSections(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.Provider = "openai"
	cfg.Agents.Defaults.ModelName = "gpt-5.4"

	msg := buildOnboardingMessage(
		cfg,
		"/tmp/nookclaw/config.json",
		"/tmp/nookclaw/launcher-config.json",
		true,
		onboardingState{
			SetupMode:        advancedMode,
			DetectedOpenClaw: "/tmp/.openclaw",
			LauncherConfig: launcherconfig.Config{
				Port:   18800,
				Public: true,
			},
			CredentialHint: "Add your OpenAI API key to the generated config before the first chat.",
		},
	)

	for _, snippet := range []string{
		"Setup Updated",
		"Safety Notes",
		"Migration",
		"/tmp/.openclaw",
		"nookclaw migrate --from openclaw",
		"Credentials",
		"OpenAI API key",
		"local network on port 18800",
	} {
		if !strings.Contains(msg, snippet) {
			t.Fatalf("expected onboarding message to contain %q\nmessage:\n%s", snippet, msg)
		}
	}
}

func TestBuildOnboardingMessage_GatewayServiceEnabled(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Channels.Telegram.Enabled = true
	cfg.Channels.Telegram.Token = "telegram-token"
	cfg.Channels.Telegram.AllowFrom = config.FlexibleStringSlice{"123456789"}

	msg := buildOnboardingMessage(
		cfg,
		"/tmp/nookclaw/config.json",
		"/tmp/nookclaw/launcher-config.json",
		false,
		onboardingState{
			SetupMode:        advancedMode,
			LauncherConfig:   launcherconfig.Default(),
			GatewayAutostart: true,
			GatewayService: gatewayServiceSetup{
				UnitPath:   "/tmp/systemd/user/nookclaw-gateway.service",
				Enabled:    true,
				LingerHint: "To start before login, run: sudo loginctl enable-linger shayea",
			},
		},
	)

	for _, snippet := range []string{
		"Gateway Runtime",
		"systemd user service",
		"/tmp/systemd/user/nookclaw-gateway.service",
		"enabled and started",
		"systemctl --user status nookclaw-gateway",
		"sudo loginctl enable-linger shayea",
	} {
		if !strings.Contains(msg, snippet) {
			t.Fatalf("expected onboarding message to contain %q\nmessage:\n%s", snippet, msg)
		}
	}
}

func TestEnabledChannelsSummary(t *testing.T) {
	cfg := config.DefaultConfig()
	if got := enabledChannelsSummary(cfg); got != "none enabled" {
		t.Fatalf("enabledChannelsSummary() = %q, want %q", got, "none enabled")
	}

	cfg.Channels.Telegram.Enabled = true
	cfg.Channels.Matrix.Enabled = true
	cfg.Channels.WeCom.Enabled = true
	if got := enabledChannelsSummary(cfg); got != "Telegram, Matrix, WeCom" {
		t.Fatalf("enabledChannelsSummary() = %q, want %q", got, "Telegram, Matrix, WeCom")
	}
}

func TestOnboard_NonInteractivePersistsConfigAndLauncher(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("NOOKCLAW_HOME", tempHome)
	t.Setenv("PICOCLAW_HOME", "")
	t.Setenv("NOOKCLAW_CONFIG", "")
	t.Setenv("PICOCLAW_CONFIG", "")

	if err := onboard(onboardOptions{
		NonInteractive: true,
		Advanced:       true,
		Provider:       "openai",
		APIKey:         "openai-key",
		Channel:        "telegram",
		ChannelSecret:  "telegram-token",
		LauncherPublic: true,
	}); err != nil {
		t.Fatalf("onboard() error = %v", err)
	}

	configPath := filepath.Join(tempHome, "config.json")
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if cfg.Agents.Defaults.Provider != "openai" {
		t.Fatalf("Provider = %q, want %q", cfg.Agents.Defaults.Provider, "openai")
	}
	if cfg.Agents.Defaults.ModelName != "gpt-5.4" {
		t.Fatalf("ModelName = %q, want %q", cfg.Agents.Defaults.ModelName, "gpt-5.4")
	}
	if cfg.Providers.OpenAI.APIKey != "openai-key" {
		t.Fatalf("OpenAI API key = %q, want %q", cfg.Providers.OpenAI.APIKey, "openai-key")
	}
	if !cfg.Channels.Telegram.Enabled || cfg.Channels.Telegram.Token != "telegram-token" {
		t.Fatal("expected Telegram channel settings to be persisted")
	}

	launcherPath := launcherconfig.PathForAppConfig(configPath)
	launcherCfg, err := launcherconfig.Load(launcherPath, launcherconfig.Default())
	if err != nil {
		t.Fatalf("launcherconfig.Load() error = %v", err)
	}
	if !launcherCfg.Public {
		t.Fatal("expected launcher config to persist public access")
	}
}
