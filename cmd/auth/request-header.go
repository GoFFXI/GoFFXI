package auth

import (
	"fmt"

	"github.com/GoFFXI/login-server/internal/tools"
)

type RequestHeader struct {
	Username      string
	Password      string
	Command       uint8
	ClientVersion string
}

func NewRequestHeader(buffer []byte) (*RequestHeader, error) {
	// verify minimum size
	if len(buffer) < 0x66 { // At least up to version field
		return nil, fmt.Errorf("buffer too small: need at least %d bytes, got %d", 0x66, len(buffer))
	}

	request := &RequestHeader{
		Username:      string(buffer[0x09:0x19]),
		Password:      string(buffer[0x19:0x39]),
		Command:       buffer[0x39],
		ClientVersion: string(buffer[0x61:0x66]),
	}

	request.Username = tools.StripStringUnicode(request.Username)
	request.Password = tools.StripStringUnicode(request.Password)
	request.ClientVersion = tools.StripStringUnicode(request.ClientVersion)

	return request, nil
}
