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
			tcase:    "${CUT_AFTER_LAST_COLON_URLENCODE_LABEL_TEST}",
			str:      "${CUT_AFTER_LAST_COLON_URLENCODE_LABEL_TEST}",
			prefix:   "LABEL",
			label:    "test",
			new:      "some replacement:8080",
			expected: "some+replacement",
		},
	}

	for _, testUnit := range testTable {
		assert.Equal(t, testUnit.expected, ReplacePlaceholders(testUnit.str, testUnit.prefix, testUnit.label, testUnit.new), testUnit.tcase)
	}
}

func TestTrimStringFromSymbol(t *testing.T) {
	t.Parallel()

	type testTableData struct {
		str      string
		symbol   string
		expected string
	}

	testTable := []testTableData{
		{
			str:      "server.domain.com:9090",
			symbol:   ":",
			expected: "server.domain.com",
		},
		{
			str:      "server.domain.com",
			symbol:   ":",
			expected: "server.domain.com",
		},
		{
			str:      "server:domain:com:9090",
			symbol:   ":",
			expected: "server:domain:com",
		},
	}

	for _, testUnit := range testTable {
		assert.Equal(t, testUnit.expected, trimStringFromSymbol(testUnit.str, testUnit.symbol))
	}
}
