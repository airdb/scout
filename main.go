package main

import (
	"context"
	"errors"
	"log"

	"github.com/airdb/scout/bootstrap"
	cachemod "github.com/airdb/scout/modules/cache"
	discordmod "github.com/airdb/scout/modules/discord"
	openaimod "github.com/airdb/scout/modules/openai"
	telemetrymod "github.com/airdb/scout/modules/telemetry"
	wecommod "github.com/airdb/scout/modules/wecom"
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

		LokiWriter *lokikit.LokiWriter
		Discord    *bootstrap.Discord
		Wecom      *bootstrap.Wecom
	}

	app := fx.New(
		telemetrymod.FxOptions(),
		cachemod.FxOptions(),
		openaimod.FxOptions(),
		discordmod.FxOptions(),
		wecommod.FxOptions(),
		bootstrap.FxOptions(),
		fx.Invoke(func(lc fx.Lifecycle, deps invokeDeps) {
			lc.Append(fx.Hook{
				OnStart: func(ctx context.Context) error {
					go deps.Discord.Start()
					go deps.Wecom.Start()
					log.Println("Press Ctrl+C to exit")
					return nil
				},
				OnStop: func(ctx context.Context) error {
					deps.LokiWriter.Shutdown()
					return errors.Join(deps.Discord.Stop(), deps.Wecom.Stop())
				},
			})
		}),
	)

	app.Run()
}
