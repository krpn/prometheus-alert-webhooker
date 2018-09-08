package utils

import (
	"fmt"
	"net/url"
	"strings"
)

// ReplacePlaceholders replaces placeholders for given params:
//     s      - destination string
//     prefix - type of label: LABEL or ANNOTATION
//     label  - label/annotation name
//     new    - new value
func ReplacePlaceholders(str, prefix, label, new string) string {

	for _, m := range modificators {
		mask := "${%v_%v}"
		if m.mod != "" {
			mask = fmt.Sprintf("${%v_%%v_%%v}", m.mod)
		}
		str = strings.Replace(
			str,
			fmt.Sprintf(mask, prefix, strings.ToUpper(label)),
			m.modFunc(new),
			-1,
		)
	}

	return str
}

var modificators = []struct {
	mod     string
	modFunc func(string) string
}{
	{
		mod: "",
		modFunc: func(s string) string {
			return s
		},
	},
	{
		mod: "URLENCODE",
		modFunc: func(s string) string {
			return url.QueryEscape(s)
		},
	},
	{
		mod: "CUT_AFTER_LAST_COLON",
		modFunc: func(s string) string {
			if idx := strings.LastIndex(s, ":"); idx != -1 {
				return s[:idx]
			}
			return s
		},
	},
	{
		mod: "JSON_ESCAPE",
		modFunc: func(s string) string {
			s = strings.Replace(s, `\`, `\\`, -1)
			s = strings.Replace(s, `"`, `\"`, -1)
			return s
		},
	},
}
