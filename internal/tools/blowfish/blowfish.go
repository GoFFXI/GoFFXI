package blowfish

import (
	"crypto/md5"
	"encoding/binary"
	"encoding/hex"
)

const (
	KeySize  = 20
	HashSize = md5.Size
)

type BlowfishStatus int

const (
	BlowfishWaiting BlowfishStatus = iota
	BlowfishSent
	BlowfishAccepted
	BlowfishPendingZone
)

const (
	pSize  = 18
	sSize  = 1024
	rounds = 16
)

type Blowfish struct {
	Key    [5]uint32
	Hash   [HashSize]byte
	P      [pSize]uint32
	S      [sSize]uint32
	Status BlowfishStatus
}

func NewBlowfish(sessionKey string) (*Blowfish, error) {
	return NewFromKeyBytes([]byte(sessionKey))
}

func NewFromKeyBytes(sessionKey []byte) (*Blowfish, error) {
	bf := &Blowfish{Status: BlowfishWaiting}
	bf.SetKeyBytes(sessionKey)
	if err := bf.initBlowfish(); err != nil {
		return nil, err
	}
	return bf, nil
}

func (bf *Blowfish) SetKeyFromString(sessionKey string) error {
	bf.SetKeyBytes([]byte(sessionKey))
	return nil
}

func (bf *Blowfish) SetKeyBytes(key []byte) {
	for i := range bf.Key {
		bf.Key[i] = 0
	}
	if len(key) == 0 {
		return
	}

	var keyBytes [KeySize]byte
	copy(keyBytes[:], key)

	for i := range bf.Key {
		offset := i * 4
		bf.Key[i] = binary.LittleEndian.Uint32(keyBytes[offset : offset+4])
	}
}

func (bf *Blowfish) GetKeyAsString() string {
	allZero := true
	for _, val := range bf.Key {
		if val != 0 {
			allZero = false
			break
		}
	}
	if allZero {
		return ""
	}

	keyBytes := make([]byte, KeySize)
	for i, val := range bf.Key {
		binary.LittleEndian.PutUint32(keyBytes[i*4:], val)
	}

	length := KeySize
	for i := KeySize - 1; i >= 0; i-- {
		if keyBytes[i] != 0 {
			length = i + 1
			break
		}
	}

	return string(keyBytes[:length])
}

func (bf *Blowfish) GetKeyBytes() []byte {
	keyBytes := make([]byte, KeySize)
	for i, val := range bf.Key {
		binary.LittleEndian.PutUint32(keyBytes[i*4:], val)
	}
	return keyBytes
}

func (bf *Blowfish) HashHex() string {
	return hex.EncodeToString(bf.Hash[:])
}

func (bf *Blowfish) initBlowfish() error {
	keyBytes := make([]byte, KeySize)
	for i, val := range bf.Key {
		binary.LittleEndian.PutUint32(keyBytes[i*4:], val)
	}

	bf.Hash = md5.Sum(keyBytes)

	for i := 0; i < HashSize; i++ {
		if bf.Hash[i] == 0 {
			for j := i; j < HashSize; j++ {
				bf.Hash[j] = 0
			}
			break
		}
	}

	bf.initializeSubkeys()
	return nil
}

func (bf *Blowfish) initializeSubkeys() {
	// Copy base P-array and S-boxes from the constant subKey dump
	for i := 0; i < pSize; i++ {
		start := i * 4
		bf.P[i] = binary.LittleEndian.Uint32(subKey[start : start+4])
	}

	offset := pSize * 4
	for i := 0; i < sSize; i++ {
		start := offset + i*4
		bf.S[i] = binary.LittleEndian.Uint32(subKey[start : start+4])
	}

	j := 0
	for i := 0; i < pSize; i++ {
		var data uint32
		for k := 0; k < 4; k++ {
			signed := uint32(int32(int8(bf.Hash[j])))
			data = (data << 8) | signed
			j++
			if j >= HashSize {
				j = 0
			}
		}
		bf.P[i] ^= data
	}

	datal, datar := uint32(0), uint32(0)
	for i := 0; i < pSize; i += 2 {
		datal, datar = bf.encipherBlock(datal, datar)
		bf.P[i] = datal
		bf.P[i+1] = datar
	}

	for i := 0; i < sSize; i += 2 {
		datal, datar = bf.encipherBlock(datal, datar)
		bf.S[i] = datal
		bf.S[i+1] = datar
	}
}

func (bf *Blowfish) IncrementKey() error {
	bf.Key[4] += 2
	return bf.initBlowfish()
}

func (bf *Blowfish) EncryptECB(data []byte) {
	for i := 0; i+7 < len(data); i += 8 {
		xl := binary.LittleEndian.Uint32(data[i:])
		xr := binary.LittleEndian.Uint32(data[i+4:])
		xl, xr = bf.encipherBlock(xl, xr)
		binary.LittleEndian.PutUint32(data[i:], xl)
		binary.LittleEndian.PutUint32(data[i+4:], xr)
	}
}

func (bf *Blowfish) DecryptECB(data []byte) {
	for i := 0; i+7 < len(data); i += 8 {
		xl := binary.LittleEndian.Uint32(data[i:])
		xr := binary.LittleEndian.Uint32(data[i+4:])
		xl, xr = bf.decipherBlock(xl, xr)
		binary.LittleEndian.PutUint32(data[i:], xl)
		binary.LittleEndian.PutUint32(data[i+4:], xr)
	}
}

func (bf *Blowfish) EncryptPacket(packet []byte, headerSize int) {
	if len(packet) <= headerSize {
		return
	}
	data := packet[headerSize:]
	blockCount := (len(data) / 4) &^ 1
	if blockCount > 0 {
		bf.EncryptECB(data[:blockCount*4])
	}
}

func (bf *Blowfish) DecryptPacket(packet []byte, headerSize int) {
	if len(packet) <= headerSize {
		return
	}
	data := packet[headerSize:]
	blockCount := (len(data) / 4) &^ 1
	if blockCount > 0 {
		bf.DecryptECB(data[:blockCount*4])
	}
}

func (bf *Blowfish) encipherBlock(xl, xr uint32) (uint32, uint32) {
	for i := 0; i < rounds; i++ {
		xl ^= bf.P[i]
		xr ^= tt(xl, &bf.S)
		xl, xr = xr, xl
	}
	xl, xr = xr, xl
	xr ^= bf.P[rounds]
	xl ^= bf.P[rounds+1]
	return xl, xr
}

func (bf *Blowfish) decipherBlock(xl, xr uint32) (uint32, uint32) {
	for i := rounds + 1; i > 1; i-- {
		xl ^= bf.P[i]
		xr ^= tt(xl, &bf.S)
		xl, xr = xr, xl
	}
	xl, xr = xr, xl
	xr ^= bf.P[1]
	xl ^= bf.P[0]
	return xl, xr
}

func tt(working uint32, S *[sSize]uint32) uint32 {
	return ((((*S)[256+((working>>8)&0xff)] & 1) ^ 32) + (((*S)[768+(working>>24)] & 1) ^ 32) + (*S)[512+((working>>16)&0xff)] + (*S)[working&0xff])
}
