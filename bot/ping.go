package bot

import (
	"github.com/rs/zerolog/log"

	"github.com/pinheirolucas/discord_instants_player/dispatcher"
)

func (b *Bot) ping(ctx *dispatcher.Context) {
	m := ctx.Message
	s := ctx.Session

	if _, err := s.ChannelMessageSend(m.ChannelID, "Pong!"); err != nil {
		log.Error().Err(err).Msg("failed to send help message")
	}
}
