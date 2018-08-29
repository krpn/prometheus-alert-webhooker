package utils

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestStringSliceContains(t *testing.T) {
	t.Parallel()

	type testTableData struct {
		tcase    string
		slice    []string
		str      string
		expected bool
	}

	testTable := []testTableData{
		{
			tcase:    "contains",
			slice:    []string{"a", "b", "c"},
			str:      "b",
			expected: true,
		},
		{
			tcase:    "not contains",
			slice:    []string{"a", "b", "c"},
			str:      "d",
			expected: false,
		},
	}

	for _, testUnit := range testTable {
		assert.Equal(t, testUnit.expected, StringSliceContains(testUnit.slice, testUnit.str), testUnit.tcase)
	}
}
