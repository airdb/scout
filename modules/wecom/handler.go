package wecommod

import (
	"github.com/go-redis/redis/v8"
	"github.com/silenceper/wechat/v2/work/kf"
	"go.uber.org/fx"
	"golang.org/x/exp/slog"
)

type handlerDeps struct {
	fx.In

	Redis    *redis.Client
	Kefu     *kf.Client
	ReplySvc *ReplySvc
	Logger   *slog.Logger
}

type Handler struct {
	*handlerDeps
}

func NewHandler(deps handlerDeps) *Handler {
	return &Handler{
		handlerDeps: &deps,
	}
}
