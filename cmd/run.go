package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/pinheirolucas/discord_instants_player/bot"
	"github.com/pinheirolucas/discord_instants_player/server"
)

var conn *discordgo.VoiceConnection

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run the discord instants player app",
	RunE:  run,
}

func run(cmd *cobra.Command, args []string) error {
	token := viper.GetString("token")
	if strings.TrimSpace(token) == "" {
		return errors.New("token not provided")
	}

	owner := viper.GetString("owner")

	host := viper.GetString("host")
	if strings.TrimSpace(host) == "" {
		return errors.New("host not provided")
	}

	port := viper.GetInt("port")
	if port == 0 {
		return errors.New("port not provided")
	}

	address := fmt.Sprintf("%s:%d", host, port)

	errchan := make(chan error, 1)
	defer close(errchan)

	playchan := make(chan string, 1)
	defer close(playchan)

	audioEndedChan := make(chan bool, 1)
	defer close(audioEndedChan)

	b, err := bot.New(token, playchan, audioEndedChan, bot.WithOwner(owner))
	if err != nil {
		return errors.Wrap(err, "failed to create a bot")
	}

	go func() {
		if err := b.Start(); err != nil {
			errchan <- err
		}
	}()

	s := server.New(playchan, audioEndedChan)

	go func() {
		if err := s.Start(address); err != nil {
			errchan <- err
		}
	}()

	err = <-errchan

	log.Info().Msg(err.Error())
	time.Sleep(time.Second * 3)

	return nil
}

func init() {
	rootCmd.AddCommand(runCmd)
}
