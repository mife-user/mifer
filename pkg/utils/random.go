package utils

import (
	"crypto/rand"
	"encoding/base64"
	"hash/fnv"
	"strconv"
)

// generateRandomString 生成随机字符串
func RandomStr(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes)[:length], nil
}

// PseudoRandom 伪随机函数，输入同样的 byte 输出同样的数字（基于 FNV-1a 64 位哈希）
func PseudoRandom(input []byte) string {
	h := fnv.New64a()
	h.Write(input)
	return strconv.FormatUint(h.Sum64(), 10)
}
