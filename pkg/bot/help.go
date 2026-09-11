package bot

import (
	"log/slog"

	"github.com/pinheirolucas/discord_instants_player/pkg/command"
)

func (b *Bot) help(ctx *command.DiscordContext) {
	d := ctx.Dispatcher
	m := ctx.Message
	s := ctx.Session

	if _, err := s.ChannelMessageSend(m.ChannelID, d.GetHelp()); err != nil {
		slog.Error("failed to send help message", "err", err)
	}
}
