package utils

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestReplacePlaceholders(t *testing.T) {
	t.Parallel()

	type testTableData struct {
		tcase    string
		str      string
		prefix   string
		label    string
		new      string
		expected string
	}

	testTable := []testTableData{
		{
			tcase:    "${LABEL_TEST}",
			str:      "${LABEL_TEST}",
			prefix:   "LABEL",
			label:    "test",
			new:      "some replacement:8080",
			expected: "some replacement:8080",
		},
		{
			tcase:    "${URLENCODE_LABEL_TEST}",
			str:      "${URLENCODE_LABEL_TEST}",
			prefix:   "LABEL",
			label:    "test",
			new:      "some replacement:8080",
			expected: "some+replacement%3A8080",
		},
		{
			tcase:    "${CUT_AFTER_LAST_COLON_LABEL_TEST}",
			str:      "${CUT_AFTER_LAST_COLON_LABEL_TEST}",
			prefix:   "LABEL",
			label:    "test",
			new:      "some replacement:8080",
			expected: "some replacement",
		},
		{
			tcase:    "${CUT_AFTER_LAST_COLON_LABEL_TEST} no colon found",
			str:      "${CUT_AFTER_LAST_COLON_LABEL_TEST}",
			prefix:   "LABEL",
			label:    "test",
			new:      "some replacement 8080",
			expected: "some replacement 8080",
		},
		{
			tcase:    "${JSON_ESCAPE_LABEL_TEST}",
			str:      "${JSON_ESCAPE_LABEL_TEST}",
			prefix:   "LABEL",
			label:    "test",
			new:      `some \replacement "8080`,
			expected: `some \\replacement \"8080`,
		},
	}

	for _, testUnit := range testTable {
		assert.Equal(t, testUnit.expected, ReplacePlaceholders(testUnit.str, testUnit.prefix, testUnit.label, testUnit.new), testUnit.tcase)
	}
}
