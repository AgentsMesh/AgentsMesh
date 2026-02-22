package mcp

// Helper to extract string from args
func getStringArg(args map[string]interface{}, key string) string {
	if v, ok := args[key].(string); ok {
		return v
	}
	return ""
}

// Helper to extract string pointer from args (nil if absent or empty)
func getStringPtrArg(args map[string]interface{}, key string) *string {
	if v, ok := args[key].(string); ok && v != "" {
		return &v
	}
	return nil
}

// Helper to extract int from args
func getIntArg(args map[string]interface{}, key string) int {
	switch v := args[key].(type) {
	case float64:
		return int(v)
	case int:
		return v
	}
	return 0
}

// Helper to extract bool from args
func getBoolArg(args map[string]interface{}, key string) bool {
	if v, ok := args[key].(bool); ok {
		return v
	}
	return false
}

// Helper to extract int pointer from args
func getIntPtrArg(args map[string]interface{}, key string) *int {
	switch v := args[key].(type) {
	case float64:
		i := int(v)
		return &i
	case int:
		return &v
	}
	return nil
}

// Helper to extract int64 pointer from args
func getInt64PtrArg(args map[string]interface{}, key string) *int64 {
	switch v := args[key].(type) {
	case float64:
		i := int64(v)
		return &i
	case int:
		i := int64(v)
		return &i
	case int64:
		return &v
	}
	return nil
}

// Helper to extract string slice from args
func getStringSliceArg(args map[string]interface{}, key string) []string {
	if v, ok := args[key].([]interface{}); ok {
		result := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				result = append(result, s)
			}
		}
		return result
	}
	return nil
}

// Helper to extract map from args
func getMapArg(args map[string]interface{}, key string) map[string]interface{} {
	if v, ok := args[key].(map[string]interface{}); ok {
		return v
	}
	return nil
}
