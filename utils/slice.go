package utils

// StringSliceContains check slice contains string.
func StringSliceContains(slice []string, str string) bool {
	for _, el := range slice {
		if el == str {
			return true
		}
	}
	return false
}
