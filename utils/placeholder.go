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

	ulabel := fmt.Sprintf("%v_%v", prefix, strings.ToUpper(label))
	str = strings.Replace(str, fmt.Sprintf("${%v}", ulabel), new, -1)

	eulabel := fmt.Sprintf("URLENCODE_%v", ulabel)
	str = strings.Replace(str, fmt.Sprintf("${%v}", eulabel), url.QueryEscape(new), -1)

	culabel := fmt.Sprintf("CUT_AFTER_LAST_COLON_%v", ulabel)
	str = strings.Replace(str, fmt.Sprintf("${%v}", culabel), trimStringFromSymbol(new, ":"), -1)

	cuelabel := fmt.Sprintf("CUT_AFTER_LAST_COLON_%v", eulabel)
	str = strings.Replace(str, fmt.Sprintf("${%v}", cuelabel), url.QueryEscape(trimStringFromSymbol(new, ":")), -1)

	return str
}

func trimStringFromSymbol(s string, symbol string) string {
	if idx := strings.LastIndex(s, symbol); idx != -1 {
		return s[:idx]
	}
	return s
}
