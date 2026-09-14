package bot

import (
	"log/slog"

	"github.com/disgoorg/disgo/discord"

	"github.com/pinheirolucas/peace-breaker-bot/pkg/command"
)

func (b *Bot) help(ctx *command.DiscordContext) {
	e := ctx.Event
	d := ctx.Dispatcher

	help := d.GetHelp(b.localeFor(e))
	if _, err := e.Client().Rest.CreateMessage(e.ChannelID, discord.MessageCreate{Content: help}); err != nil {
		slog.Error("failed to send help message", "err", err)
	}
}
