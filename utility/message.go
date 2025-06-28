package utility

// Helper function to safely extract a string from a map
func GetString(data map[string]any, key string) string {
	if val, ok := data[key].(string); ok {
		return val
	}
	return ""
}
