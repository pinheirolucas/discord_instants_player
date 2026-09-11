package cmd

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/pinheirolucas/discord_instants_player/pkg/bot"
	"github.com/pinheirolucas/discord_instants_player/pkg/instant"
	"github.com/pinheirolucas/discord_instants_player/pkg/server"
)

var cfgFile string

var rootCmd = &cobra.Command{
	Use:           "discord_instants_player",
	Short:         "Application layer that manages the bot and creates an HTTP inteface for controlling the bot playback",
	SilenceErrors: true,
	SilenceUsage:  true,
	RunE:          runRootCmd,
}

// Execute ...
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
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
		return fmt.Errorf("failed to create a bot: %w", err)
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

	slog.Error("", "err", err)
	time.Sleep(time.Second * 3)

	return nil
}

func initConfig() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				a.Value = slog.StringValue(a.Value.Time().Format("2006-01-02 15:04:05"))
			}
			return a
		},
	})))

	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			slog.Error("find homedir", "err", err)
			os.Exit(1)
		}

		cwd, err := os.Getwd()
		if err != nil {
			slog.Error("find cwd", "err", err)
			os.Exit(1)
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
		slog.Info("using config file", "configFile", viper.ConfigFileUsed())
	}
}
