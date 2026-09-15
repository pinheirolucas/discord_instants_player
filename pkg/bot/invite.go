package bot

import (
	"log/slog"
	"regexp"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
)

// inviteLinkPattern matches the invite link Discord's client sends as a
// plain-text DM when a user uses the "invite to voice" quick-invite feature
// on a bot — confirmed by capturing the actual DM content, since there is no
// dedicated API or gateway event for that client feature.
var inviteLinkPattern = regexp.MustCompile(`discord(?:app)?\.(?:gg|com/invite)/([A-Za-z0-9-]+)`)

// parseInviteCode extracts the invite code from a Discord invite link
// anywhere in content, reporting whether one was found.
func parseInviteCode(content string) (string, bool) {
	match := inviteLinkPattern.FindStringSubmatch(content)
	if match == nil {
		return "", false
	}
	return match[1], true
}

// handleInviteDM checks whether a DM's content is a Discord invite link and,
// if so, resolves it and joins the target voice channel directly — bypassing
// the guild member voice-state lookup !join relies on, which has nothing to
// look up in a DM context. It reports whether the message was an invite link
// (handled either way, success or failure), so the caller knows not to fall
// through to the normal DM refusal.
func (b *Bot) handleInviteDM(e *events.MessageCreate) bool {
	code, ok := parseInviteCode(e.Message.Content)
	if !ok {
		return false
	}

	client := e.Client()
	invite, err := client.Rest.GetInvite(code)
	if err != nil {
		slog.Error("failed to resolve invite", "Code", code, "err", err)
		return true
	}

	if invite.Guild == nil || invite.Channel == nil {
		slog.Info("invite has no voice channel target", "Code", code)
		return true
	}

	if invite.Channel.Type != discord.ChannelTypeGuildVoice && invite.Channel.Type != discord.ChannelTypeGuildStageVoice {
		slog.Info("invite does not target a voice channel",
			"Code", code,
			"ChannelType", invite.Channel.Type,
		)
		return true
	}

	b.joinVoiceChannel(client, invite.Guild.ID, invite.Channel.ID, invite.Channel.Name)
	return true
}
