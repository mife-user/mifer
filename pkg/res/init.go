package res

import (
	"github.com/redis/go-redis/v9"
)

// Init 创建 Redis 客户端连接，参数由调用方传入
func Init(addr, username, password string, db, protocol int, unstableResp3 bool) (*redis.Client, error) {
	redisbase := redis.NewClient(&redis.Options{
		Addr:     addr,
		Username: username,
		Password: password,
		DB:       db,
		Protocol: protocol,
	})
	redisbase.Options().UnstableResp3 = unstableResp3
	return redisbase, nil
}
