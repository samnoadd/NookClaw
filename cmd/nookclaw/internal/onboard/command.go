package onboard

import (
	"embed"

	"github.com/spf13/cobra"
)

//go:generate cp -r ../../../../workspace .
//go:embed workspace
var embeddedFiles embed.FS

func NewOnboardCommand() *cobra.Command {
	opts := onboardOptions{}
	cmd := &cobra.Command{
		Use:     "onboard",
		Aliases: []string{"o"},
		Short:   "Initialize NookClaw configuration and workspace",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return onboard(opts)
		},
	}

	cmd.Flags().BoolVar(&opts.NonInteractive, "non-interactive", false, "Run onboarding without prompts")
	cmd.Flags().BoolVar(&opts.Advanced, "advanced", false, "Use the advanced setup flow")
	cmd.Flags().BoolVar(&opts.Force, "force", false, "Replace an existing setup without prompting")
	cmd.Flags().BoolVar(&opts.LauncherPublic, "launcher-public", false, "Expose the web launcher on the local network")
	cmd.Flags().BoolVar(&opts.GatewayAutostart, "gateway-autostart", false, "Install a systemd user service for nookclaw gateway when supported")
	cmd.Flags().StringVar(&opts.Provider, "provider", "", "Default model backend: ollama, openai, anthropic, or openrouter")
	cmd.Flags().StringVar(&opts.APIKey, "api-key", "", "API key for the selected provider")
	cmd.Flags().StringVar(&opts.Channel, "channel", "", "Select a chat channel: telegram, discord, matrix, slack, line, irc, onebot, qq, dingtalk, or feishu")
	cmd.Flags().StringVar(&opts.ChannelSecret, "channel-secret", "", "Primary token or secret for the selected channel")
	cmd.Flags().StringVar(&opts.ChannelAppToken, "channel-app-token", "", "Secondary token for Slack or LINE when required")
	cmd.Flags().StringVar(&opts.ChannelUserID, "channel-user-id", "", "User ID for Matrix channel setup")
	cmd.Flags().StringVar(&opts.ChannelHomeserver, "channel-homeserver", "", "Homeserver URL for Matrix channel setup")
	cmd.Flags().StringVar(&opts.ChannelAllowFrom, "channel-allow-from", "", "Comma-separated allow_from entries for Telegram onboarding")

	return cmd
}
