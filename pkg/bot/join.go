package bot

import (
	"context"
	"log/slog"

	"github.com/pinheirolucas/discord_instants_player/pkg/command"
)

func (b *Bot) join(ctx *command.DiscordContext) {
	e := ctx.Event
	client := e.Client()

	guild, ok := client.Caches.Guild(*e.GuildID)
	if !ok {
		slog.Error("failed to fetch guild info", "GuildID", e.GuildID)
		return
	}

	voiceState, ok := client.Caches.VoiceState(guild.ID, e.Message.Author.ID)
	if !ok || voiceState.ChannelID == nil {
		slog.Info("voice channel not found", "AuthorUsername", e.Message.Author.Username)
		return
	}

	channel, ok := client.Caches.Channel(*voiceState.ChannelID)
	if !ok {
		slog.Error("failed to fetch voice channel info", "ChannelID", *voiceState.ChannelID)
		return
	}

	if b.vc == nil {
		conn := client.VoiceManager.CreateConn(guild.ID)
		if err := conn.Open(context.Background(), *voiceState.ChannelID, false, true); err != nil {
			slog.Error("failed to join voice channel",
				"GuildID", guild.ID,
				"GuildName", guild.Name,
				"ChannelID", *voiceState.ChannelID,
				"ChannelName", channel.Name(),
				"err", err,
			)
			return
		}
		b.vc = conn
	}
}
