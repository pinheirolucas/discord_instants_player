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
	Short:         "A brief description of your application",
	SilenceErrors: true,
	SilenceUsage:  true,
	RunE:          runRootCmd,
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.discord_instants_player.yaml)")

	rootCmd.PersistentFlags().String("owner", "", "username for the bot owner")
	viper.BindPFlag("discord_instants_player_owner", rootCmd.PersistentFlags().Lookup("owner"))

	rootCmd.PersistentFlags().String("token", "", "bot token to bind the application")
	viper.BindPFlag("discord_instants_player_token", rootCmd.PersistentFlags().Lookup("token"))

	rootCmd.PersistentFlags().String("address", "", "address to bind the http server")
	viper.BindPFlag("discord_instants_player_address", rootCmd.PersistentFlags().Lookup("address"))
}

func runRootCmd(cmd *cobra.Command, args []string) error {
	token := viper.GetString("discord_instants_player_token")
	if strings.TrimSpace(token) == "" {
		return errors.New("token not provided")
	}

	owner := viper.GetString("discord_instants_player_owner")
	if strings.TrimSpace(owner) == "" {
		return errors.New("owner not provided")
	}

	address := viper.GetString("discord_instants_player_address")
	if strings.TrimSpace(address) == "" {
		return errors.New("address not provided")
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

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil {
		log.Info().Str("configFile", viper.ConfigFileUsed()).Msg("using config file")
	}
}
