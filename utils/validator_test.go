package utils

import (
	"errors"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestCheckMapIsNotEmpty(t *testing.T) {
	t.Parallel()

	type testTableData struct {
		tcase    string
		m        map[string]string
		expected error
	}

	testTable := []testTableData{
		{
			tcase: "correct",
			m: map[string]string{
				"a": "b",
			},
			expected: nil,
		},
		{
			tcase: "empty key",
			m: map[string]string{
				"": "b",
			},
			expected: errors.New("key is empty"),
		},
		{
			tcase: "empty value",
			m: map[string]string{
				"a": "",
			},
			expected: errors.New("value for key a is empty"),
		},
	}

	for _, testUnit := range testTable {
		assert.Equal(t, testUnit.expected, CheckMapIsNotEmpty(testUnit.m), testUnit.tcase)
	}
}
