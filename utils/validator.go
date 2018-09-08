package utils

import (
	"errors"
	"fmt"
)

// CheckMapIsNotEmpty returns error if any key or value from map is empty.
func CheckMapIsNotEmpty(m map[string]string) error {
	for key, value := range m {
		if len(key) == 0 {
			return errors.New("key is empty")
		}

		if len(value) == 0 {
			return fmt.Errorf("value for key %v is empty", key)
		}
	}

	return nil
}
