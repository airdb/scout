package discordmod

import (
	"context"
	"fmt"
	"strings"

	openaimod "github.com/airdb/scout/modules/openai"
	"github.com/bwmarrin/discordgo"
	"github.com/gofrs/uuid"
	"go.uber.org/fx"
	"golang.org/x/exp/slog"
)

type commandHandlersDeps struct {
	fx.In

	ChatGpt *openaimod.ChatGpt
	Logger  *slog.Logger
}

func CommandHandlers(deps commandHandlersDeps) map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate) {
	return map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
		"gpt": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			requestId, err := uuid.NewV6()
			if err != nil {
				panic(err)
			}
			entry := deps.Logger.With(
				"requestID", requestId.String(),
				"user", i.Member.User.String(),
				"command", "gtp",
			)
			entry.Info("command begin")

			// Access options in the order provided by the user.
			options := i.ApplicationCommandData().Options

			// Or convert the slice into a map
			optionMap := make(map[string]*discordgo.ApplicationCommandInteractionDataOption, len(options))
			for _, opt := range options {
				optionMap[opt.Name] = opt
			}

			content := "I can't understand what you're saying"
			if opt, ok := optionMap["prompt"]; ok {
				entry.With("prompt", opt.StringValue()).Info("query chatgpt start")
				msg, err := deps.ChatGpt.GetResponse(context.TODO(), opt.StringValue())
				if err != nil {
					return
				}
				entry.With("prompt", opt.StringValue()).Info("query chatgpt over")
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
			entry.Info("command over")
		},
	}
}
