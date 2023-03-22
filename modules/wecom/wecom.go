package wecommod

import (
	"context"
	"os"
	"strconv"

	"github.com/go-redis/redis/v8"
	"github.com/silenceper/wechat/v2/cache"
	"github.com/silenceper/wechat/v2/work"
	"github.com/silenceper/wechat/v2/work/config"
	"github.com/silenceper/wechat/v2/work/kf"
)

func New(c *redis.Client) (*kf.Client, error) {
	redisDB, err := strconv.Atoi(os.Getenv("WECOM_REDIS_DB"))
	if err != nil {
		redisDB = 2
	}

	// SDK
	cfg := &config.Config{
		CorpID:     os.Getenv("WECOM_CORP_ID"),
		AgentID:    os.Getenv("WECOM_AGENT_ID"),
		CorpSecret: os.Getenv("WECOM_SECRET"),
		// AgentID: "",
		Cache: cache.NewRedis(context.TODO(), &cache.RedisOpts{
			Host:     os.Getenv("REDIS_ADDR"),
			Password: os.Getenv("REDIS_PASSWD"),
			Database: redisDB,
		}),
		RasPrivateKey:  "",
		Token:          os.Getenv("WECOM_TOKEN"),
		EncodingAESKey: os.Getenv("WECOM_ENCODING_AES_KEY"),
	}

	clientWork := work.NewWork(cfg)

	return clientWork.GetKF()
}
