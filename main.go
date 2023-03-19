package main

import (
	"context"
	"log"

	"github.com/airdb/scout/bootstrap"
	discordmod "github.com/airdb/scout/modules/discord"
	openaimod "github.com/airdb/scout/modules/openai"
	telemetrymod "github.com/airdb/scout/modules/telemetry"
	"github.com/airdb/scout/pkg/lokikit"
	"github.com/joho/godotenv"
	"go.uber.org/fx"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	type invokeDeps struct {
		fx.In

		Discord    *bootstrap.Discord
		LokiWriter *lokikit.LokiWriter
	}

	app := fx.New(
		openaimod.FxOptions(),
		telemetrymod.FxOptions(),
		discordmod.FxOptions(),
		bootstrap.FxOptions(),
		fx.Invoke(func(lc fx.Lifecycle, deps invokeDeps) {
			lc.Append(fx.Hook{
				OnStart: func(ctx context.Context) error {
					go deps.Discord.Start()
					log.Println("Press Ctrl+C to exit")
					return nil
				},
				OnStop: func(ctx context.Context) error {
					deps.LokiWriter.Shutdown()
					return deps.Discord.Stop()
				},
			})
		}),
	)

	app.Run()
}
