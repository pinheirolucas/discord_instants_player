package bot

import (
	"github.com/bwmarrin/discordgo"
	"github.com/rs/zerolog/log"

	"github.com/pinheirolucas/discord_instants_player/pkg/command"
)

func (b *Bot) join(ctx *command.DiscordContext) {
	s := ctx.Session
	m := ctx.Message

	guild, err := s.State.Guild(m.GuildID)
	if err != nil {
		log.Error().
			Str("GuildID", m.GuildID).
			Err(err).
			Msg("failed to fetch guild info")
		return
	}

	var currentVoiceChannel *discordgo.Channel
	for _, vs := range guild.VoiceStates {
		if vs.UserID != m.Author.ID {
			continue
		}

		channel, err := s.State.Channel(vs.ChannelID)
		if err != nil {
			log.Error().
				Str("ChannelID", vs.ChannelID).
				Err(err).
				Msg("failed to fetch voice channel info")
			return
		}

		currentVoiceChannel = channel
	}

	if currentVoiceChannel == nil {
		log.Info().
			Str("AuthorUsername", m.Author.Username).
			Msg("voice channel not found")
		return
	}

	if b.vc == nil {
		connection, err := s.ChannelVoiceJoin(guild.ID, currentVoiceChannel.ID, false, true)
		if err != nil {
			log.Error().
				Str("GuildID", guild.ID).
				Str("GuildName", guild.Name).
				Str("ChannelID", currentVoiceChannel.ID).
				Str("ChannelName", currentVoiceChannel.Name).
				Err(err).
				Msg("failed to join voice channel")
			return
		}
		b.vc = connection
	}
}
