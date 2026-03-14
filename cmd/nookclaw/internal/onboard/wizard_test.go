package onboard

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/samnoadd/NookClaw/pkg/config"
)

func disableOllamaDiscovery(t *testing.T) {
	t.Helper()
	original := discoverOllamaModelsFn
	discoverOllamaModelsFn = func() []string { return nil }
	t.Cleanup(func() {
		discoverOllamaModelsFn = original
	})
}

func TestSetupWizardRun_AdvancedFlow(t *testing.T) {
	disableOllamaDiscovery(t)
	cfg := config.DefaultConfig()
	var out bytes.Buffer

	input := strings.NewReader("1\n2\n4\ny\nanthropic-key\ny\nn\ny\ny\n2\ntelegram-token\n2\n")
	state, err := newSetupWizard(input, &out, true, onboardOptions{}).run(cfg)
	if err != nil {
		t.Fatalf("run() error = %v", err)
	}

	if state.SetupMode != advancedMode {
		t.Fatalf("SetupMode = %q, want %q", state.SetupMode, advancedMode)
	}
	if cfg.Agents.Defaults.Provider != "anthropic" {
		t.Fatalf("Provider = %q, want %q", cfg.Agents.Defaults.Provider, "anthropic")
	}
	if cfg.Agents.Defaults.ModelName != "claude-sonnet-4.6" {
		t.Fatalf("ModelName = %q, want %q", cfg.Agents.Defaults.ModelName, "claude-sonnet-4.6")
	}
	if cfg.Providers.Anthropic.APIKey != "anthropic-key" {
		t.Fatalf("Anthropic API key = %q, want %q", cfg.Providers.Anthropic.APIKey, "anthropic-key")
	}
	if !cfg.Tools.Web.Enabled {
		t.Fatal("expected web tools to be enabled")
	}
	if cfg.Tools.Cron.Enabled {
		t.Fatal("expected scheduler to remain disabled")
	}
	if !cfg.Heartbeat.Enabled {
		t.Fatal("expected heartbeat to be enabled")
	}
	if !cfg.Tools.Exec.AllowRemote {
		t.Fatal("expected remote exec to be enabled")
	}
	if !cfg.Channels.Telegram.Enabled || cfg.Channels.Telegram.Token != "telegram-token" {
		t.Fatal("expected Telegram channel to be configured")
	}
	if state.ConfiguredChannel != "Telegram" {
		t.Fatalf("ConfiguredChannel = %q, want %q", state.ConfiguredChannel, "Telegram")
	}
	if !state.LauncherConfig.Public {
		t.Fatal("expected launcher to be public on the local network")
	}
	if state.CredentialHint != "" {
		t.Fatalf("CredentialHint = %q, want empty after saving API key", state.CredentialHint)
	}

	output := out.String()
	for _, snippet := range []string{
		"Security brief",
		"Setup mode",
		"Choose a setup profile:",
		"1. Model selection",
		"Select a model:",
		"Add your Anthropic API key now?",
		"Telegram setup",
		"@BotFather",
		"3. Channel access",
		"4. Launcher access",
	} {
		if !strings.Contains(output, snippet) {
			t.Fatalf("expected wizard output to contain %q\noutput:\n%s", snippet, output)
		}
	}
}

