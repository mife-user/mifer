package res

import (
	"fmt"

	"mifer/pkg/conf"

	"github.com/redis/go-redis/v9"
)

// Redis初始化
func Init() (*redis.Client, error) {
	config := conf.GetConfig()
	dsn := fmt.Sprintf("%s:%s", config.Redis.Host, config.Redis.Port)
	redisbase := redis.NewClient(&redis.Options{
		Addr:     dsn,
		Username: config.Redis.Username,
		Password: config.Redis.Password,
		DB:       config.Redis.DB,
		Protocol: config.Redis.Protocol,
	})
	redisbase.Options().UnstableResp3 = config.Redis.UnstableResp3
	return redisbase, nil
}
