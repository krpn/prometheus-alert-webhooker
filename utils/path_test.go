package utils

import (
	"github.com/stretchr/testify/assert"
	"net/url"
	"testing"
)

func TestParsePath(t *testing.T) {
	t.Parallel()

	type testTableData struct {
		tcase             string
		rawPath           string
		defaultExtension  string
		configProvider    string
		expectedEndpoint  string
		expectedPath      string
		expectedExtension string
		expectedErr       error
	}

	testTable := []testTableData{
		{
			tcase:             "yaml",
			rawPath:           "config.yaml",
			defaultExtension:  "yaml",
			configProvider:    "file",
			expectedEndpoint:  "",
			expectedPath:      "config.yaml",
			expectedExtension: "yaml",
			expectedErr:       nil,
		},
		{
			tcase:             "json",
			rawPath:           "config/config.json",
			defaultExtension:  "yaml",
			configProvider:    "file",
			expectedEndpoint:  "",
			expectedPath:      "config/config.json",
			expectedExtension: "json",
			expectedErr:       nil,
		},
		{
			tcase:             "empty extension",
			rawPath:           "config",
			defaultExtension:  "yaml",
			configProvider:    "file",
			expectedEndpoint:  "",
			expectedPath:      "config",
			expectedExtension: "yaml",
			expectedErr:       nil,
		},
		{
			tcase:             "json url",
			rawPath:           "http://127.0.0.1:4001/config/hugo.json",
			defaultExtension:  "yaml",
			configProvider:    "etcd",
			expectedEndpoint:  "http://127.0.0.1:4001",
			expectedPath:      "/config/hugo.json",
			expectedExtension: "json",
			expectedErr:       nil,
		},
		{
			tcase:             "json url with get param",
			rawPath:           "http://127.0.0.1:4001/config/hugo.json?ver=1",
			defaultExtension:  "yaml",
			configProvider:    "etcd",
			expectedEndpoint:  "http://127.0.0.1:4001",
			expectedPath:      "/config/hugo.json?ver=1",
			expectedExtension: "json",
			expectedErr:       nil,
		},
		{
			tcase:             "uncorrect url",
			rawPath:           "http://127 0 0 1:4001/config/hugo.json?ver=1",
			defaultExtension:  "yaml",
			configProvider:    "etcd",
			expectedEndpoint:  "",
			expectedPath:      "",
			expectedExtension: "",
			expectedErr:       &url.Error{Op: "parse", URL: "http://127 0 0 1:4001/config/hugo.json?ver=1", Err: url.InvalidHostError(" ")},
		},
		{
			tcase:             "json url with get param",
			rawPath:           "http://techserv1.adsterratech.com:8500/v1/kv/common/db.json",
			defaultExtension:  "json",
			configProvider:    "consul",
			expectedEndpoint:  "techserv1.adsterratech.com:8500",
			expectedPath:      "common/db.json",
			expectedExtension: "json",
			expectedErr:       nil,
		},
	}

	for _, testUnit := range testTable {
		endpoint, path, extension, err := ParsePath(testUnit.rawPath, testUnit.defaultExtension, testUnit.configProvider)
		assert.Equal(t, testUnit.expectedEndpoint, endpoint, testUnit.tcase)
		assert.Equal(t, testUnit.expectedPath, path, testUnit.tcase)
		assert.Equal(t, testUnit.expectedExtension, extension, testUnit.tcase)
		assert.Equal(t, testUnit.expectedErr, err, testUnit.tcase)
	}
}
