package utils

import (
	"crypto/rand"
	"encoding/base64"
)

// generateRandomString 生成随机字符串
func RandomStr(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes)[:length], nil
}
