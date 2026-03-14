// NookClaw - Ultra-lightweight personal AI agent
// Inspired by and based on nanobot: https://github.com/HKUDS/nanobot
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/samnoadd/NookClaw/cmd/nookclaw/internal"
	"github.com/samnoadd/NookClaw/cmd/nookclaw/internal/agent"
	"github.com/samnoadd/NookClaw/cmd/nookclaw/internal/auth"
	"github.com/samnoadd/NookClaw/cmd/nookclaw/internal/cron"
	"github.com/samnoadd/NookClaw/cmd/nookclaw/internal/gateway"
	"github.com/samnoadd/NookClaw/cmd/nookclaw/internal/migrate"
	"github.com/samnoadd/NookClaw/cmd/nookclaw/internal/model"
	"github.com/samnoadd/NookClaw/cmd/nookclaw/internal/onboard"
	"github.com/samnoadd/NookClaw/cmd/nookclaw/internal/skills"
	"github.com/samnoadd/NookClaw/cmd/nookclaw/internal/status"
	"github.com/samnoadd/NookClaw/cmd/nookclaw/internal/version"
	"github.com/samnoadd/NookClaw/pkg/config"
)

func NewNookClawCommand() *cobra.Command {
	short := fmt.Sprintf("%s NookClaw - Personal AI Assistant v%s\n\n", internal.Logo, config.GetVersion())

	cmd := &cobra.Command{
		Use:     "nookclaw",
		Short:   short,
		Example: "nookclaw version",
	}

	cmd.AddCommand(
		onboard.NewOnboardCommand(),
		agent.NewAgentCommand(),
		auth.NewAuthCommand(),
		gateway.NewGatewayCommand(),
		status.NewStatusCommand(),
		cron.NewCronCommand(),
		migrate.NewMigrateCommand(),
		skills.NewSkillsCommand(),
		model.NewModelCommand(),
		version.NewVersionCommand(),
	)

	return cmd
}

const (
	colorBlue = "\033[1;38;2;62;93;185m"
	colorRed  = "\033[1;38;2;213;70;70m"
	banner    = "\r\n" + colorBlue + "Nook" + colorRed + "Claw" + "\033[0m\r\n\r\n"
)

func main() {
	fmt.Printf("%s", banner)
	cmd := NewNookClawCommand()
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
