package slack

import (
	"github.com/samnoadd/NookClaw/pkg/bus"
	"github.com/samnoadd/NookClaw/pkg/channels"
	"github.com/samnoadd/NookClaw/pkg/config"
)

func init() {
	channels.RegisterFactory("slack", func(cfg *config.Config, b *bus.MessageBus) (channels.Channel, error) {
		return NewSlackChannel(cfg.Channels.Slack, b)
	})
}
