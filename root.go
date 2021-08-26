package main

import (
	"os"
	"strings"
	"time"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/pinheirolucas/discord_instants_player/bot"
	"github.com/pinheirolucas/discord_instants_player/instant"
	"github.com/pinheirolucas/discord_instants_player/server"
)

var cfgFile string

var rootCmd = &cobra.Command{
	Use:           "discord_instants_player",
	Short:         "Application layer that manages the bot and creates an HTTP inteface for controlling the bot playback",
	SilenceErrors: true,
	SilenceUsage:  true,
	RunE:          runRootCmd,
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.discord_instants_player.yaml)")

	rootCmd.PersistentFlags().String("bot-owner", "", "bot owner username")
	viper.BindPFlag("bot.owner", rootCmd.PersistentFlags().Lookup("bot-owner"))

	rootCmd.PersistentFlags().String("bot-token", "", "application oauth token to authenticate the bot")
	viper.BindPFlag("bot.token", rootCmd.PersistentFlags().Lookup("bot-token"))

	rootCmd.PersistentFlags().String("server-address", "", "address to bind the http server")
	viper.BindPFlag("server.address", rootCmd.PersistentFlags().Lookup("server-address"))
}

func runRootCmd(cmd *cobra.Command, args []string) error {
	token := viper.GetString("bot.token")
	if strings.TrimSpace(token) == "" {
		return errors.New("bot token not provided")
	}

	owner := viper.GetString("bot.owner")
	if strings.TrimSpace(owner) == "" {
		return errors.New("bot owner not provided")
	}

	address := viper.GetString("server.address")
	if strings.TrimSpace(address) == "" {
		return errors.New("server address not provided")
	}

	errchan := make(chan error, 1)
	defer close(errchan)

	player := instant.NewPlayer()
	defer player.Close()

	b, err := bot.New(token, player, bot.WithOwner(owner))
	if err != nil {
		return errors.Wrap(err, "failed to create a bot")
	}

	go func() {
		if err := b.Start(); err != nil {
			errchan <- err
		}
	}()

	s := server.New(player)

	go func() {
		if err := s.Start(address); err != nil {
			errchan <- err
		}
	}()

	err = <-errchan

	log.Info().Err(err).Msg("")
	time.Sleep(time.Second * 3)

	return nil
}

func initConfig() {
	log.Logger = log.Output(zerolog.ConsoleWriter{
		Out:        os.Stdout,
		TimeFormat: "2006-01-02 15:04:05",
	})

	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := homedir.Dir()
		if err != nil {
			log.Fatal().Err(err).Msg("find homedir")
		}

		cwd, err := os.Getwd()
		if err != nil {
			log.Fatal().Err(err).Msg("find cwd")
		}

		viper.AddConfigPath(home)
		viper.AddConfigPath(cwd)
		viper.SetConfigName(".discord_instants_player")
	}

	replacer := strings.NewReplacer(
		".", "_",
		"-", "_",
	)
	viper.SetEnvKeyReplacer(replacer)
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil {
		log.Info().Str("configFile", viper.ConfigFileUsed()).Msg("using config file")
	}
}
