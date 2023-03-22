package wecommod

import (
	"go.uber.org/fx"
)

func FxOptions() fx.Option {
	return fx.Options(
		fx.Provide(New),
		fx.Provide(NewReplySvc),
		fx.Provide(NewHandler),
		fx.Provide(NewTpls),
	)
}
