package tools

import "strings"

func StripStringUnicode(tmp string) string {
	unicodePosition := strings.Index(tmp, "\u0000")
	if unicodePosition != -1 {
		tmp = tmp[:unicodePosition]
	}

	return tmp
}
