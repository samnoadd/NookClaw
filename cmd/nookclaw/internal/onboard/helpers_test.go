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
		"Runtime Profile",
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
