package onboard

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
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
	GatewayAutostart  bool
	Provider          string
	APIKey            string
	Channel           string
	ChannelSecret     string
	ChannelAppToken   string
	ChannelUserID     string
	ChannelHomeserver string
	ChannelAllowFrom  string
}

type onboardingState struct {
	SetupMode         string
	DetectedOpenClaw  string
	LauncherConfig    launcherconfig.Config
	CredentialHint    string
	ConfiguredChannel string
	GatewayAutostart  bool
	GatewayService    gatewayServiceSetup
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
	Guidance    []string
}

type modelChoice struct {
	Key         string
	Label       string
	Description string
	Alias       string
	Provider    string
	Target      string
	KeyLabel    string
	NeedsAPIKey bool
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
		Description: "Bot token via @BotFather.",
		Guidance: []string{
			"Open Telegram and message @BotFather.",
			"Run /newbot, follow the prompts, then copy the bot token it returns.",
			"Paste the token below, then add optional allow_from entries such as 123456789, telegram:123456789, or @username.",
			"Leave allow_from blank only if you want the bot reachable by any Telegram account that can find it.",
		},
	},
	{
		ID:          "discord",
		Label:       "Discord",
		Description: "Bot token from the Discord developer portal.",
		Guidance: []string{
			"Create an application in the Discord Developer Portal, then add a Bot user.",
			"Copy the bot token and enable the Message Content intent if you expect normal chat input.",
			"Paste the token below. Invite the bot to your server after onboarding.",
		},
	},
	{
		ID:          "matrix",
		Label:       "Matrix",
		Description: "Homeserver, user ID, and access token.",
		Guidance: []string{
			"Choose the homeserver your bot account lives on, such as https://matrix.org.",
			"Create or log in with the bot account, then generate an access token for it.",
			"Paste the homeserver, user ID, and access token below.",
		},
	},
	{
		ID:          "slack",
		Label:       "Slack",
		Description: "Bot token and Socket Mode app token.",
		Guidance: []string{
			"Create a Slack app, enable Socket Mode, and install it into your workspace.",
			"Collect the Bot User OAuth token (starts with xoxb-) and the App-Level token (starts with xapp-).",
			"Paste both values below.",
		},
	},
	{
		ID:          "line",
		Label:       "LINE",
		Description: "Channel secret and Messaging API access token.",
		Guidance: []string{
			"Create a Messaging API channel in the LINE Developers Console.",
			"Copy the channel secret and generate a long-lived channel access token.",
			"Paste both values below.",
		},
	},
	{
		ID:          "irc",
		Label:       "IRC",
		Description: "Server, nickname, and channels to join.",
		Guidance: []string{
			"Pick the IRC network and server address, for example irc.libera.chat:6697.",
			"Choose the bot nickname and the channels it should join, separated by commas.",
			"If the network supports TLS, enable it during the next step.",
		},
	},
	{
		ID:          "onebot",
		Label:       "OneBot",
		Description: "WebSocket URL and optional access token.",
		Guidance: []string{
			"Point NookClaw at the WebSocket endpoint exposed by your OneBot bridge.",
			"Use the access token if your bridge requires one.",
			"Paste the WebSocket URL and token below.",
		},
	},
	{
		ID:          "qq",
		Label:       "QQ",
		Description: "App ID and app secret.",
		Guidance: []string{
			"Create a QQ bot application and copy the App ID and App Secret.",
			"Paste both values below.",
			"After onboarding, verify allowlists and message limits in the config if needed.",
		},
	},
	{
		ID:          "dingtalk",
		Label:       "DingTalk",
		Description: "Client ID and client secret.",
		Guidance: []string{
			"Create a DingTalk application and collect the Client ID and Client Secret.",
			"Paste both values below.",
			"Review callback and allowlist settings after onboarding if you plan to expose it widely.",
		},
	},
	{
		ID:          "feishu",
		Label:       "Feishu",
		Description: "App ID, app secret, verification token, and encrypt key.",
		Guidance: []string{
			"Create a Feishu app and collect the App ID and App Secret.",
			"Enable the event callback and copy the verification token and encrypt key.",
			"Paste all four values below.",
		},
	},
}

