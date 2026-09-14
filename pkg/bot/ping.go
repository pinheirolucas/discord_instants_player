package bot

import (
	"log/slog"

	"github.com/disgoorg/disgo/discord"

	"github.com/pinheirolucas/peace-breaker-bot/pkg/command"
)

func (b *Bot) ping(ctx *command.DiscordContext) {
	e := ctx.Event

	if _, err := e.Client().Rest.CreateMessage(e.ChannelID, discord.MessageCreate{Content: "Pong!"}); err != nil {
		slog.Error("failed to send help message", "err", err)
	}
}
