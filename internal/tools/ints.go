package tools

import "math/big"

func GetIntFromByteBuffer(buffer []byte, position int) int {
	if (position + 1) > len(buffer) {
		return 0
	}

	return int(big.NewInt(0).SetBytes(buffer[position : position+1]).Int64())
}
