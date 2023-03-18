package discordmod

import (
	"context"
	"fmt"
	"strings"

	openaimod "github.com/airdb/scout/modules/openai"
	"github.com/bwmarrin/discordgo"
	"go.uber.org/fx"
)

type commandHandlersDeps struct {
	fx.In

	ChatGpt *openaimod.ChatGpt
}

func CommandHandlers(deps commandHandlersDeps) map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate) {
	return map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
		"gpt": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			// Access options in the order provided by the user.
			options := i.ApplicationCommandData().Options

			// Or convert the slice into a map
			optionMap := make(map[string]*discordgo.ApplicationCommandInteractionDataOption, len(options))
			for _, opt := range options {
				optionMap[opt.Name] = opt
			}

			content := "I can't understand what you're saying"
			if opt, ok := optionMap["prompt"]; ok {
				msg, err := deps.ChatGpt.GetResponse(context.TODO(), opt.StringValue())
				if err != nil {
					return
				}
				content = fmt.Sprintf(
					"> %s - <@%s>\n%s",
					opt.StringValue(),
					i.Member.User.ID,
					strings.Trim(msg, "\n "),
				)
			}

			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				// Ignore type for now, they will be discussed in "responses"
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: content,
				},
			})
		},
	}

}
