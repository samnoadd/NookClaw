package onboard

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/samnoadd/NookClaw/pkg/config"
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

	msg := buildOnboardingMessage(cfg, "/tmp/nookclaw/config.json", false)

	for _, snippet := range []string{
		"NookClaw Setup Complete",
		"Created",
		"Default Runtime",
		"Recommended Next Steps",
		"nookclaw status",
		"nookclaw agent -m \"hello\"",
		"nookclaw migrate --from openclaw",
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
