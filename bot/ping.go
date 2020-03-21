package bot

import (
	"github.com/rs/zerolog/log"

	"github.com/pinheirolucas/discord_instants_player/command"
)

func (b *Bot) ping(ctx *command.DiscordContext) {
	m := ctx.Message
	s := ctx.Session

	if _, err := s.ChannelMessageSend(m.ChannelID, "Pong!"); err != nil {
		log.Error().Err(err).Msg("failed to send help message")
	}
}
