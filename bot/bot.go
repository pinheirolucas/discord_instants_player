package bot

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/bwmarrin/dgvoice"
	"github.com/bwmarrin/discordgo"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"

	"github.com/pinheirolucas/discord_instants_player/command"
	"github.com/pinheirolucas/discord_instants_player/instant"
)

type Bot struct {
	token string
	owner string

	client *discordgo.Session
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
		return errors.Wrap(err, "failed to create a client")
	}
	defer client.Close()

	client.AddHandler(b.handleReady)
	client.AddHandler(b.handleMessages)

	// For some reason dg voice thinks that logging errors by himself is a good idea.
	// The line below disables those logs.
	dgvoice.OnError = func(str string, err error) {}

	if err = client.Open(); err != nil {
		return errors.Wrap(err, "failed to open websocket connection")
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

			log.Info().Str("path", path).Msg("playing instant")
			dgvoice.PlayAudioFile(b.vc, path, b.player.StopChan)
			b.player.End()
		}
	}()

	log.Info().Msg("bot is now running")

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	return errors.New("application is shutting down")
}

func (b *Bot) handleReady(s *discordgo.Session, r *discordgo.Ready) {
	log.Info().Msg("bot is ready")
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
