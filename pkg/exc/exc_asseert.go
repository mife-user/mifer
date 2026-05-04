package exc

// IsUint 检查是否为uint类型
func IsUint(v any) (uint, bool) {
	r, ok := v.(uint)
	return r, ok
}

// IsString 检查是否为string类型
func IsString(v any) (string, bool) {
	r, ok := v.(string)
	return r, ok
}
