package zlib

import (
	"encoding/binary"
	"errors"
	"fmt"
	"unsafe"
)

func compressedSize(bits uint32) int {
	return int((bits + 7) / 8)
}

func compressSub(bitPattern []byte, read, elem uint32, out []byte) error {
	if compressedSize(elem) > 4 {
		return fmt.Errorf("bit pattern requires %d bytes (>4)", compressedSize(elem))
	}

	if compressedSize(read+elem) > len(out) {
		return errors.New("compression output exceeded buffer")
	}

	for i := 0; i < int(elem); i++ {
		shift := (read + uint32(i)) & 7
		index := (read + uint32(i)) / 8

		invMask := ^(uint8(1) << shift)
		bit := (bitPattern[i/8] >> (i & 7)) & 1
		out[index] = (invMask & out[index]) + (bit << shift)
	}
	return nil
}

func jumpEntryAt(entry *jumpEntry, offset int) *jumpEntry {
	if entry == nil {
		return nil
	}
	ptr := unsafe.Add(unsafe.Pointer(entry), uintptr(offset)*unsafe.Sizeof(jumpEntry{}))
	return (*jumpEntry)(ptr)
}

func bytesToUint32(data []byte) ([]uint32, error) {
	if len(data)%4 != 0 {
		return nil, fmt.Errorf("malformed resource length %d", len(data))
	}

	vals := make([]uint32, len(data)/4)
	for i := 0; i < len(vals); i++ {
		vals[i] = binary.LittleEndian.Uint32(data[i*4:])
	}

	return vals, nil
}
