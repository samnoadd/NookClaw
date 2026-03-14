package onboard

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/samnoadd/NookClaw/pkg/config"
	"github.com/samnoadd/NookClaw/web/backend/launcherconfig"
	"golang.org/x/term"
)

const (
	quickStartMode = "Quick Start"
	advancedMode   = "Advanced"
)

var errOnboardingAborted = errors.New("onboarding aborted")

type onboardOptions struct {
	NonInteractive    bool
	Advanced          bool
	Force             bool
	LauncherPublic    bool
	Provider          string
	APIKey            string
	Channel           string
	ChannelSecret     string
	ChannelAppToken   string
	ChannelUserID     string
	ChannelHomeserver string
}

type onboardingState struct {
	SetupMode         string
	DetectedOpenClaw  string
	LauncherConfig    launcherconfig.Config
	CredentialHint    string
	ConfiguredChannel string
}

type wizardOption struct {
	Key         string
	Label       string
	Description string
	Tone        string
}

type providerPreset struct {
	ID          string
	Label       string
	Alias       string
	Description string
	KeyName     string
}

type channelPreset struct {
	ID          string
	Label       string
	Description string
}

type setupWizard struct {
	in          io.Reader
	reader      *bufio.Reader
	out         io.Writer
	interactive bool
	opts        onboardOptions
	step        int
}

var providerPresets = []providerPreset{
	{
		ID:          "ollama",
		Label:       "Ollama on this machine",
		Alias:       "private-local",
		Description: "Use the local Ollama model configured by the `private-local` alias.",
	},
	{
		ID:          "openai",
		Label:       "OpenAI API",
		Alias:       "gpt-5.4",
		Description: "Start with the `gpt-5.4` alias and add your API key now or later.",
		KeyName:     "OpenAI API key",
	},
	{
		ID:          "anthropic",
		Label:       "Anthropic API",
		Alias:       "claude-sonnet-4.6",
		Description: "Start with the `claude-sonnet-4.6` alias and add your API key now or later.",
		KeyName:     "Anthropic API key",
	},
	{
		ID:          "openrouter",
		Label:       "OpenRouter",
		Alias:       "openrouter-auto",
		Description: "Start with the `openrouter-auto` alias and add your API key now or later.",
		KeyName:     "OpenRouter API key",
	},
}

var channelPresets = []channelPreset{
	{
		ID:          "telegram",
		Label:       "Telegram",
		Description: "Prompt for a bot token and enable the Telegram channel.",
	},
	{
		ID:          "discord",
		Label:       "Discord",
		Description: "Prompt for a bot token and enable the Discord channel.",
	},
	{
		ID:          "matrix",
		Label:       "Matrix",
		Description: "Prompt for homeserver, user ID, and access token.",
	},
	{
		ID:          "slack",
		Label:       "Slack",
		Description: "Prompt for a bot token and app token.",
	},
}

func newSetupWizard(in io.Reader, out io.Writer, interactive bool, opts onboardOptions) *setupWizard {
	if in == nil {
		in = strings.NewReader("")
	}
	if out == nil {
		out = io.Discard
	}
	return &setupWizard{
		in:          in,
		reader:      bufio.NewReader(in),
		out:         out,
		interactive: interactive,
		opts:        opts,
	}
}

func isInteractiveTerminal() bool {
	stdinInfo, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	stdoutInfo, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return stdinInfo.Mode()&os.ModeCharDevice != 0 && stdoutInfo.Mode()&os.ModeCharDevice != 0
}

