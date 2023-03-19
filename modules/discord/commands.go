package discordmod

import (
	"github.com/bwmarrin/discordgo"
)

var (
	integerOptionMinValue          = 1.0
	dmPermission                   = false
	defaultMemberPermissions int64 = discordgo.PermissionManageServer
)

var commands = []*discordgo.ApplicationCommand{
	{
		Name: "gpt",
		// All commands and options must have a description
		// Commands/options without description will fail the registration
		// of the command.
		Description: "Chat with OpenAI",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "prompt",
				Description: "Then content of your promps",
				Required:    true,
			},
		},
	},
}
