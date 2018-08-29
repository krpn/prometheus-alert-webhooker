package utils

import (
	"net/url"
	"path/filepath"
	"strings"
)

// ParsePath parses given path for endpoint, path and extension.
// Example:
//     rawPath  : http://127.0.0.1:4001/config/hugo.json
//     endpoint : http://127.0.0.1:4001 or 127.0.0.1:4001 (for consul)
//     path     : /config/hugo.json or cut /v1/kv/ (for consul)
//     extension: json
// If extension is not set, defaultExtension will be returned.
func ParsePath(rawPath, defaultExtension, configProvider string) (endpoint, path, extension string, err error) {
	u, err := url.Parse(rawPath)
	if err != nil {
		return
	}

	if configProvider == "consul" {
		path = strings.Replace(u.RequestURI(), "/v1/kv/", "", -1)
		endpoint = u.Host
	} else {
		path = u.RequestURI()
		endpoint = strings.Replace(rawPath, path, "", 1)
	}
	extension = filepath.Ext(path)
	if len(extension) == 0 {
		extension = defaultExtension
		return
	}

	extension = strings.Split(extension[1:], "?")[0]
	return
}
