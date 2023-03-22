package wecommod

import (
	"context"
	"fmt"

	"github.com/go-redis/redis/v8"
	"github.com/silenceper/wechat/v2/work/kf"
	"golang.org/x/exp/slog"
)

const (
	InviteImagePrefix     = "wechat:kf:invite:image"
	SentMsgPrefix         = "wechat:kf:sent"
	SyncMsgNextCursor     = "wechat:kf:sync_msg:next_cursor"
	SyncMsgTokenProcessed = "wechat:kf:msg_token:processed"
	SyncMsgProcessed      = "wechat:kf:msg:processed"
)

type ctxKey int

const (
	redisCtxKey  ctxKey = 1
	weKfCtxKey   ctxKey = 2
	loggerCtxKey ctxKey = 3
)

func WithCache(ctx context.Context, val *redis.Client) context.Context {
	return context.WithValue(ctx, redisCtxKey, val)
}

func MustFromCache(ctx context.Context) *redis.Client {
	if val, ok := ctx.Value(redisCtxKey).(*redis.Client); ok {
		return val
	}
	panic(fmt.Errorf("can not find redis client"))
}

func WithWekf(ctx context.Context, val *kf.Client) context.Context {
	return context.WithValue(ctx, weKfCtxKey, val)
}

func MustFromWekf(ctx context.Context) *kf.Client {
	if val, ok := ctx.Value(weKfCtxKey).(*kf.Client); ok {
		return val
	}
	panic(fmt.Errorf("can not find wecom kf"))
}

func WithLogger(ctx context.Context, val *slog.Logger) context.Context {
	return context.WithValue(ctx, loggerCtxKey, val)
}

func MustFromLogger(ctx context.Context) *slog.Logger {
	if val, ok := ctx.Value(loggerCtxKey).(*slog.Logger); ok {
		return val
	}
	panic(fmt.Errorf("can not find wecom kf"))
}
