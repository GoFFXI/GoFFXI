package tools

import "math/big"

func GetIntFromByteBuffer(buffer []byte, position int) int {
	if (position + 1) > len(buffer) {
		return 0
	}

	return int(big.NewInt(0).SetBytes(buffer[position : position+1]).Int64())
}

func GetUint32FromByteBuffer(buffer []byte, position int) uint32 {
	end := position + 4
	if end > len(buffer) {
		return 0
	}

	return uint32(big.NewInt(0).SetBytes(buffer[position:end]).Int64())
}