func validateOnboardOptions(opts onboardOptions) error {
	provider := normalizeProvider(opts.Provider)
	channel := normalizeChannel(opts.Channel)

	if opts.Provider != "" && provider == "" {
		return fmt.Errorf("unsupported provider %q: use ollama, openai, anthropic, or openrouter", opts.Provider)
	}
	if opts.APIKey != "" && provider == "" {
		return fmt.Errorf("--api-key requires --provider openai, anthropic, or openrouter")
	}
	if provider == "ollama" && opts.APIKey != "" {
		return fmt.Errorf("--api-key cannot be used with --provider ollama")
	}

	if opts.Channel != "" && channel == "" {
		return fmt.Errorf("unsupported channel %q: use telegram, discord, matrix, or slack", opts.Channel)
	}
	if opts.ChannelSecret != "" && channel == "" {
		return fmt.Errorf("--channel-secret requires --channel")
	}
	if opts.ChannelAppToken != "" && channel != "slack" {
		return fmt.Errorf("--channel-app-token requires --channel slack")
	}
	if opts.ChannelUserID != "" && channel != "matrix" {
		return fmt.Errorf("--channel-user-id requires --channel matrix")
	}
	if opts.ChannelHomeserver != "" && channel != "matrix" {
		return fmt.Errorf("--channel-homeserver requires --channel matrix")
	}

	if opts.NonInteractive {
		switch channel {
		case "telegram", "discord":
			if strings.TrimSpace(opts.ChannelSecret) == "" {
				return fmt.Errorf("--channel %s requires --channel-secret in non-interactive mode", channel)
			}
		case "slack":
			if strings.TrimSpace(opts.ChannelSecret) == "" || strings.TrimSpace(opts.ChannelAppToken) == "" {
				return fmt.Errorf("--channel slack requires --channel-secret and --channel-app-token in non-interactive mode")
			}
		case "matrix":
			if strings.TrimSpace(opts.ChannelSecret) == "" || strings.TrimSpace(opts.ChannelUserID) == "" {
				return fmt.Errorf("--channel matrix requires --channel-secret and --channel-user-id in non-interactive mode")
			}
		}
	}

	return nil
}

func (o onboardOptions) preferAdvanced() bool {
	return o.Advanced ||
		o.Provider != "" ||
		o.APIKey != "" ||
		o.Channel != "" ||
		o.ChannelSecret != "" ||
		o.ChannelAppToken != "" ||
		o.ChannelUserID != "" ||
		o.ChannelHomeserver != "" ||
		o.LauncherPublic
}

