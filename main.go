package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/bwmarrin/dgvoice"
	"github.com/bwmarrin/discordgo"
)

var conn *discordgo.VoiceConnection

func main() {
	bot, err := discordgo.New("Bot Njg4MjI2MjI1MDQyMzU4Mjg5.XmxOqQ.LX2epEjT8D6LJbTtOAC7HcoIpX0")
	if err != nil {
		fmt.Printf("failed to create a client: %v", err)
		os.Exit(1)
	}
	defer bot.Close()

	bot.AddHandler(handleReady)
	bot.AddHandler(handleMessages)

	if err = bot.Open(); err != nil {
		fmt.Printf("failed to open websocket connection: %v", err)
		os.Exit(1)
	}

	dgvoice.OnError = func(str string, err error) {}

	defer func() {
		if conn != nil {
			conn.Close()
		}
	}()

	fmt.Println("Bot is now running. Press CTRL + C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc
}

func handleReady(s *discordgo.Session, r *discordgo.Ready) {
	fmt.Println("I'm ready!")
}

func handleMessages(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return
	}

	if m.GuildID == "" {
		s.ChannelMessageSend(m.ChannelID, "Esse bot não funciona em mensagens privadas.")
		return
	}

	guild, err := s.State.Guild(m.GuildID)
	if err != nil {
		fmt.Printf("failed to fetch guild %s info: %v\n", m.GuildID, err)
		return
	}

	var currentVoiceChannel *discordgo.Channel
	for _, vs := range guild.VoiceStates {
		if vs.UserID != m.Author.ID {
			continue
		}

		channel, err := s.State.Channel(vs.ChannelID)
		if err != nil {
			fmt.Printf("failed to fetch the voice channel %s info: %v\n", vs.ChannelID, err)
			return
		}

		currentVoiceChannel = channel
	}

	if currentVoiceChannel == nil {
		fmt.Println("voice channel not found for", m.Author.Username)
		return
	}

	if conn == nil {
		connection, err := s.ChannelVoiceJoin(m.GuildID, currentVoiceChannel.ID, false, true)
		if err != nil {
			fmt.Printf("failed to join %s@%s: %v", m.GuildID, m.ChannelID, err)
			return
		}
		conn = connection
	}

	dgvoice.PlayAudioFile(conn, "e-morreu-didi.mp3", make(chan bool))
}
