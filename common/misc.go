package common

func Contains(container map[string]map[string]int, key string) bool {
	for k := range container {
		if k == key {
			return true
		}
	}
	return false
}
