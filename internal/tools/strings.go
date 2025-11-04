package tools

import (
	"fmt"
	"strings"
)

func StripStringUnicode(tmp string) string {
	unicodePosition := strings.Index(tmp, "\u0000")
	if unicodePosition != -1 {
		tmp = tmp[:unicodePosition]
	}

	return tmp
}

func GetIPAddressFromBuffer(buffer []byte, position int) string {
	octetOne := GetIntFromByteBuffer(buffer, position)
	octetTwo := GetIntFromByteBuffer(buffer, position+1)
	octetThree := GetIntFromByteBuffer(buffer, position+2)
	octetFour := GetIntFromByteBuffer(buffer, position+3)

	return fmt.Sprintf("%d.%d.%d.%d", octetOne, octetTwo, octetThree, octetFour)
}
