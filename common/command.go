package common

type Command interface {
	Execute()
}

func GetConfigString(arguments map[string]interface{}, key string) string {
	configPath, hasConfigPath := arguments[key]
	if hasConfigPath {
		value, ok := configPath.(string)
		if ok {
			return value
		}
	}
	return ""
}

func GetConfigStringArray(arguments map[string]interface{}, key string) []string {
	configPath, hasConfigPath := arguments[key]
	if hasConfigPath {
		value, ok := configPath.([]string)
		if ok {
			return value
		}
	}
	return []string{}
}

func GetConfigBool(arguments map[string]interface{}, key string) bool {
	configPath, hasConfigPath := arguments[key]
	if hasConfigPath {
		value, ok := configPath.(bool)
		if ok {
			return value
		}
	}
	return false
}

func GetConfigInt(arguments map[string]interface{}, key string) int {
	configPath, hasConfigPath := arguments[key]
	if hasConfigPath {
		value, ok := configPath.(int)
		if ok {
			return value
		}
	}
	return 0
}
