package cmd

import (
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/pinheirolucas/discord_instants_player/bot"
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

	b, err := bot.New(token, bot.WithOwner(owner))
	if err != nil {
		return errors.Wrap(err, "failed to create a bot")
	}

	if err := b.Start(); err != nil {
		return err
	}

	return nil
}

func init() {
	rootCmd.AddCommand(runCmd)
}
