package discordmod

import (
	"os"

	"github.com/bwmarrin/discordgo"
	"go.uber.org/fx"
)

func FxOptions() fx.Option {
	return fx.Options(
		fx.Provide(func() *Config {
			return &Config{
				GuildId:        os.Getenv("DISCORD_GUILD_ID"),
				BotToken:       os.Getenv("DISCORD_BOT_TOKEN"),
				RemoveCommands: os.Getenv("DISCORD_REMOVE_COMMAND") == "true",
			}
		}),
		fx.Provide(func(cfg *Config) (*discordgo.Session, error) {
			ds, err := discordgo.New("Bot " + cfg.BotToken)
			return ds, err
		}),
		fx.Provide(func() ([]*discordgo.ApplicationCommand, error) {
			return commands, nil
		}),
		fx.Provide(CommandHandlers),
	)
}
