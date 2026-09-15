package bot

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/disgoorg/disgo"
	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/cache"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/disgo/gateway"
	"github.com/disgoorg/disgo/voice"
	davesession "github.com/thomas-vilte/dave-go/session"
	"golang.org/x/text/language"

	"github.com/pinheirolucas/peace-breaker-bot/pkg/command"
	"github.com/pinheirolucas/peace-breaker-bot/pkg/i18n"
	"github.com/pinheirolucas/peace-breaker-bot/pkg/instant"
	"github.com/pinheirolucas/peace-breaker-bot/pkg/opusaudio"
)

type Bot struct {
	token string
	owner string

	// locale overrides per-guild locale detection when set (bot.locale),
	// for a single-owner bot that wants a fixed response language.
	locale string

	vc     voice.Conn
	disp   *command.DiscordDispatcher
	player *instant.Player
}

func New(token string, player *instant.Player, options ...Option) (*Bot, error) {
	b := &Bot{
		token:  token,
		disp:   command.NewDiscordDispatcher(),
		player: player,
	}

	b.disp.Register("!ping", "bot.ping.help", b.ping)
	b.disp.Register("!join", "bot.join.help", b.join)
	b.disp.Register("!leave", "bot.leave.help", b.leave)
	b.disp.Register("!help", "bot.help.help", b.help)

	for _, option := range options {
		option(b)
	}

	return b, nil
}

func (b *Bot) Start() error {
	client, err := disgo.New(b.token,
		bot.WithGatewayConfigOpts(
			gateway.WithIntents(
				gateway.IntentGuilds,
				gateway.IntentGuildVoiceStates,
				gateway.IntentGuildMessages,
				gateway.IntentDirectMessages,
				gateway.IntentMessageContent,
			),
		),
		// Guilds/Channels/VoiceStates are uncached by default; !join looks
		// all three up, so they must be explicitly enabled here.
		bot.WithCacheConfigOpts(
			cache.WithCaches(cache.FlagGuilds, cache.FlagChannels, cache.FlagVoiceStates),
		),
		bot.WithEventListenerFunc(b.handleReady),
		bot.WithEventListenerFunc(b.handleMessages),
		// Listeners run synchronously on the gateway's websocket read loop
		// unless this is set. !join blocks on the voice handshake, which
		// would otherwise stall that loop long enough to miss heartbeat
		// ACKs and get disconnected as a zombie connection.
		bot.WithEventManagerConfigOpts(bot.WithAsyncEventsEnabled()),
		// dave-go is a pure-Go DAVE/E2EE implementation; without a session
		// factory here voice defaults to godave's noop (unencrypted) session,
		// which Discord's voice gateway rejects with close code 4017.
		bot.WithVoiceManagerConfigOpts(
			voice.WithDaveSessionCreateFunc(davesession.CreateFunc()),
		),
	)
	if err != nil {
		return fmt.Errorf("failed to create a client: %w", err)
	}
	defer client.Close(context.Background())

	opusaudio.OnError = func(str string, err error) {
		slog.Debug(str, "err", err)
	}

	if err = client.OpenGateway(context.Background()); err != nil {
		return fmt.Errorf("failed to open websocket connection: %w", err)
	}

	defer func() {
		if b.vc == nil {
			return
		}

		b.vc.Close(context.Background())
	}()

	go func() {
		// TODO: create a bot client to manage all this complexity
		for {
			path := b.player.GetNextPlay()

			if b.vc == nil {
				b.player.End()
				continue
			}

			slog.Info("playing instant", "path", path)
			opusaudio.PlayAudioFile(b.vc, path, b.player.StopChan)
			b.player.End()
		}
	}()

	slog.Info("bot is now running")

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	return errors.New("application is shutting down")
}

func (b *Bot) handleReady(e *events.Ready) {
	slog.Info("bot is ready")
}

func (b *Bot) handleMessages(e *events.MessageCreate) {
	if b.owner != "" && b.owner != e.Message.Author.Username {
		return
	}

	if e.Message.Author.ID == e.Client().ID() {
		return
	}

	if e.GuildID == nil {
		b.handleInviteDM(e)
		return
	}

	b.disp.Dispatch(e)
}

// localeFor resolves the language a response to e should be written in: the
// configured bot.locale override always wins, for a single-owner bot that
// wants a fixed language regardless of guild; otherwise the invoking guild's
// own PreferredLocale. A DM carries no guild at all, so it falls straight
// through to pkg/i18n's own English default.
func (b *Bot) localeFor(e *events.MessageCreate) language.Tag {
	if b.locale != "" {
		return i18n.Match(b.locale)
	}

	if e.GuildID == nil {
		return i18n.Supported[0]
	}

	guild, ok := e.Client().Caches.Guild(*e.GuildID)
	if !ok {
		return i18n.Supported[0]
	}

	return i18n.Match(guild.PreferredLocale)
}
