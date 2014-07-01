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

func getConfigStringArray(arguments map[string]interface{}, key string) []string {
	configPath, hasConfigPath := arguments[key]
	if hasConfigPath {
		value, ok := configPath.([]string)
		if ok {
			return value
		}
	}
	return []string{}
}

func getConfigBool(arguments map[string]interface{}, key string) bool {
	configPath, hasConfigPath := arguments[key]
	if hasConfigPath {
		value, ok := configPath.(bool)
		if ok {
			return value
		}
	}
	return false
}

func getConfigInt(arguments map[string]interface{}, key string) int {
	configPath, hasConfigPath := arguments[key]
	if hasConfigPath {
		value, ok := configPath.(int)
		if ok {
			return value
		}
	}
	return 0
}

func GetCommand(arguments map[string]interface{}) string {
	if getConfigBool(arguments, "render") {
		return "render"
	} else if getConfigBool(arguments, "renderV2") {
		return "renderV2"
	} else if getConfigBool(arguments, "verify") {
		return "verify"
	}
	return "daemon"
}
