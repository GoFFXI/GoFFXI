package auth

import (
	"fmt"
	"strconv"
	"strings"
)

type RequestHeader struct {
	Command     uint8    `json:"command"`
	Username    string   `json:"username"`
	Password    string   `json:"password"`
	NewPassword string   `json:"new_password"`
	OTP         string   `json:"otp"`
	Version     [3]uint8 `json:"version"`
}

func (rh *RequestHeader) ClientVersion() string {
	return fmt.Sprintf("%d.%d.%d", rh.Version[0], rh.Version[1], rh.Version[2])
}

func (rh *RequestHeader) ClientMajorVersion() uint8 {
	return rh.Version[0]
}

func (rh *RequestHeader) ClientMinorVersion() uint8 {
	return rh.Version[1]
}

func (rh *RequestHeader) ClientPatchVersion() uint8 {
	return rh.Version[2]
}

func (rh *RequestHeader) parseExpectedVersion(expectedVersion string) [3]uint8 {
	parts := strings.Split(expectedVersion, ".")
	if len(parts) == 0 {
		return [3]uint8{0, 0, 0}
	} else if len(parts) == 1 {
		parts = append(parts, "0", "0")
	} else if len(parts) == 2 {
		parts = append(parts, "0")
	} else {
		parts = parts[:3]
	}

	var version [3]uint8
	for i := range parts {
		tmp, _ := strconv.Atoi(parts[i])
		version[i] = uint8(tmp) //nolint:gosec // technically safe, version numbers should never be that large
	}

	return version
}

func (rh *RequestHeader) VersionMatches(expectedVersion string) bool {
	expected := rh.parseExpectedVersion(expectedVersion)

	return rh.Version == expected
}

func (rh *RequestHeader) VersionAtLeast(expectedVersion string) bool {
	expected := rh.parseExpectedVersion(expectedVersion)

	// compare major version
	if rh.Version[0] > expected[0] {
		return true
	} else if rh.Version[0] < expected[0] {
		return false
	}

	// major versions are equal, compare minor
	if rh.Version[1] > expected[1] {
		return true
	} else if rh.Version[1] < expected[1] {
		return false
	}

	// major and minor versions are equal, compare patch
	if rh.Version[2] >= expected[2] {
		return true
	}

	return false
}
