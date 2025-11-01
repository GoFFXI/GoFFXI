package tools

import (
	"bytes"
)

func BytesToString(data []byte, start, length int) string {
	if (start < 0) || (length < 0) || ((start + length) > len(data)) {
		return ""
	}

	return string(bytes.Trim(data[start:start+length], "\x00"))
}
