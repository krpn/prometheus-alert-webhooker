package utils

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestMD5Hash(t *testing.T) {
	t.Parallel()

	type testTableData struct {
		tcase    string
		text     string
		expected string
	}

	testTable := []testTableData{
		{
			tcase:    "text is not empty",
			text:     "some string",
			expected: "5ac749fbeec93607fc28d666be85e73a",
		},
		{
			tcase:    "text is empty",
			text:     "",
			expected: "d41d8cd98f00b204e9800998ecf8427e",
		},
	}

	for _, testUnit := range testTable {
		assert.Equal(t, testUnit.expected, MD5Hash(testUnit.text), testUnit.tcase)
	}
}

func TestMD5HashFromTime(t *testing.T) {
	t.Parallel()

	type testTableData struct {
		tcase    string
		time     time.Time
		expected string
	}

	testTable := []testTableData{
		{
			tcase:    "text is not empty",
			time:     time.Unix(1535086351, 0),
			expected: "dc122c6b21f32e4286edd931f38b7b5e",
		},
	}

	for _, testUnit := range testTable {
		assert.Equal(t, testUnit.expected, MD5HashFromTime(testUnit.time), testUnit.tcase)
	}
}
