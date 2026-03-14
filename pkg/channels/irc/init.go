package irc

import (
	"github.com/samnoadd/NookClaw/pkg/bus"
	"github.com/samnoadd/NookClaw/pkg/channels"
	"github.com/samnoadd/NookClaw/pkg/config"
)

func init() {
	channels.RegisterFactory("irc", func(cfg *config.Config, b *bus.MessageBus) (channels.Channel, error) {
		if !cfg.Channels.IRC.Enabled {
			return nil, nil
		}
		return NewIRCChannel(cfg.Channels.IRC, b)
	})
}
