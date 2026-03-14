package dingtalk

import (
	"github.com/samnoadd/NookClaw/pkg/bus"
	"github.com/samnoadd/NookClaw/pkg/channels"
	"github.com/samnoadd/NookClaw/pkg/config"
)

func init() {
	channels.RegisterFactory("dingtalk", func(cfg *config.Config, b *bus.MessageBus) (channels.Channel, error) {
		return NewDingTalkChannel(cfg.Channels.DingTalk, b)
	})
}