func handleExistingSetup(
	configPath string,
	interactive bool,
	force bool,
	in io.Reader,
	out io.Writer,
) (bool, bool) {
	if _, err := os.Stat(configPath); err != nil {
		return false, true
	}

	if force {
		fmt.Fprint(out, renderCallout("Existing setup", []string{
			fmt.Sprintf("Replacing the current setup at %s.", configPath),
		}))
		fmt.Fprintln(out)
		return true, true
	}

	if !interactive {
		fmt.Fprint(out, renderCallout("Existing setup", []string{
			fmt.Sprintf("A NookClaw setup already exists at %s.", configPath),
			"Re-run with `--force` if you want to replace it in non-interactive mode.",
		}))
		return false, false
	}

	wizard := newSetupWizard(in, out, true, onboardOptions{})
	choice := wizard.promptChoiceWithContext(
		[]selectorLine{
			{
				Text: styleSuccess("✔") + " " + stylePrimary("Existing setup detected"),
				Role: "raw",
			},
			{
				Text: "  " + displayPath(configPath),
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
		"keep",
	)

	if choice == "keep" {
		fmt.Fprintln(out, "No changes made.")
		return false, false
	}

	return true, true
}

func (w *setupWizard) run(cfg *config.Config) (onboardingState, error) {
	state := onboardingState{
		SetupMode:         quickStartMode,
		DetectedOpenClaw:  detectOpenClawSource(),
		LauncherConfig:    launcherconfig.Default(),
		ConfiguredChannel: "none",
	}
	if w.opts.LauncherPublic {
		state.LauncherConfig.Public = true
	}

	forceAdvanced := w.opts.preferAdvanced()
	if forceAdvanced {
		state.SetupMode = advancedMode
	}

	if !w.interactive {
		if err := w.applyFlagSelections(cfg, &state); err != nil {
			return onboardingState{}, err
		}
		state.CredentialHint = credentialHint(cfg)
		return state, nil
	}

	if !w.presentSafetyBrief() {
		fmt.Fprintln(w.out, "Aborted.")
		return onboardingState{}, errOnboardingAborted
	}

	if !forceAdvanced {
		mode := w.promptChoiceWithContext(
			[]selectorLine{
				{Text: "Setup mode", Role: "primary"},
				{Text: "Choose the fastest way to start.", Role: "secondary"},
				{Text: "Advanced exposes provider, channel, launcher, and background settings now.", Role: "secondary"},
			},
			"Choose a setup profile:",
			[]wizardOption{
				{
					Key:         "quick",
					Label:       quickStartMode,
					Description: "Use the default runtime, starter workspace, and launcher settings.",
				},
				{
					Key:         "advanced",
					Label:       advancedMode,
					Description: "Choose the model backend, credentials, background features, and launcher visibility.",
				},
			},
			"quick",
		)
		state.SetupMode = labelForChoice(mode, quickStartMode, map[string]string{
			"quick":    quickStartMode,
			"advanced": advancedMode,
		})
	}

	if state.DetectedOpenClaw != "" && !w.selectorUIEnabled() {
		fmt.Fprint(w.out, renderCallout("Migration detected", []string{
			fmt.Sprintf("OpenClaw content was found at %s.", state.DetectedOpenClaw),
			"Import it later with `nookclaw migrate --from openclaw` after this setup completes.",
		}))
		fmt.Fprintln(w.out)
	}

	if state.SetupMode == advancedMode {
		w.printSection(
			"Runtime profile",
			"Choose the default model backend and add credentials now if you want this setup ready for an immediate first chat.",
		)
		if err := w.configureProvider(cfg); err != nil {
			return onboardingState{}, err
		}
		fmt.Fprintln(w.out)
		w.printSection(
			"Background features",
			"Enable only the automation and reachability features you plan to use on day one.",
		)
		cfg.Tools.Web.Enabled = w.promptYesNo("Enable web tools for search and page fetch?", cfg.Tools.Web.Enabled)
		cfg.Tools.Cron.Enabled = w.promptYesNo("Enable the scheduler for recurring tasks?", cfg.Tools.Cron.Enabled)
		cfg.Heartbeat.Enabled = w.promptYesNo("Enable heartbeat background checks?", cfg.Heartbeat.Enabled)
		cfg.Tools.Exec.AllowRemote = w.promptYesNo("Allow remote exec requests?", cfg.Tools.Exec.AllowRemote)
		fmt.Fprintln(w.out)
	} else if !w.selectorUIEnabled() {
		fmt.Fprint(w.out, renderCallout("Quick Start profile", []string{
			"Quick Start keeps the default model, starter workspace, and conservative background settings.",
			"You can re-run `nookclaw onboard --advanced --force` later if you want a fuller configuration pass.",
		}))
		fmt.Fprintln(w.out)
	}

	w.printSection(
		"Channel access",
		"Attach one inbound channel now if you want the assistant reachable outside the terminal. Leave it empty if you prefer to finish setup first.",
	)
	channelLabel, err := w.configureChannel(cfg)
	if err != nil {
		return onboardingState{}, err
	}
	state.ConfiguredChannel = channelLabel

	w.printSection(
		"Launcher access",
		"Choose whether the web launcher should stay on this machine or be reachable from the local network.",
	)
	state.LauncherConfig = w.configureLauncher(state.LauncherConfig)
	state.CredentialHint = credentialHint(cfg)
	return state, nil
}

func (w *setupWizard) applyFlagSelections(cfg *config.Config, state *onboardingState) error {
	if state == nil {
		return nil
	}
	if w.opts.preferAdvanced() {
		state.SetupMode = advancedMode
	}

	if err := w.configureProviderFromFlags(cfg); err != nil {
		return err
	}

	channelLabel, err := w.configureChannelFromFlags(cfg)
	if err != nil {
		return err
	}
	if channelLabel != "" {
		state.ConfiguredChannel = channelLabel
	}

	return nil
}

func (w *setupWizard) configureProvider(cfg *config.Config) error {
	providerID := normalizeProvider(w.opts.Provider)
	if providerID == "" {
		options := make([]wizardOption, 0, len(providerPresets))
		for index, preset := range providerPresets {
			options = append(options, wizardOption{
				Key:         fmt.Sprintf("%d", index+1),
				Label:       preset.Label,
				Description: preset.Description,
			})
		}
		choice := w.promptChoiceWithContext(
			[]selectorLine{
				{Text: "Runtime profile", Role: "primary"},
				{Text: "Choose the default model backend for this setup.", Role: "secondary"},
			},
			"Select a provider:",
			options,
			"1",
		)
		preset := providerPresets[0]
		switch choice {
		case "2":
			preset = providerPresets[1]
		case "3":
			preset = providerPresets[2]
		case "4":
			preset = providerPresets[3]
		}
		applyProviderPreset(cfg, preset)
		if requiresProviderKey(preset.ID) {
			key := strings.TrimSpace(w.opts.APIKey)
			if key == "" && w.promptYesNo(fmt.Sprintf("Add your %s now?", preset.KeyName), false) {
				key = w.promptSecret(preset.KeyName)
			}
			if key != "" {
				setProviderAPIKey(cfg, preset.ID, key)
			}
		}
		return nil
	}

	preset, ok := providerPresetByID(providerID)
	if !ok {
		return fmt.Errorf("unsupported provider %q", providerID)
	}
	applyProviderPreset(cfg, preset)
	if requiresProviderKey(preset.ID) {
		key := strings.TrimSpace(w.opts.APIKey)
		if key == "" && w.promptYesNo(fmt.Sprintf("Add your %s now?", preset.KeyName), false) {
			key = w.promptSecret(preset.KeyName)
		}
		if key != "" {
			setProviderAPIKey(cfg, preset.ID, key)
		}
	}
	return nil
}

func (w *setupWizard) configureProviderFromFlags(cfg *config.Config) error {
	providerID := normalizeProvider(w.opts.Provider)
	if providerID == "" {
		return nil
	}
	preset, ok := providerPresetByID(providerID)
	if !ok {
		return fmt.Errorf("unsupported provider %q", providerID)
	}
	applyProviderPreset(cfg, preset)
	if key := strings.TrimSpace(w.opts.APIKey); key != "" {
		setProviderAPIKey(cfg, preset.ID, key)
	}
	return nil
}

func (w *setupWizard) configureChannel(cfg *config.Config) (string, error) {
	channelID := normalizeChannel(w.opts.Channel)
	if channelID == "" {
		if !w.promptYesNo("Configure a chat channel now?", false) {
			return "none", nil
		}
		options := []wizardOption{
			{
				Key:         "1",
				Label:       channelPresets[0].Label,
				Description: channelPresets[0].Description,
			},
			{
				Key:         "2",
				Label:       channelPresets[1].Label,
				Description: channelPresets[1].Description,
			},
			{
				Key:         "3",
				Label:       channelPresets[2].Label,
				Description: channelPresets[2].Description,
			},
			{
				Key:         "4",
				Label:       channelPresets[3].Label,
				Description: channelPresets[3].Description,
			},
		}
		choice := w.promptChoiceWithContext(
			[]selectorLine{
				{Text: "Channel access", Role: "primary"},
				{Text: "Attach one inbound channel now, or skip it and finish setup first.", Role: "secondary"},
			},
			"Select a channel:",
			options,
			"1",
		)
		switch choice {
		case "2":
			channelID = channelPresets[1].ID
		case "3":
			channelID = channelPresets[2].ID
		case "4":
			channelID = channelPresets[3].ID
		default:
			channelID = channelPresets[0].ID
		}
	}

	label, err := w.configureSelectedChannel(cfg, channelID)
	if err != nil {
		return "", err
	}
	return label, nil
}

func (w *setupWizard) configureChannelFromFlags(cfg *config.Config) (string, error) {
	channelID := normalizeChannel(w.opts.Channel)
	if channelID == "" {
		return "", nil
	}
	return w.configureSelectedChannel(cfg, channelID)
}

func (w *setupWizard) configureSelectedChannel(cfg *config.Config, channelID string) (string, error) {
	switch channelID {
	case "telegram":
		token := strings.TrimSpace(w.opts.ChannelSecret)
		if token == "" {
			token = w.promptSecret("Telegram bot token")
		}
		if strings.TrimSpace(token) == "" {
			return "", fmt.Errorf("telegram channel requires a bot token")
		}
		cfg.Channels.Telegram.Enabled = true
		cfg.Channels.Telegram.Token = token
		return "Telegram", nil
	case "discord":
		token := strings.TrimSpace(w.opts.ChannelSecret)
		if token == "" {
			token = w.promptSecret("Discord bot token")
		}
		if strings.TrimSpace(token) == "" {
			return "", fmt.Errorf("discord channel requires a bot token")
		}
		cfg.Channels.Discord.Enabled = true
		cfg.Channels.Discord.Token = token
		return "Discord", nil
	case "matrix":
		homeserver := strings.TrimSpace(w.opts.ChannelHomeserver)
		if homeserver == "" {
			homeserver = cfg.Channels.Matrix.Homeserver
		}
		if w.interactive && strings.TrimSpace(w.opts.ChannelHomeserver) == "" {
			homeserver = w.promptText("Matrix homeserver", homeserver)
		}
		userID := strings.TrimSpace(w.opts.ChannelUserID)
		if userID == "" {
			userID = w.promptText("Matrix user ID", "")
		}
		token := strings.TrimSpace(w.opts.ChannelSecret)
		if token == "" {
			token = w.promptSecret("Matrix access token")
		}
		if strings.TrimSpace(userID) == "" || strings.TrimSpace(token) == "" {
			return "", fmt.Errorf("matrix channel requires a user ID and access token")
		}
		cfg.Channels.Matrix.Enabled = true
		cfg.Channels.Matrix.Homeserver = homeserver
		cfg.Channels.Matrix.UserID = userID
		cfg.Channels.Matrix.AccessToken = token
		return "Matrix", nil
	case "slack":
		botToken := strings.TrimSpace(w.opts.ChannelSecret)
		if botToken == "" {
			botToken = w.promptSecret("Slack bot token")
		}
		appToken := strings.TrimSpace(w.opts.ChannelAppToken)
		if appToken == "" {
			appToken = w.promptSecret("Slack app token")
		}
		if strings.TrimSpace(botToken) == "" || strings.TrimSpace(appToken) == "" {
			return "", fmt.Errorf("slack channel requires a bot token and app token")
		}
		cfg.Channels.Slack.Enabled = true
		cfg.Channels.Slack.BotToken = botToken
		cfg.Channels.Slack.AppToken = appToken
		return "Slack", nil
	default:
		return "", fmt.Errorf("unsupported channel %q", channelID)
	}
}

func (w *setupWizard) configureLauncher(current launcherconfig.Config) launcherconfig.Config {
	cfg := current
	if !w.interactive && w.opts.LauncherPublic {
		cfg.Public = true
		return cfg
	}

	if !w.interactive {
		return cfg
	}

	defaultChoice := "1"
	if cfg.Public {
		defaultChoice = "2"
	}
	choice := w.promptChoiceWithContext(
		[]selectorLine{
			{Text: "Launcher access", Role: "primary"},
			{Text: "Choose whether the web launcher stays on this machine or is reachable from your local network.", Role: "secondary"},
		},
		"Select launcher access:",
		[]wizardOption{
			{
				Key:         "1",
				Label:       "Local only",
				Description: "Expose the launcher only on this machine.",
			},
			{
				Key:         "2",
				Label:       "Local network",
				Description: "Allow other devices on your LAN to reach the launcher.",
			},
		},
		defaultChoice,
	)
	cfg.Public = choice == "2"
	return cfg
}

func (w *setupWizard) presentSafetyBrief() bool {
	choice := w.promptChoiceWithContext(
		[]selectorLine{
			{Text: "Security brief", Role: "primary"},
			{Text: "NookClaw can read files, call models, and expose channels when those capabilities are enabled.", Role: "secondary"},
			{Text: "Shared or network-visible setups should be treated as privileged automation.", Role: "secondary"},
			{Text: "Review the generated config before enabling remote access or connecting live channels.", Role: "secondary"},
		},
		"Continue with guided setup?",
		[]wizardOption{
			{
				Key:         "continue",
				Label:       "Continue setup",
				Description: "Create a fresh NookClaw configuration and starter workspace.",
			},
			{
				Key:         "cancel",
				Label:       "Cancel",
				Description: "Leave the current machine state unchanged.",
			},
		},
		"continue",
	)
	return choice == "continue"
}

func (w *setupWizard) printSection(title string, intro string) {
	if w.selectorUIEnabled() {
		return
	}
	w.step++
	fmt.Fprint(w.out, renderSectionHeader(w.step, title, intro))
}

func (w *setupWizard) promptChoice(title string, options []wizardOption, defaultKey string) string {
	return w.promptChoiceWithContext(
		[]selectorLine{{Text: title, Role: "primary"}},
		"Choose an option:",
		options,
		defaultKey,
	)
}

func (w *setupWizard) promptChoiceWithContext(context []selectorLine, actionLabel string, options []wizardOption, defaultKey string) string {
	if !w.interactive {
		return defaultKey
	}

	if w.selectorUIEnabled() {
		return w.promptChoiceSelector(context, actionLabel, options, defaultKey)
	}

	return w.promptChoiceFallback(context, actionLabel, options, defaultKey)
}

func (w *setupWizard) promptChoiceFallback(context []selectorLine, actionLabel string, options []wizardOption, defaultKey string) string {
	for _, line := range context {
		if strings.TrimSpace(line.Text) == "" {
			fmt.Fprintln(w.out)
			continue
		}
		fmt.Fprintln(w.out, stripANSI(line.Text))
	}
	fmt.Fprintln(w.out)
	fmt.Fprintln(w.out, stripANSI(actionLabel))
	defaultIndex := optionIndexForKey(options, defaultKey)
	for index, option := range options {
		fmt.Fprintf(w.out, "  %d. %s\n", index+1, option.Label)
		if option.Description != "" {
			fmt.Fprintf(w.out, "     %s\n", option.Description)
		}
	}
	fmt.Fprintf(w.out, "Select an option [%d]: ", defaultIndex+1)

	input := strings.TrimSpace(w.readLine())
	fmt.Fprintln(w.out)
	if input == "" {
		return defaultKey
	}
	for index, option := range options {
		if input == option.Key || input == fmt.Sprintf("%d", index+1) {
			return option.Key
		}
	}
	return defaultKey
}

func (w *setupWizard) promptYesNo(label string, defaultValue bool) bool {
	if !w.interactive {
		return defaultValue
	}

	if w.selectorUIEnabled() {
		defaultKey := "no"
		if defaultValue {
			defaultKey = "yes"
		}
		choice := w.promptChoiceWithContext(
			[]selectorLine{{Text: label, Role: "primary"}},
			"Select an option:",
			[]wizardOption{
				{
					Key:         "yes",
					Label:       "Yes",
					Description: "Apply this setting in the generated setup.",
				},
				{
					Key:         "no",
					Label:       "No",
					Description: "Leave this setting disabled for now.",
				},
			},
			defaultKey,
		)
		return choice == "yes"
	}

	defaultLabel := "y/N"
	if defaultValue {
		defaultLabel = "Y/n"
	}
	fmt.Fprintf(w.out, "%s [%s]: ", label, defaultLabel)
	input := strings.ToLower(strings.TrimSpace(w.readLine()))
	fmt.Fprintln(w.out)
	if input == "" {
		return defaultValue
	}
	return input == "y" || input == "yes"
}

func (w *setupWizard) promptText(label string, defaultValue string) string {
	if !w.interactive {
		return defaultValue
	}
	if defaultValue != "" {
		fmt.Fprintf(w.out, "%s %s: ", stylePrimary(label), styleSecondary("["+defaultValue+"]"))
	} else {
		fmt.Fprintf(w.out, "%s: ", stylePrimary(label))
	}
	input := strings.TrimSpace(w.readLine())
	fmt.Fprintln(w.out)
	if input == "" {
		return defaultValue
	}
	return input
}

func (w *setupWizard) promptSecret(label string) string {
	if !w.interactive {
		return ""
	}
	if file, ok := w.in.(*os.File); ok && term.IsTerminal(int(file.Fd())) {
		fmt.Fprintf(w.out, "%s: ", stylePrimary(label))
		value, err := term.ReadPassword(int(file.Fd()))
		fmt.Fprintln(w.out)
		if err == nil {
			return strings.TrimSpace(string(value))
		}
	}
	fmt.Fprintf(w.out, "%s: ", stylePrimary(label))
	value := strings.TrimSpace(w.readLine())
	fmt.Fprintln(w.out)
	return value
}

func (w *setupWizard) readLine() string {
	line, err := w.reader.ReadString('\n')
	if err != nil {
		return strings.TrimSpace(line)
	}
	return strings.TrimSpace(line)
}

func (w *setupWizard) selectorUIEnabled() bool {
	if !w.interactive {
		return false
	}

	inFile, ok := w.in.(*os.File)
	if !ok || !term.IsTerminal(int(inFile.Fd())) {
		return false
	}

	outFile, ok := w.out.(*os.File)
	if !ok || !term.IsTerminal(int(outFile.Fd())) {
		return false
	}

	return true
}

func (w *setupWizard) promptChoiceSelector(context []selectorLine, actionLabel string, options []wizardOption, defaultKey string) string {
	inFile := w.in.(*os.File)
	selected := optionIndexForKey(options, defaultKey)

	state, err := term.MakeRaw(int(inFile.Fd()))
	if err != nil {
		return w.promptChoiceFallback(context, actionLabel, options, defaultKey)
	}
	defer term.Restore(int(inFile.Fd()), state)

	for {
		clearInteractiveScreen(w.out)
		fmt.Fprint(w.out, renderSelectorScreen(context, actionLabel, options, selected))

		key, err := readSelectorKey(inFile)
		if err != nil {
			return options[selected].Key
		}

		switch key {
		case "up":
			if selected > 0 {
				selected--
			}
		case "down":
			if selected < len(options)-1 {
				selected++
			}
		case "enter":
			fmt.Fprintln(w.out)
			return options[selected].Key
		}
	}
}

func optionIndexForKey(options []wizardOption, defaultKey string) int {
	for index, option := range options {
		if option.Key == defaultKey {
			return index
		}
	}
	return 0
}

func clearInteractiveScreen(out io.Writer) {
	fmt.Fprint(out, "\033[H\033[2J")
}

func readSelectorKey(in *os.File) (string, error) {
	buf := make([]byte, 3)
	for {
		n, err := in.Read(buf[:1])
		if err != nil {
			return "", err
		}
		if n == 0 {
			continue
		}

		switch buf[0] {
		case '\r', '\n':
			return "enter", nil
		case 'k', 'K':
			return "up", nil
		case 'j', 'J':
			return "down", nil
		case 27:
			if _, err := io.ReadFull(in, buf[1:2]); err != nil {
				return "", err
			}
			if buf[1] != '[' {
				continue
			}
			if _, err := io.ReadFull(in, buf[2:3]); err != nil {
				return "", err
			}
			switch buf[2] {
			case 'A':
				return "up", nil
			case 'B':
				return "down", nil
			}
		}
	}
}

func displayPath(path string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}

	cleanPath := filepath.Clean(path)
	cleanHome := filepath.Clean(home)
	if cleanPath == cleanHome {
		return "~"
	}
	if strings.HasPrefix(cleanPath, cleanHome+string(os.PathSeparator)) {
		return "~" + strings.TrimPrefix(cleanPath, cleanHome)
	}
	return cleanPath
}

func stripANSI(value string) string {
	var b strings.Builder
	inEscape := false
	for _, r := range value {
		switch {
		case r == '\x1b':
			inEscape = true
		case inEscape && r == 'm':
			inEscape = false
		case !inEscape:
			b.WriteRune(r)
		}
	}
	return b.String()
}

func applyProviderPreset(cfg *config.Config, preset providerPreset) {
	cfg.Agents.Defaults.Provider = preset.ID
	cfg.Agents.Defaults.ModelName = preset.Alias
	cfg.Agents.Defaults.Model = ""
}

func providerPresetByID(id string) (providerPreset, bool) {
	for _, preset := range providerPresets {
		if preset.ID == id {
			return preset, true
		}
	}
	return providerPreset{}, false
}

func requiresProviderKey(providerID string) bool {
	switch providerID {
	case "openai", "anthropic", "openrouter":
		return true
	default:
		return false
	}
}

func setProviderAPIKey(cfg *config.Config, providerID string, key string) {
	key = strings.TrimSpace(key)
	if key == "" {
		return
	}

	switch providerID {
	case "openai":
		cfg.Providers.OpenAI.APIKey = key
	case "anthropic":
		cfg.Providers.Anthropic.APIKey = key
	case "openrouter":
		cfg.Providers.OpenRouter.APIKey = key
	}

	alias := cfg.Agents.Defaults.GetModelName()
	for index := range cfg.ModelList {
		if cfg.ModelList[index].ModelName == alias {
			cfg.ModelList[index].APIKey = key
		}
	}
}

func providerHasAPIKey(cfg *config.Config) bool {
	switch strings.ToLower(strings.TrimSpace(cfg.Agents.Defaults.Provider)) {
	case "openai", "gpt":
		return strings.TrimSpace(cfg.Providers.OpenAI.APIKey) != ""
	case "anthropic", "claude":
		return strings.TrimSpace(cfg.Providers.Anthropic.APIKey) != ""
	case "openrouter":
		return strings.TrimSpace(cfg.Providers.OpenRouter.APIKey) != ""
	default:
		return true
	}
}

func launcherAccessLabel(cfg launcherconfig.Config) string {
	if cfg.Public {
		return fmt.Sprintf("local network on port %d", cfg.Port)
	}
	return fmt.Sprintf("local only on port %d", cfg.Port)
}

func credentialHint(cfg *config.Config) string {
	if providerHasAPIKey(cfg) {
		return ""
	}
	switch strings.ToLower(strings.TrimSpace(cfg.Agents.Defaults.Provider)) {
	case "openai", "gpt":
		return "Add your OpenAI API key to the generated config before the first chat."
	case "anthropic", "claude":
		return "Add your Anthropic API key to the generated config before the first chat."
	case "openrouter":
		return "Add your OpenRouter API key to the generated config before the first chat."
	default:
		return ""
	}
}

func detectOpenClawSource() string {
	var candidates []string
	if envHome := strings.TrimSpace(os.Getenv("OPENCLAW_HOME")); envHome != "" {
		candidates = append(candidates, envHome)
	}
	if home, err := os.UserHomeDir(); err == nil {
		candidates = append(candidates, filepath.Join(home, ".openclaw"))
	}

	seen := make(map[string]struct{}, len(candidates))
	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}
		candidate = expandHome(candidate)
		if _, ok := seen[candidate]; ok {
			continue
		}
		seen[candidate] = struct{}{}
		if hasOpenClawConfig(candidate) {
			return candidate
		}
	}
	return ""
}

func hasOpenClawConfig(sourceHome string) bool {
	for _, name := range []string{"openclaw.json", "config.json"} {
		if _, err := os.Stat(filepath.Join(sourceHome, name)); err == nil {
			return true
		}
	}
	return false
}

func expandHome(path string) string {
	if path == "" || path[0] != '~' {
		return path
	}
	home, _ := os.UserHomeDir()
	if len(path) > 1 && path[1] == '/' {
		return home + path[1:]
	}
	return home
}

func normalizeProvider(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "":
		return ""
	case "ollama":
		return "ollama"
	case "openai", "gpt":
		return "openai"
	case "anthropic", "claude":
		return "anthropic"
	case "openrouter":
		return "openrouter"
	default:
		return ""
	}
}

func normalizeChannel(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "":
		return ""
	case "telegram":
		return "telegram"
	case "discord":
		return "discord"
	case "matrix":
		return "matrix"
	case "slack":
		return "slack"
	default:
		return ""
	}
}

func labelForChoice(value string, fallback string, labels map[string]string) string {
	if label, ok := labels[value]; ok {
		return label
	}
	return fallback
}
