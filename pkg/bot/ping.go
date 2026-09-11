package bot

import (
	"log/slog"

	"github.com/pinheirolucas/discord_instants_player/pkg/command"
)

func (b *Bot) ping(ctx *command.DiscordContext) {
	m := ctx.Message
	s := ctx.Session

	if _, err := s.ChannelMessageSend(m.ChannelID, "Pong!"); err != nil {
		slog.Error("failed to send help message", "err", err)
	}
}
