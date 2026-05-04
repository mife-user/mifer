package cache

import (
	cryptoRand "crypto/rand"
	"math"
	"math/big"
	"time"
)

// CacheConfig 缓存配置
type CacheConfig struct {
	BaseTTL     time.Duration // 基础 TTL
	RandomRange time.Duration // 随机偏移范围
}

// NewCacheConfig 创建缓存配置
func NewCacheConfig(baseTTL, randomRange time.Duration) *CacheConfig {
	return &CacheConfig{
		BaseTTL:     baseTTL,
		RandomRange: randomRange,
	}
}

// RandomTTL 生成带随机偏移的缓存过期时间
func (c *CacheConfig) RandomTTL() time.Duration {
	if c.RandomRange <= 0 {
		return c.BaseTTL
	}

	// 生成 0 到 2*RandomRange 之间的随机整数
	max := big.NewInt(int64(c.RandomRange * 2))
	n, err := cryptoRand.Int(cryptoRand.Reader, max)
	if err != nil {
		// 如果随机失败，返回基础 TTL
		return c.BaseTTL
	}

	// 计算偏移量：从 -RandomRange 到 +RandomRange
	offset := time.Duration(n.Int64()) - c.RandomRange
	return c.BaseTTL + offset
}

// RandomTTLWithSeed 生成带随机偏移的缓存过期时间（使用种子）
func RandomTTLWithSeed(baseTTL, randomRange time.Duration) time.Duration {
	if randomRange <= 0 {
		return baseTTL
	}

	// 生成 0 到 2*randomRange 之间的随机整数
	max := big.NewInt(int64(randomRange * 2))
	n, err := cryptoRand.Int(cryptoRand.Reader, max)
	if err != nil {
		// 如果随机失败，返回基础 TTL
		return baseTTL
	}

	// 计算偏移量：从 -randomRange 到 +randomRange
	offset := time.Duration(n.Int64()) - randomRange
	return baseTTL + offset
}

// RandomInt 生成指定范围内的随机整数（加密安全）
func RandomInt(min, max int64) (int64, error) {
	if min > max {
		min, max = max, min
	}
	diff := max - min + 1
	n, err := cryptoRand.Int(cryptoRand.Reader, big.NewInt(diff))
	if err != nil {
		return 0, err
	}
	return n.Int64() + min, nil
}

// RandomFloat 生成指定范围内的随机浮点数（加密安全）
func RandomFloat(min, max float64) (float64, error) {
	if min > max {
		min, max = max, min
	}
	diff := max - min

	// 生成 0 到 1 之间的随机数
	maxInt := big.NewInt(math.MaxInt64)
	n, err := cryptoRand.Int(cryptoRand.Reader, maxInt)
	if err != nil {
		return (min + max) / 2, err
	}

	random := float64(n.Int64()) / float64(math.MaxInt64)
	return min + random*diff, nil
}

// 预定义的缓存配置
var (
	// UserCacheConfig 用户缓存配置：基础 1 小时，随机偏移 ±10 分钟
	UserCacheConfig = NewCacheConfig(1*time.Hour, 10*time.Minute)

	// FileCacheConfig 文件缓存配置：基础 3 小时，随机偏移 ±30 分钟
	FileCacheConfig = NewCacheConfig(3*time.Hour, 30*time.Minute)

	// FavoriteCacheConfig 收藏缓存配置：基础 30 分钟，随机偏移 ±5 分钟
	FavoriteCacheConfig = NewCacheConfig(30*time.Minute, 5*time.Minute)

	// ShareCacheConfig 分享缓存配置：基础 15 分钟，随机偏移 ±3 分钟
	ShareCacheConfig = NewCacheConfig(15*time.Minute, 3*time.Minute)
)
