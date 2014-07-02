package common

func Contains(container []string, key string) bool {
	for _, s := range container {
		if s == key {
			return true
		}
	}
	return false
}
