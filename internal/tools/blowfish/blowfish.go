package blowfish

import (
	"crypto/md5"
	"encoding/binary"
	"encoding/hex"
	"fmt"

	"golang.org/x/crypto/blowfish"
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

type Blowfish struct {
	Key    [5]uint32        // The raw key (20 bytes as 5 uint32s)
	Hash   [HashSize]byte   // MD5 hash of the key
	Cipher *blowfish.Cipher // The actual Blowfish cipher
	Status BlowfishStatus   // Current status
}

// NewBlowfish creates a new FFXI Blowfish instance from a session key string
func NewBlowfish(sessionKey string) (*Blowfish, error) {
	return NewFromKeyBytes([]byte(sessionKey))
}

// NewFromKeyBytes creates a new Blowfish instance from a raw 20-byte key.
func NewFromKeyBytes(sessionKey []byte) (*Blowfish, error) {
	bf := &Blowfish{Status: BlowfishWaiting}

	bf.SetKeyBytes(sessionKey)

	// Initialize the Blowfish cipher with the key
	if err := bf.initBlowfish(); err != nil {
		return nil, err
	}

	return bf, nil
}

// SetKeyFromString sets the key from a session key string
func (bf *Blowfish) SetKeyFromString(sessionKey string) error {
	bf.SetKeyBytes([]byte(sessionKey))
	return nil
}

// SetKeyBytes sets the raw key data directly (matches memcpy semantics in LSB).
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

// GetKeyAsString returns the key as a string for database storage
func (bf *Blowfish) GetKeyAsString() string {
	// Special case: if key is all zeros, return empty string
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

	// Find the actual length (trim trailing zeros)
	length := KeySize
	for i := KeySize - 1; i >= 0; i-- {
		if keyBytes[i] != 0 {
			length = i + 1
			break
		}
	}

	return string(keyBytes[:length])
}

// GetKeyBytes returns a raw copy of the 20-byte key
func (bf *Blowfish) GetKeyBytes() []byte {
	keyBytes := make([]byte, KeySize)
	for i, val := range bf.Key {
		binary.LittleEndian.PutUint32(keyBytes[i*4:], val)
	}

	return keyBytes
}

// HashHex returns the current MD5 hash representation as hex.
func (bf *Blowfish) HashHex() string {
	return hex.EncodeToString(bf.Hash[:])
}

// initBlowfish initializes the Blowfish cipher (matches C++ initBlowfish)
func (bf *Blowfish) initBlowfish() error {
	// Create MD5 hash of the key (20 bytes)
	keyBytes := make([]byte, KeySize)
	for i, val := range bf.Key {
		binary.LittleEndian.PutUint32(keyBytes[i*4:], val)
	}

	bf.Hash = md5.Sum(keyBytes)

	// The C++ code zeroes out the hash after the first zero byte
	// This is unusual but we need to match it for compatibility
	for i := 0; i < HashSize; i++ {
		if bf.Hash[i] == 0 {
			// Zero out the rest of the hash
			for j := i; j < HashSize; j++ {
				bf.Hash[j] = 0
			}
			break
		}
	}

	// Create Blowfish cipher with the hash as key
	var err error
	bf.Cipher, err = blowfish.NewCipher(bf.Hash[:])
	if err != nil {
		return fmt.Errorf("failed to create Blowfish cipher: %w", err)
	}

	return nil
}

// IncrementKey increments the key for zone transitions (matches C++ incrementBlowfish)
func (bf *Blowfish) IncrementKey() error {
	// Increment the 5th uint32 by 2
	bf.Key[4] += 2

	// Reinitialize the cipher with the new key
	return bf.initBlowfish()
}

// EncryptECB encrypts data using ECB mode (FFXI uses ECB, not CBC)
func (bf *Blowfish) EncryptECB(data []byte) {
	// ECB mode encrypts each 8-byte block independently
	for i := 0; i < len(data)-7; i += 8 {
		bf.Cipher.Encrypt(data[i:i+8], data[i:i+8])
	}
}

// DecryptECB decrypts data using ECB mode
func (bf *Blowfish) DecryptECB(data []byte) {
	// ECB mode decrypts each 8-byte block independently
	for i := 0; i < len(data)-7; i += 8 {
		bf.Cipher.Decrypt(data[i:i+8], data[i:i+8])
	}
}

// EncryptPacket encrypts an FFXI packet (after the header)
func (bf *Blowfish) EncryptPacket(packet []byte, headerSize int) {
	if len(packet) <= headerSize {
		return
	}

	// Only encrypt data after the header
	data := packet[headerSize:]

	// FFXI encrypts in pairs of uint32s (8 bytes)
	// Calculate the number of 8-byte blocks
	blockCount := (len(data) / 4) & ^1 // Round down to even number of uint32s

	if blockCount > 0 {
		bf.EncryptECB(data[:blockCount*4])
	}
}

// DecryptPacket decrypts an FFXI packet (after the header)
func (bf *Blowfish) DecryptPacket(packet []byte, headerSize int) {
	if len(packet) <= headerSize {
		return
	}

	// Only decrypt data after the header
	data := packet[headerSize:]

	// FFXI decrypts in pairs of uint32s (8 bytes)
	// Calculate the number of 8-byte blocks
	blockCount := (len(data) / 4) & ^1 // Round down to even number of uint32s

	if blockCount > 0 {
		bf.DecryptECB(data[:blockCount*4])
	}
}
