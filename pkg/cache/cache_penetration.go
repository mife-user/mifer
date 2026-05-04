package cache

import (
	"context"
	"time"

	"github.com/go-redis/redis/v8"
)

const (
	NullValueMarker = "__NULL_VALUE__"
	NullValueTTL    = 5 * time.Minute
	NullHashKey     = "null"
)

var (
	NullCacheConfig = NewCacheConfig(NullValueTTL, 1*time.Minute)
)

// CacheNullValue 缓存空值（普通 key）
func CacheNullValue(ctx context.Context, rdb *redis.Client, key string) error {
	ttl := NullCacheConfig.RandomTTL()
	return rdb.Set(ctx, key, NullValueMarker, ttl).Err()
}

// IsNullValue 判断是否为空值
func IsNullValue(value string) bool {
	return value == NullValueMarker
}

// CacheHashNullValue 缓存 Hash 结构中的空值
func CacheHashNullValue(ctx context.Context, rdb *redis.Client, hashKey, fieldKey string) error {
	ttl := NullCacheConfig.RandomTTL()
	if err := rdb.HSet(ctx, hashKey, fieldKey, NullValueMarker).Err(); err != nil {
		return err
	}
	return rdb.Expire(ctx, hashKey, ttl).Err()
}

// IsHashNullValue 判断 Hash 字段中是否为空值
func IsHashNullValue(value string) bool {
	return value == NullValueMarker
}
