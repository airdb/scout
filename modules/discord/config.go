package discordmod

type Config struct {
	// Test guild ID. If not passed - bot registers commands globally
	GuildId string
	// Bot access token
	BotToken string
	// Remove all commands after shutdowning or not
	RemoveCommands bool
}
