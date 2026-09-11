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
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/disgo/gateway"
	"github.com/disgoorg/disgo/voice"

	"github.com/pinheirolucas/discord_instants_player/pkg/command"
	"github.com/pinheirolucas/discord_instants_player/pkg/instant"
	"github.com/pinheirolucas/discord_instants_player/pkg/opusaudio"
)

type Bot struct {
	token string
	owner string

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

	b.disp.Register("!ping", "Teste para verificar se o bot está online", b.ping)
	b.disp.Register("!join", "Chamar o bot para o canal de áudio em que você está", b.join)
	b.disp.Register("!help", "Mostrar informações de utilização", b.help)

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
		_, _ = e.Client().Rest.CreateMessage(e.ChannelID, discord.MessageCreate{
			Content: "Maninho, eu não funciono em mensagens privadas.",
		})
		return
	}

	b.disp.Dispatch(e)
}
