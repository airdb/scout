package main

import (
	"context"
	"log"

	"github.com/airdb/scout/bootstrap"
	discordmod "github.com/airdb/scout/modules/discord"
	openaimod "github.com/airdb/scout/modules/openai"
	"go.uber.org/fx"
)

func main() {
	app := fx.New(
		openaimod.FxOptions(),
		discordmod.FxOptions(),
		bootstrap.FxOptions(),
		fx.Invoke(func(lc fx.Lifecycle, discord *bootstrap.Discord) {
			lc.Append(fx.Hook{
				OnStart: func(ctx context.Context) error {
					go discord.Start()
					log.Println("Press Ctrl+C to exit")
					return nil
				},
				OnStop: func(ctx context.Context) error {
					return discord.Stop()
				},
			})
		}),
	)

	app.Run()
}
