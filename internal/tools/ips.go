package tools

import (
	"encoding/binary"
	"net"
)

func StringToIP(ipStr string) uint32 {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return 0
	}

	ip = ip.To4()
	if ip == nil {
		return 0
	}

	return binary.BigEndian.Uint32(ip)
}

func IPToString(ip uint32) string {
	bytes := make([]byte, 4)
	binary.BigEndian.PutUint32(bytes, ip)

	return net.IP(bytes).String()
}
