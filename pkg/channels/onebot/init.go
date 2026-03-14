package onebot

import (
	"github.com/samnoadd/NookClaw/pkg/bus"
	"github.com/samnoadd/NookClaw/pkg/channels"
	"github.com/samnoadd/NookClaw/pkg/config"
)

func init() {
	channels.RegisterFactory("onebot", func(cfg *config.Config, b *bus.MessageBus) (channels.Channel, error) {
		return NewOneBotChannel(cfg.Channels.OneBot, b)
	})
}