func TestSetupWizardRun_QuickStartMentionsDetectedOpenClaw(t *testing.T) {
	disableOllamaDiscovery(t)
	tempHome := t.TempDir()
	openClawHome := filepath.Join(tempHome, ".openclaw")
	if err := os.MkdirAll(openClawHome, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(openClawHome, "config.json"), []byte("{}\n"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	t.Setenv("HOME", tempHome)
	t.Setenv("OPENCLAW_HOME", "")

	cfg := config.DefaultConfig()
	var out bytes.Buffer
	state, err := newSetupWizard(strings.NewReader("1\n1\n1\n1\n1\n"), &out, true, onboardOptions{}).run(cfg)
	if err != nil {
		t.Fatalf("run() error = %v", err)
	}

	if state.SetupMode != quickStartMode {
		t.Fatalf("SetupMode = %q, want %q", state.SetupMode, quickStartMode)
	}
	if state.DetectedOpenClaw != openClawHome {
		t.Fatalf("DetectedOpenClaw = %q, want %q", state.DetectedOpenClaw, openClawHome)
	}
	if state.ConfiguredChannel != "none" {
		t.Fatalf("ConfiguredChannel = %q, want %q", state.ConfiguredChannel, "none")
	}
	if state.LauncherConfig.Public {
		t.Fatal("expected launcher to remain local-only by default")
	}
	if cfg.Agents.Defaults.Provider != "ollama" {
		t.Fatalf("Provider = %q, want %q", cfg.Agents.Defaults.Provider, "ollama")
	}

	output := out.String()
	for _, snippet := range []string{
		"Security brief",
		"OpenClaw content was found",
		"nookclaw migrate --from openclaw",
		"Select a model:",
		"Select a channel:",
	} {
		if !strings.Contains(output, snippet) {
			t.Fatalf("expected wizard output to contain %q\noutput:\n%s", snippet, output)
		}
	}
}

func TestSetupWizardRun_NonInteractiveFlags(t *testing.T) {
	disableOllamaDiscovery(t)
	cfg := config.DefaultConfig()
	state, err := newSetupWizard(
		strings.NewReader(""),
		&bytes.Buffer{},
		false,
		onboardOptions{
			Advanced:       true,
			Provider:       "openai",
			APIKey:         "openai-key",
			Channel:        "telegram",
			ChannelSecret:  "telegram-token",
			LauncherPublic: true,
		},
	).run(cfg)
	if err != nil {
		t.Fatalf("run() error = %v", err)
	}

	if state.SetupMode != advancedMode {
		t.Fatalf("SetupMode = %q, want %q", state.SetupMode, advancedMode)
	}
	if !state.LauncherConfig.Public {
		t.Fatal("expected launcher to be public")
	}
	if cfg.Agents.Defaults.Provider != "openai" || cfg.Agents.Defaults.ModelName != "gpt-5.4" {
		t.Fatal("expected OpenAI provider defaults to be applied")
	}
	if cfg.Providers.OpenAI.APIKey != "openai-key" {
		t.Fatalf("OpenAI API key = %q, want %q", cfg.Providers.OpenAI.APIKey, "openai-key")
	}
	if !cfg.Channels.Telegram.Enabled || cfg.Channels.Telegram.Token != "telegram-token" {
		t.Fatal("expected Telegram channel to be configured")
	}
	if state.CredentialHint != "" {
		t.Fatalf("CredentialHint = %q, want empty after saving API key", state.CredentialHint)
	}
}

func TestValidateOnboardOptions(t *testing.T) {
	if err := validateOnboardOptions(onboardOptions{
		NonInteractive: true,
		Provider:       "openai",
		APIKey:         "sk-test",
		Channel:        "matrix",
		ChannelSecret:  "matrix-token",
		ChannelUserID:  "@bot:matrix.org",
		LauncherPublic: true,
	}); err != nil {
		t.Fatalf("validateOnboardOptions() unexpected error = %v", err)
	}

	err := validateOnboardOptions(onboardOptions{
		NonInteractive: true,
		Channel:        "matrix",
		ChannelSecret:  "matrix-token",
	})
	if err == nil || !strings.Contains(err.Error(), "--channel matrix requires --channel-secret and --channel-user-id") {
		t.Fatalf("validateOnboardOptions() error = %v, want missing matrix user ID error", err)
	}
}

func TestHandleExistingSetup_InteractiveKeep(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(configPath, []byte("{}\n"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	var out bytes.Buffer
	updated, proceed := handleExistingSetup(configPath, true, false, strings.NewReader("1\n"), &out)

	if updated {
		t.Fatal("expected updated to be false when keeping existing setup")
	}
	if proceed {
		t.Fatal("expected proceed to be false when keeping existing setup")
	}
	if !strings.Contains(out.String(), "Keep existing setup") {
		t.Fatalf("expected keep prompt in output\noutput:\n%s", out.String())
	}
	if !strings.Contains(out.String(), "Existing setup detected") {
		t.Fatalf("expected existing setup status in output\noutput:\n%s", out.String())
	}
}

func TestHandleExistingSetup_NonInteractiveAbort(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(configPath, []byte("{}\n"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	var out bytes.Buffer
	updated, proceed := handleExistingSetup(configPath, false, false, strings.NewReader(""), &out)

	if updated {
		t.Fatal("expected updated to be false")
	}
	if proceed {
		t.Fatal("expected proceed to be false")
	}
	if !strings.Contains(out.String(), "`--force`") {
		t.Fatalf("expected non-interactive guidance in output\noutput:\n%s", out.String())
	}
}

func TestHandleExistingSetup_Force(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(configPath, []byte("{}\n"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	var out bytes.Buffer
	updated, proceed := handleExistingSetup(configPath, false, true, strings.NewReader(""), &out)

	if !updated || !proceed {
		t.Fatalf("expected force mode to continue, got updated=%v proceed=%v", updated, proceed)
	}
	if !strings.Contains(out.String(), "Replacing the current setup") {
		t.Fatalf("expected force message in output\noutput:\n%s", out.String())
	}
}

func TestRawTTYScreenUsesCRLF(t *testing.T) {
	got := rawTTYScreen("line one\nline two\n")
	want := "line one\r\nline two\r\n"
	if got != want {
		t.Fatalf("rawTTYScreen() = %q, want %q", got, want)
	}
}

func TestBuildModelChoices_PrefersDiscoveredOllamaModels(t *testing.T) {
	original := discoverOllamaModelsFn
	discoverOllamaModelsFn = func() []string { return []string{"llama3.2:3b", "qwen2.5:3b"} }
	defer func() {
		discoverOllamaModelsFn = original
	}()

	cfg := config.DefaultConfig()
	choices, defaultKey := buildModelChoices(cfg)

	if len(choices) == 0 {
		t.Fatal("expected model choices")
	}
	if choices[0].Key != "local:llama3.2:3b" {
		t.Fatalf("first choice = %q, want local Ollama model first", choices[0].Key)
	}
	if defaultKey == "" {
		t.Fatal("expected a default key")
	}
	for _, choice := range choices {
		if choice.Key == "alias:private-local" {
			t.Fatal("did not expect alias:private-local when discovered Ollama models are present")
		}
	}
}

func TestSecretPromptLabelMarksHiddenInput(t *testing.T) {
	got := stripANSI(secretPromptLabel("Telegram bot token", true))
	want := "Telegram bot token [input hidden]"
	if got != want {
		t.Fatalf("secretPromptLabel() = %q, want %q", got, want)
	}
}
