package onboard

import (
	"strings"
	"testing"
)

func TestRenderSelectorScreen_ModernLayout(t *testing.T) {
	screen := renderSelectorScreen(
		[]selectorLine{
			{
				Text: styleSuccess("✔") + " " + stylePrimary("Existing setup detected"),
				Role: "raw",
			},
			{
				Text: "  ~/.nookclaw/config.json",
				Role: "secondary",
			},
		},
		"Choose an action:",
		[]wizardOption{
			{
				Key:         "keep",
				Label:       "Keep existing setup",
				Description: "Leave the current config and workspace unchanged.",
			},
			{
				Key:         "reset",
				Label:       "Reset and start fresh",
				Description: "Replace the config and starter workspace files with a fresh onboarding setup.",
				Tone:        "warning",
			},
		},
		0,
	)

	for _, snippet := range []string{
		"🐾",
		"NookClaw",
		"Existing setup detected",
		"~/.nookclaw/config.json",
		"Choose an action:",
		"Keep existing setup",
		"Reset and start fresh",
		"Press ↑ ↓ to navigate • Enter to confirm",
	} {
		if !strings.Contains(screen, snippet) {
			t.Fatalf("expected selector screen to contain %q\nscreen:\n%s", snippet, screen)
		}
	}
}