var discoverOllamaModelsFn = discoverOllamaModels

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
		return fmt.Errorf("unsupported channel %q: use telegram, discord, matrix, slack, line, irc, onebot, qq, dingtalk, or feishu", opts.Channel)
	}
	if opts.ChannelSecret != "" && channel == "" {
		return fmt.Errorf("--channel-secret requires --channel")
	}
	if opts.ChannelAppToken != "" && channel != "slack" && channel != "line" {
		return fmt.Errorf("--channel-app-token requires --channel slack or --channel line")
	}
	if opts.ChannelUserID != "" && channel != "matrix" {
		return fmt.Errorf("--channel-user-id requires --channel matrix")
	}
	if opts.ChannelHomeserver != "" && channel != "matrix" {
		return fmt.Errorf("--channel-homeserver requires --channel matrix")
	}
	if opts.ChannelAllowFrom != "" && channel != "telegram" {
		return fmt.Errorf("--channel-allow-from currently requires --channel telegram")
	}
	if opts.GatewayAutostart && !gatewayUserServiceSupported() {
		return fmt.Errorf("--gateway-autostart currently requires Linux with systemd user services")
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
		case "line", "irc", "onebot", "qq", "dingtalk", "feishu":
			return fmt.Errorf("--channel %s is only supported interactively for now", channel)
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
		o.ChannelAllowFrom != "" ||
		o.GatewayAutostart ||
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
				{Text: "Choose how much you want to configure in this pass.", Role: "secondary"},
				{Text: "Both paths let you pick a model. Advanced also exposes automation and launcher policy now.", Role: "secondary"},
			},
			"Choose a setup profile:",
			[]wizardOption{
				{
					Key:         "quick",
					Label:       quickStartMode,
					Description: "Pick a model, keep the rest minimal, and get to a first chat quickly.",
				},
				{
					Key:         "advanced",
					Label:       advancedMode,
					Description: "Pick a model, credentials, channels, background features, and launcher access in one pass.",
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

	if err := w.configureProviderFromFlags(cfg); err != nil {
		return onboardingState{}, err
	}

	w.printSection(
		"Model selection",
		"Choose the model NookClaw should use by default. Quick Start still needs a real model choice.",
	)
	if err := w.configureModel(cfg); err != nil {
		return onboardingState{}, err
	}
	fmt.Fprintln(w.out)

	if state.SetupMode == advancedMode {
		w.printSection(
			"Background features",
			"Enable only the automation and reachability features you plan to use on day one.",
		)
		cfg.Tools.Web.Enabled = w.promptYesNo("Enable web tools for search and page fetch?", cfg.Tools.Web.Enabled)
		cfg.Tools.Cron.Enabled = w.promptYesNo("Enable the scheduler for recurring tasks?", cfg.Tools.Cron.Enabled)
		cfg.Heartbeat.Enabled = w.promptYesNo("Enable heartbeat background checks?", cfg.Heartbeat.Enabled)
		cfg.Tools.Exec.AllowRemote = w.promptYesNo("Allow remote exec requests?", cfg.Tools.Exec.AllowRemote)
		fmt.Fprintln(w.out)
	}

	w.printSection(
		"Channel access",
		"Pick one inbound channel to wire up now, or skip it and finish the rest of onboarding first.",
	)
	channelLabel, err := w.configureChannel(cfg)
	if err != nil {
		return onboardingState{}, err
	}
	state.ConfiguredChannel = channelLabel
	if channelLabel != "none" {
		w.printSection(
			"Gateway runtime",
			"Inbound channels only receive messages while `nookclaw gateway` is running.",
		)
		state.GatewayAutostart = w.configureGatewayRuntime()
		fmt.Fprintln(w.out)
	}

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
	state.GatewayAutostart = w.opts.GatewayAutostart

	return nil
}

func (w *setupWizard) configureModel(cfg *config.Config) error {
	choices, defaultKey := buildModelChoices(cfg)
	if len(choices) == 0 {
		return fmt.Errorf("no selectable models found in config or local Ollama")
	}

	options := make([]wizardOption, 0, len(choices))
	for _, choice := range choices {
		options = append(options, wizardOption{
			Key:         choice.Key,
			Label:       choice.Label,
			Description: choice.Description,
		})
	}

	selected := w.promptChoiceWithContext(
		[]selectorLine{
			{Text: "Model selection", Role: "primary"},
			{Text: "Installed Ollama models are listed first when available. Configured cloud aliases follow.", Role: "secondary"},
			{Text: "Pick the model NookClaw should use by default after onboarding.", Role: "secondary"},
		},
		"Select a model:",
		options,
		defaultKey,
	)

	choice, ok := modelChoiceByKey(choices, selected)
	if !ok {
		return fmt.Errorf("unknown model selection %q", selected)
	}

	if err := applyModelChoice(cfg, choice); err != nil {
		return err
	}

	if choice.NeedsAPIKey {
		key := strings.TrimSpace(w.opts.APIKey)
		if providerID := normalizeProvider(w.opts.Provider); providerID != "" && providerID != choice.Provider {
			key = ""
		}
		if key == "" {
			if providerHasAPIKey(cfg) {
				return nil
			}
			w.showGuidance(choice.Label+" setup", []string{
				fmt.Sprintf("%s uses %s.", choice.Label, choice.Target),
				fmt.Sprintf("Paste your %s now, or leave it for later and edit config.json before the first real chat.", choice.KeyLabel),
			})
			if w.promptYesNo(fmt.Sprintf("Add your %s now?", choice.KeyLabel), false) {
				key = w.promptSecret(choice.KeyLabel)
			}
		}
		if key != "" {
			setProviderAPIKey(cfg, choice.Provider, key)
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
		options := []wizardOption{
			{
				Key:         "skip",
				Label:       "Skip for now",
				Description: "Finish onboarding first and add a channel later from the config or launcher.",
			},
		}
		for _, preset := range channelPresets {
			options = append(options, wizardOption{
				Key:         preset.ID,
				Label:       preset.Label,
				Description: preset.Description,
			})
		}
		choice := w.promptChoiceWithContext(
			[]selectorLine{
				{Text: "Channel access", Role: "primary"},
				{Text: "Choose one channel to wire up now, or skip it and finish setup first.", Role: "secondary"},
				{Text: "More complex integrations can still be tuned in config.json after onboarding.", Role: "secondary"},
			},
			"Select a channel:",
			options,
			"skip",
		)
		if choice == "skip" {
			return "none", nil
		}
		channelID = choice
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
	if preset, ok := channelPresetByID(channelID); ok {
		w.showGuidance(preset.Label+" setup", preset.Guidance)
	}

	switch channelID {
	case "telegram":
		token := strings.TrimSpace(w.opts.ChannelSecret)
		if token == "" {
			token = w.promptSecret("Telegram bot token")
		}
		if strings.TrimSpace(token) == "" {
			return "", fmt.Errorf("telegram channel requires a bot token")
		}
		allowFromInput := strings.TrimSpace(w.opts.ChannelAllowFrom)
		if allowFromInput == "" && w.interactive {
			allowFromInput = w.promptText(
				"Telegram allow_from entries (comma separated, optional)",
				strings.Join(cfg.Channels.Telegram.AllowFrom, ","),
			)
		}
		cfg.Channels.Telegram.Enabled = true
		cfg.Channels.Telegram.Token = token
		cfg.Channels.Telegram.AllowFrom = flexibleStringSliceFromCSV(allowFromInput)
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
	case "line":
		secret := strings.TrimSpace(w.opts.ChannelSecret)
		if secret == "" {
			secret = w.promptSecret("LINE channel secret")
		}
		accessToken := strings.TrimSpace(w.opts.ChannelAppToken)
		if accessToken == "" {
			accessToken = w.promptSecret("LINE channel access token")
		}
		if strings.TrimSpace(secret) == "" || strings.TrimSpace(accessToken) == "" {
			return "", fmt.Errorf("line channel requires a channel secret and access token")
		}
		cfg.Channels.LINE.Enabled = true
		cfg.Channels.LINE.ChannelSecret = secret
		cfg.Channels.LINE.ChannelAccessToken = accessToken
		return "LINE", nil
	case "irc":
		server := w.promptText("IRC server", cfg.Channels.IRC.Server)
		tlsEnabled := w.promptYesNo("Use TLS for IRC?", cfg.Channels.IRC.TLS)
		nick := w.promptText("IRC nickname", cfg.Channels.IRC.Nick)
		channels := parseCommaSeparatedList(w.promptText("IRC channels (comma separated)", strings.Join(cfg.Channels.IRC.Channels, ",")))
		if strings.TrimSpace(server) == "" || strings.TrimSpace(nick) == "" || len(channels) == 0 {
			return "", fmt.Errorf("irc channel requires a server, nickname, and at least one channel")
		}
		cfg.Channels.IRC.Enabled = true
		cfg.Channels.IRC.Server = server
		cfg.Channels.IRC.TLS = tlsEnabled
		cfg.Channels.IRC.Nick = nick
		cfg.Channels.IRC.Channels = config.FlexibleStringSlice(channels)
		return "IRC", nil
	case "onebot":
		wsURL := w.promptText("OneBot WebSocket URL", cfg.Channels.OneBot.WSUrl)
		token := strings.TrimSpace(w.opts.ChannelSecret)
		if token == "" {
			token = w.promptText("OneBot access token (optional)", cfg.Channels.OneBot.AccessToken)
		}
		if strings.TrimSpace(wsURL) == "" {
			return "", fmt.Errorf("onebot channel requires a WebSocket URL")
		}
		cfg.Channels.OneBot.Enabled = true
		cfg.Channels.OneBot.WSUrl = wsURL
		cfg.Channels.OneBot.AccessToken = token
		return "OneBot", nil
	case "qq":
		appID := w.promptText("QQ app ID", cfg.Channels.QQ.AppID)
		appSecret := strings.TrimSpace(w.opts.ChannelSecret)
		if appSecret == "" {
			appSecret = w.promptSecret("QQ app secret")
		}
		if strings.TrimSpace(appID) == "" || strings.TrimSpace(appSecret) == "" {
			return "", fmt.Errorf("qq channel requires an app ID and app secret")
		}
		cfg.Channels.QQ.Enabled = true
		cfg.Channels.QQ.AppID = appID
		cfg.Channels.QQ.AppSecret = appSecret
		return "QQ", nil
	case "dingtalk":
		clientID := w.promptText("DingTalk client ID", cfg.Channels.DingTalk.ClientID)
		clientSecret := strings.TrimSpace(w.opts.ChannelSecret)
		if clientSecret == "" {
			clientSecret = w.promptSecret("DingTalk client secret")
		}
		if strings.TrimSpace(clientID) == "" || strings.TrimSpace(clientSecret) == "" {
			return "", fmt.Errorf("dingtalk channel requires a client ID and client secret")
		}
		cfg.Channels.DingTalk.Enabled = true
		cfg.Channels.DingTalk.ClientID = clientID
		cfg.Channels.DingTalk.ClientSecret = clientSecret
		return "DingTalk", nil
	case "feishu":
		appID := w.promptText("Feishu app ID", cfg.Channels.Feishu.AppID)
		appSecret := strings.TrimSpace(w.opts.ChannelSecret)
		if appSecret == "" {
			appSecret = w.promptSecret("Feishu app secret")
		}
		verificationToken := w.promptText("Feishu verification token", cfg.Channels.Feishu.VerificationToken)
		encryptKey := w.promptText("Feishu encrypt key", cfg.Channels.Feishu.EncryptKey)
		if strings.TrimSpace(appID) == "" || strings.TrimSpace(appSecret) == "" || strings.TrimSpace(verificationToken) == "" || strings.TrimSpace(encryptKey) == "" {
			return "", fmt.Errorf("feishu channel requires app ID, app secret, verification token, and encrypt key")
		}
		cfg.Channels.Feishu.Enabled = true
		cfg.Channels.Feishu.AppID = appID
		cfg.Channels.Feishu.AppSecret = appSecret
		cfg.Channels.Feishu.VerificationToken = verificationToken
		cfg.Channels.Feishu.EncryptKey = encryptKey
		return "Feishu", nil
	default:
		return "", fmt.Errorf("unsupported channel %q", channelID)
	}
}

func (w *setupWizard) configureGatewayRuntime() bool {
	if !gatewayUserServiceSupported() {
		fmt.Fprint(w.out, renderCallout("Gateway runtime", []string{
			"Inbound channels only work while `nookclaw gateway` is running.",
			"Automatic startup during onboarding is currently available only on Linux systems with systemd user services.",
		}))
		fmt.Fprintln(w.out)
		return false
	}

	return w.promptYesNo("Install a systemd user service so the gateway starts automatically on login?", false)
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
	promptLabel := secretPromptLabel(label, true)
	if file, ok := w.in.(*os.File); ok && term.IsTerminal(int(file.Fd())) {
		fmt.Fprintf(w.out, "%s: ", promptLabel)
		value, err := term.ReadPassword(int(file.Fd()))
		fmt.Fprintln(w.out)
		if err == nil {
			return strings.TrimSpace(string(value))
		}
	}
	fmt.Fprintf(w.out, "%s: ", promptLabel)
	value := strings.TrimSpace(w.readLine())
	fmt.Fprintln(w.out)
	return value
}

func secretPromptLabel(label string, hidden bool) string {
	if !hidden {
		return stylePrimary(label)
	}
	return fmt.Sprintf("%s %s", stylePrimary(label), styleSecondary("[input hidden]"))
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
		fmt.Fprint(w.out, rawTTYScreen(renderSelectorScreen(context, actionLabel, options, selected)))

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
	fmt.Fprint(out, "\033[H\033[2J\r")
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

func rawTTYScreen(value string) string {
	return strings.ReplaceAll(value, "\n", "\r\n")
}

func (w *setupWizard) showGuidance(title string, lines []string) {
	if !w.interactive || len(lines) == 0 {
		return
	}
	if w.selectorUIEnabled() {
		clearInteractiveScreen(w.out)
	}
	fmt.Fprint(w.out, renderGuideScreen(title, lines))
}

func buildModelChoices(cfg *config.Config) ([]modelChoice, string) {
	choices := make([]modelChoice, 0, len(cfg.ModelList)+4)
	defaultKey := ""
	currentAlias := strings.TrimSpace(cfg.Agents.Defaults.GetModelName())

	localModels := discoverOllamaModelsFn()
	for _, name := range localModels {
		target := "ollama/" + name
		choice := modelChoice{
			Key:         "local:" + name,
			Label:       name,
			Description: "Detected locally via Ollama on this machine.",
			Alias:       "private-local",
			Provider:    "ollama",
			Target:      target,
		}
		choices = append(choices, choice)
		if currentAlias == "private-local" && defaultKey == "" {
			if mc, err := cfg.GetModelConfig(currentAlias); err == nil && mc != nil && mc.Model == target {
				defaultKey = choice.Key
			}
		}
	}

	for _, model := range cfg.ModelList {
		if len(localModels) > 0 && model.ModelName == "private-local" {
			continue
		}
		provider := providerFromModelTarget(model.Model)
		if provider == "" {
			continue
		}

		desc := model.Model
		if provider == "ollama" {
			desc += " • configured local alias"
		} else if hasModelAPIKey(cfg, model) {
			desc += " • API key ready"
		} else {
			desc += " • API key needed"
		}

		choice := modelChoice{
			Key:         "alias:" + model.ModelName,
			Label:       model.ModelName,
			Description: desc,
			Alias:       model.ModelName,
			Provider:    provider,
			Target:      model.Model,
			KeyLabel:    providerKeyLabel(provider),
			NeedsAPIKey: provider != "ollama" && !hasModelAPIKey(cfg, model),
		}
		choices = append(choices, choice)

		if defaultKey == "" && currentAlias == model.ModelName {
			defaultKey = choice.Key
		}
	}

	if defaultKey == "" && len(choices) > 0 {
		defaultKey = choices[0].Key
	}
	return choices, defaultKey
}

func modelChoiceByKey(choices []modelChoice, key string) (modelChoice, bool) {
	for _, choice := range choices {
		if choice.Key == key {
			return choice, true
		}
	}
	return modelChoice{}, false
}

func applyModelChoice(cfg *config.Config, choice modelChoice) error {
	switch {
	case strings.HasPrefix(choice.Key, "local:"):
		ensurePrivateLocalModel(cfg, choice.Target)
		cfg.Agents.Defaults.Provider = "ollama"
		cfg.Agents.Defaults.ModelName = "private-local"
		cfg.Agents.Defaults.Model = ""
		return nil
	case strings.HasPrefix(choice.Key, "alias:"):
		cfg.Agents.Defaults.Provider = choice.Provider
		cfg.Agents.Defaults.ModelName = choice.Alias
		cfg.Agents.Defaults.Model = ""
		return nil
	default:
		return fmt.Errorf("unsupported model choice %q", choice.Key)
	}
}

func ensurePrivateLocalModel(cfg *config.Config, target string) {
	for index := range cfg.ModelList {
		if cfg.ModelList[index].ModelName == "private-local" {
			cfg.ModelList[index].Model = target
			cfg.ModelList[index].APIBase = "http://localhost:11434/v1"
			cfg.ModelList[index].APIKey = "ollama"
			return
		}
	}

	cfg.ModelList = append([]config.ModelConfig{{
		ModelName: "private-local",
		Model:     target,
		APIBase:   "http://localhost:11434/v1",
		APIKey:    "ollama",
	}}, cfg.ModelList...)
}

func providerFromModelTarget(target string) string {
	parts := strings.SplitN(strings.TrimSpace(target), "/", 2)
	if len(parts) == 0 {
		return ""
	}
	switch parts[0] {
	case "ollama", "openai", "anthropic", "openrouter", "deepseek", "gemini", "qwen", "groq", "zhipu", "moonshot", "volcengine", "nvidia":
		return parts[0]
	default:
		return ""
	}
}

func hasModelAPIKey(cfg *config.Config, model config.ModelConfig) bool {
	if strings.TrimSpace(model.APIKey) != "" {
		return true
	}

	switch providerFromModelTarget(model.Model) {
	case "ollama":
		return true
	case "openai":
		return strings.TrimSpace(cfg.Providers.OpenAI.APIKey) != ""
	case "anthropic":
		return strings.TrimSpace(cfg.Providers.Anthropic.APIKey) != ""
	case "openrouter":
		return strings.TrimSpace(cfg.Providers.OpenRouter.APIKey) != ""
	case "deepseek":
		return strings.TrimSpace(cfg.Providers.DeepSeek.APIKey) != ""
	case "gemini":
		return strings.TrimSpace(cfg.Providers.Gemini.APIKey) != ""
	case "qwen":
		return strings.TrimSpace(cfg.Providers.Qwen.APIKey) != ""
	case "groq":
		return strings.TrimSpace(cfg.Providers.Groq.APIKey) != ""
	case "zhipu":
		return strings.TrimSpace(cfg.Providers.Zhipu.APIKey) != ""
	case "moonshot":
		return strings.TrimSpace(cfg.Providers.Moonshot.APIKey) != ""
	case "volcengine":
		return strings.TrimSpace(cfg.Providers.VolcEngine.APIKey) != ""
	case "nvidia":
		return strings.TrimSpace(cfg.Providers.Nvidia.APIKey) != ""
	default:
		return false
	}
}

func discoverOllamaModels() []string {
	if _, err := exec.LookPath("ollama"); err != nil {
		return nil
	}

	out, err := exec.Command("ollama", "list").Output()
	if err != nil {
		return nil
	}

	lines := strings.Split(string(out), "\n")
	models := make([]string, 0, len(lines))
	seen := map[string]struct{}{}
	for index, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if index == 0 && strings.HasPrefix(strings.ToUpper(line), "NAME ") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		name := fields[0]
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		models = append(models, name)
	}

	sort.Strings(models)
	return models
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
	return strings.TrimSpace(providerID) != "" && providerID != "ollama"
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
	case "deepseek":
		cfg.Providers.DeepSeek.APIKey = key
	case "gemini":
		cfg.Providers.Gemini.APIKey = key
	case "qwen":
		cfg.Providers.Qwen.APIKey = key
	case "groq":
		cfg.Providers.Groq.APIKey = key
	case "zhipu":
		cfg.Providers.Zhipu.APIKey = key
	case "moonshot":
		cfg.Providers.Moonshot.APIKey = key
	case "volcengine":
		cfg.Providers.VolcEngine.APIKey = key
	case "nvidia":
		cfg.Providers.Nvidia.APIKey = key
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
	case "deepseek":
		return strings.TrimSpace(cfg.Providers.DeepSeek.APIKey) != ""
	case "gemini":
		return strings.TrimSpace(cfg.Providers.Gemini.APIKey) != ""
	case "qwen":
		return strings.TrimSpace(cfg.Providers.Qwen.APIKey) != ""
	case "groq":
		return strings.TrimSpace(cfg.Providers.Groq.APIKey) != ""
	case "zhipu":
		return strings.TrimSpace(cfg.Providers.Zhipu.APIKey) != ""
	case "moonshot":
		return strings.TrimSpace(cfg.Providers.Moonshot.APIKey) != ""
	case "volcengine":
		return strings.TrimSpace(cfg.Providers.VolcEngine.APIKey) != ""
	case "nvidia":
		return strings.TrimSpace(cfg.Providers.Nvidia.APIKey) != ""
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
	case "deepseek":
		return "Add your DeepSeek API key to the generated config before the first chat."
	case "gemini":
		return "Add your Gemini API key to the generated config before the first chat."
	case "qwen":
		return "Add your Qwen API key to the generated config before the first chat."
	case "groq":
		return "Add your Groq API key to the generated config before the first chat."
	case "zhipu":
		return "Add your Zhipu API key to the generated config before the first chat."
	case "moonshot":
		return "Add your Moonshot API key to the generated config before the first chat."
	case "volcengine":
		return "Add your VolcEngine API key to the generated config before the first chat."
	case "nvidia":
		return "Add your Nvidia API key to the generated config before the first chat."
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
	case "line":
		return "line"
	case "irc":
		return "irc"
	case "onebot":
		return "onebot"
	case "qq":
		return "qq"
	case "dingtalk":
		return "dingtalk"
	case "feishu":
		return "feishu"
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

func providerKeyLabel(providerID string) string {
	switch providerID {
	case "openai":
		return "OpenAI API key"
	case "anthropic":
		return "Anthropic API key"
	case "openrouter":
		return "OpenRouter API key"
	case "deepseek":
		return "DeepSeek API key"
	case "gemini":
		return "Gemini API key"
	case "qwen":
		return "Qwen API key"
	case "groq":
		return "Groq API key"
	case "zhipu":
		return "Zhipu API key"
	case "moonshot":
		return "Moonshot API key"
	case "volcengine":
		return "VolcEngine API key"
	case "nvidia":
		return "Nvidia API key"
	default:
		return "API key"
	}
}

func channelPresetByID(id string) (channelPreset, bool) {
	for _, preset := range channelPresets {
		if preset.ID == id {
			return preset, true
		}
	}
	return channelPreset{}, false
}

func parseCommaSeparatedList(value string) []string {
	value = strings.ReplaceAll(value, "，", ",")
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			result = append(result, part)
		}
	}
	return result
}

func flexibleStringSliceFromCSV(value string) config.FlexibleStringSlice {
	entries := parseCommaSeparatedList(value)
	if len(entries) == 0 {
		return nil
	}
	return config.FlexibleStringSlice(entries)
}
