package bot

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/bwmarrin/discordgo"

	"github.com/pinheirolucas/discord_instants_player/pkg/command"
	"github.com/pinheirolucas/discord_instants_player/pkg/dgvoice"
	"github.com/pinheirolucas/discord_instants_player/pkg/instant"
)

type Bot struct {
	token string
	owner string

	vc     *discordgo.VoiceConnection
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
	client, err := discordgo.New("Bot " + b.token)
	if err != nil {
		return fmt.Errorf("failed to create a client: %w", err)
	}
	defer client.Close()

	client.Identify.Intents = discordgo.IntentGuilds |
		discordgo.IntentGuildVoiceStates |
		discordgo.IntentGuildMessages |
		discordgo.IntentDirectMessages |
		discordgo.IntentMessageContent

	client.AddHandler(b.handleReady)
	client.AddHandler(b.handleMessages)

	dgvoice.OnError = func(str string, err error) {
		slog.Debug(str, "err", err)
	}

	if err = client.Open(); err != nil {
		return fmt.Errorf("failed to open websocket connection: %w", err)
	}

	defer func() {
		if b.vc == nil {
			return
		}

		b.vc.Close()
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
			dgvoice.PlayAudioFile(b.vc, path, b.player.StopChan)
			b.player.End()
		}
	}()

	slog.Info("bot is now running")

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	return errors.New("application is shutting down")
}

func (b *Bot) handleReady(s *discordgo.Session, r *discordgo.Ready) {
	slog.Info("bot is ready")
}

func (b *Bot) handleMessages(s *discordgo.Session, m *discordgo.MessageCreate) {
	if b.owner != "" && b.owner != m.Author.Username {
		return
	}

	if m.Author.ID == s.State.User.ID {
		return
	}

	if m.GuildID == "" {
		s.ChannelMessageSend(m.ChannelID, "Maninho, eu não funciono em mensagens privadas.")
		return
	}

	b.disp.Dispatch(s, m)
}
