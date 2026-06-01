package exc

import (
	"encoding/json"
)

// 将任意结构体序列化为JSON字符串
func ExcFileToJSON(file interface{}) (string, error) {
	fileJSON, err := json.Marshal(file)
	if err != nil {
		return "", err
	}
	return string(fileJSON), nil
}

// 将JSON字符串反序列化为任意结构体
func ExcJSONToFile(fileJSON string, file interface{}) error {
	err := json.Unmarshal([]byte(fileJSON), file)
	if err != nil {
		return err
	}
	return nil
}
