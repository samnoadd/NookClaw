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
	short := fmt.Sprintf("NookClaw - Personal AI Assistant v%s", config.GetVersion())

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

func main() {
	cmd := NewNookClawCommand()
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
