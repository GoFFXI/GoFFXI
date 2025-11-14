package router

import (
	"crypto/md5"
	"encoding/binary"
	"fmt"

	"golang.org/x/crypto/blowfish"
)

type BlowfishStatus int

const (
	BlowfishWaiting BlowfishStatus = iota
	BlowfishSent
	BlowfishAccepted
	BlowfishPendingZone
)

type Blowfish struct {
	key    [5]uint32        // The raw key (20 bytes as 5 uint32s)
	hash   [16]byte         // MD5 hash of the key
	cipher *blowfish.Cipher // The actual Blowfish cipher
	status BlowfishStatus   // Current status
}

// NewBlowfish creates a new FFXI Blowfish instance from a session key string
func NewBlowfish(sessionKey string) (*Blowfish, error) {
	bf := &Blowfish{
		status: BlowfishWaiting,
	}

	// Initialize the key array
	// The session key from DB is varchar(16), but we need to handle it properly
	if err := bf.SetKeyFromString(sessionKey); err != nil {
		return nil, err
	}

	// Initialize the Blowfish cipher with the key
	if err := bf.initBlowfish(); err != nil {
		return nil, err
	}

	return bf, nil
}

// SetKeyFromString sets the key from a session key string
func (bf *Blowfish) SetKeyFromString(sessionKey string) error {
	// Clear the key first
	for i := range bf.key {
		bf.key[i] = 0
	}

	// Convert string to bytes
	keyBytes := []byte(sessionKey)

	// The key is stored as 5 uint32s (20 bytes total)
	// If the session key is shorter, it will be padded with zeros
	for i := 0; i < len(keyBytes) && i < 20; i += 4 {
		idx := i / 4
		remaining := len(keyBytes) - i

		if remaining >= 4 {
			bf.key[idx] = binary.LittleEndian.Uint32(keyBytes[i : i+4])
		} else {
			// Handle partial uint32 at the end
			partial := make([]byte, 4)
			copy(partial, keyBytes[i:])
			bf.key[idx] = binary.LittleEndian.Uint32(partial)
		}
	}

	return nil
}

// GetKeyAsString returns the key as a string for database storage
func (bf *Blowfish) GetKeyAsString() string {
	// Special case: if key is all zeros, return empty string
	allZero := true
	for _, val := range bf.key {
		if val != 0 {
			allZero = false
			break
		}
	}
	if allZero {
		return ""
	}

	keyBytes := make([]byte, 20)
	for i, val := range bf.key {
		binary.LittleEndian.PutUint32(keyBytes[i*4:], val)
	}

	// Find the actual length (trim trailing zeros)
	length := 20
	for i := 19; i >= 0; i-- {
		if keyBytes[i] != 0 {
			length = i + 1
			break
		}
	}

	return string(keyBytes[:length])
}

// GetKeyBytes returns a raw copy of the 20-byte key
func (bf *Blowfish) GetKeyBytes() []byte {
	keyBytes := make([]byte, 20)
	for i, val := range bf.key {
		binary.LittleEndian.PutUint32(keyBytes[i*4:], val)
	}

	return keyBytes
}

// initBlowfish initializes the Blowfish cipher (matches C++ initBlowfish)
func (bf *Blowfish) initBlowfish() error {
	// Create MD5 hash of the key (20 bytes)
	keyBytes := make([]byte, 20)
	for i, val := range bf.key {
		binary.LittleEndian.PutUint32(keyBytes[i*4:], val)
	}

	bf.hash = md5.Sum(keyBytes)

	// The C++ code zeroes out the hash after the first zero byte
	// This is unusual but we need to match it for compatibility
	for i := 0; i < 16; i++ {
		if bf.hash[i] == 0 {
			// Zero out the rest of the hash
			for j := i; j < 16; j++ {
				bf.hash[j] = 0
			}
			break
		}
	}

	// Create Blowfish cipher with the hash as key
	var err error
	bf.cipher, err = blowfish.NewCipher(bf.hash[:])
	if err != nil {
		return fmt.Errorf("failed to create Blowfish cipher: %w", err)
	}

	return nil
}

// IncrementKey increments the key for zone transitions (matches C++ incrementBlowfish)
func (bf *Blowfish) IncrementKey() error {
	// Increment the 5th uint32 by 2
	bf.key[4] += 2

	// Reinitialize the cipher with the new key
	return bf.initBlowfish()
}

// EncryptECB encrypts data using ECB mode (FFXI uses ECB, not CBC)
func (bf *Blowfish) EncryptECB(data []byte) {
	// ECB mode encrypts each 8-byte block independently
	for i := 0; i < len(data)-7; i += 8 {
		bf.cipher.Encrypt(data[i:i+8], data[i:i+8])
	}
}

// DecryptECB decrypts data using ECB mode
func (bf *Blowfish) DecryptECB(data []byte) {
	// ECB mode decrypts each 8-byte block independently
	for i := 0; i < len(data)-7; i += 8 {
		bf.cipher.Decrypt(data[i:i+8], data[i:i+8])
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
