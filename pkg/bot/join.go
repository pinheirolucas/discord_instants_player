package bot

import (
	"log/slog"

	"github.com/bwmarrin/discordgo"

	"github.com/pinheirolucas/discord_instants_player/pkg/command"
)

func (b *Bot) join(ctx *command.DiscordContext) {
	s := ctx.Session
	m := ctx.Message

	guild, err := s.State.Guild(m.GuildID)
	if err != nil {
		slog.Error("failed to fetch guild info",
			"GuildID", m.GuildID,
			"err", err,
		)
		return
	}

	var currentVoiceChannel *discordgo.Channel
	for _, vs := range guild.VoiceStates {
		if vs.UserID != m.Author.ID {
			continue
		}

		channel, err := s.State.Channel(vs.ChannelID)
		if err != nil {
			slog.Error("failed to fetch voice channel info",
				"ChannelID", vs.ChannelID,
				"err", err,
			)
			return
		}

		currentVoiceChannel = channel
	}

	if currentVoiceChannel == nil {
		slog.Info("voice channel not found",
			"AuthorUsername", m.Author.Username,
		)
		return
	}

	if b.vc == nil {
		connection, err := s.ChannelVoiceJoin(guild.ID, currentVoiceChannel.ID, false, true)
		if err != nil {
			slog.Error("failed to join voice channel",
				"GuildID", guild.ID,
				"GuildName", guild.Name,
				"ChannelID", currentVoiceChannel.ID,
				"ChannelName", currentVoiceChannel.Name,
				"err", err,
			)
			return
		}
		b.vc = connection
	}
}
