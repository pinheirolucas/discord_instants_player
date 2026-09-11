package bot

import (
	"log/slog"

	"github.com/disgoorg/disgo/discord"

	"github.com/pinheirolucas/discord_instants_player/pkg/command"
)

func (b *Bot) help(ctx *command.DiscordContext) {
	e := ctx.Event
	d := ctx.Dispatcher

	if _, err := e.Client().Rest.CreateMessage(e.ChannelID, discord.MessageCreate{Content: d.GetHelp()}); err != nil {
		slog.Error("failed to send help message", "err", err)
	}
}
