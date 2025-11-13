package mappackets

import (
	"bytes"
	"crypto/md5"
)

const (
	MD5ChecksumSize = 16 // 16 bytes
)

func PerformPacketChecksum(data []byte) bool {
	// the checksum is located at the end of the packet and is 16 bytes long
	checksumStart := len(data) - MD5ChecksumSize

	// There must be at least enough data for header + checksum
	if len(data) < (HeaderSize + MD5ChecksumSize) {
		return false
	}

	// Calculate MD5 hash of the data portion (without the header)
	hash := md5.Sum(data[HeaderSize:checksumStart])

	// Return whether they match
	return bytes.Equal(hash[:], data[checksumStart:])
}
