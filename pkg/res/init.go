package res

import (
	"fmt"

	"mifer/pkg/conf"

	"github.com/go-redis/redis/v8"
)

// Redis初始化
func Init(config *conf.Config) (*redis.Client, error) {
	dsn := fmt.Sprintf("%s:%s", config.Redis.Host, config.Redis.Port)
	redisbase := redis.NewClient(&redis.Options{
		Addr:     dsn,
		Username: config.Redis.Username,
		Password: config.Redis.Password,
		DB:       config.Redis.DB,
	})
	return redisbase, nil
}
