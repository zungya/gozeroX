package consumer

// toInt64 将 interface{} 转换为 int64（JSON 数字默认解析为 float64）
func toInt64(v interface{}) int64 {
	if v == nil {
		return 0
	}
	if f, ok := v.(float64); ok {
		return int64(f)
	}
	return 0
}

// toString 将 interface{} 转换为 string
func toString(v interface{}) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}
