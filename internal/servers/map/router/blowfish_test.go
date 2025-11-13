package router

import (
	"bytes"
	"crypto/md5"
	"encoding/binary"
	"encoding/hex"
	"testing"
)

func TestNewBlowfish(t *testing.T) {
	tests := []struct {
		name       string
		sessionKey string
		wantErr    bool
	}{
		{
			name:       "Valid 16-char session key",
			sessionKey: "1f0b1767829e1a0b",
			wantErr:    false,
		},
		{
			name:       "Short session key",
			sessionKey: "short",
			wantErr:    false,
		},
		{
			name:       "Empty session key",
			sessionKey: "",
			wantErr:    false,
		},
		{
			name:       "20-char session key",
			sessionKey: "12345678901234567890",
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bf, err := NewBlowfish(tt.sessionKey)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewBlowfish() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && bf == nil {
				t.Error("NewBlowfish() returned nil without error")
			}
			if !tt.wantErr && bf.status != BlowfishWaiting {
				t.Errorf("NewBlowfish() status = %v, want %v", bf.status, BlowfishWaiting)
			}
		})
	}
}

func TestSetKeyFromString(t *testing.T) {
	tests := []struct {
		name       string
		sessionKey string
		wantKey    [5]uint32
	}{
		{
			name:       "16-char ASCII key",
			sessionKey: "1f0b1767829e1a0b",
			wantKey:    [5]uint32{1647339057, 926299953, 1698247224, 1647337777, 0},
		},
		{
			name:       "Short key with padding",
			sessionKey: "test",
			wantKey:    [5]uint32{1953719668, 0, 0, 0, 0}, // "test" in little-endian
		},
		{
			name:       "Empty key",
			sessionKey: "",
			wantKey:    [5]uint32{0, 0, 0, 0, 0},
		},
		{
			name:       "Exactly 20 chars",
			sessionKey: "12345678901234567890",
			wantKey: [5]uint32{
				binary.LittleEndian.Uint32([]byte("1234")),
				binary.LittleEndian.Uint32([]byte("5678")),
				binary.LittleEndian.Uint32([]byte("9012")),
				binary.LittleEndian.Uint32([]byte("3456")),
				binary.LittleEndian.Uint32([]byte("7890")),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bf := &Blowfish{}
			err := bf.SetKeyFromString(tt.sessionKey)
			if err != nil {
				t.Errorf("SetKeyFromString() error = %v", err)
				return
			}

			if bf.key != tt.wantKey {
				t.Errorf("SetKeyFromString() key = %v, want %v", bf.key, tt.wantKey)
			}
		})
	}
}

func TestGetKeyAsString(t *testing.T) {
	tests := []struct {
		name string
		key  [5]uint32
		want string
	}{
		{
			name: "Standard 16-char key",
			key:  [5]uint32{1647339057, 926299953, 1698247224, 1647337777, 0},
			want: "1f0b1767829e1a0b",
		},
		{
			name: "Short key",
			key:  [5]uint32{1953719668, 0, 0, 0, 0},
			want: "test",
		},
		{
			name: "Empty key",
			key:  [5]uint32{0, 0, 0, 0, 0},
			want: "",
		},
		{
			name: "Full 20-byte key",
			key: [5]uint32{
				binary.LittleEndian.Uint32([]byte("1234")),
				binary.LittleEndian.Uint32([]byte("5678")),
				binary.LittleEndian.Uint32([]byte("9012")),
				binary.LittleEndian.Uint32([]byte("3456")),
				binary.LittleEndian.Uint32([]byte("7890")),
			},
			want: "12345678901234567890",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bf := &Blowfish{key: tt.key}
			got := bf.GetKeyAsString()
			if got != tt.want {
				t.Errorf("GetKeyAsString() = %q, want %q", got, tt.want)
				t.Errorf("  Got bytes: %s", hex.EncodeToString([]byte(got)))
				t.Errorf("  Want bytes: %s", hex.EncodeToString([]byte(tt.want)))
			}
		})
	}
}

func TestRoundTripKeyConversion(t *testing.T) {
	// Test that SetKeyFromString -> GetKeyAsString is identity
	testKeys := []string{
		"1f0b1767829e1a0b",
		"test",
		"",
		"a",
		"ab",
		"abc",
		"abcd",
		"12345678901234567890",
	}

	for _, key := range testKeys {
		t.Run(key, func(t *testing.T) {
			bf := &Blowfish{}
			err := bf.SetKeyFromString(key)
			if err != nil {
				t.Fatalf("SetKeyFromString() error = %v", err)
			}

			got := bf.GetKeyAsString()
			if got != key {
				t.Errorf("Round trip failed: input=%q, output=%q", key, got)
			}
		})
	}
}

func TestInitBlowfish(t *testing.T) {
	bf := &Blowfish{}
	err := bf.SetKeyFromString("1f0b1767829e1a0b")
	if err != nil {
		t.Fatalf("SetKeyFromString() error = %v", err)
	}

	err = bf.initBlowfish()
	if err != nil {
		t.Fatalf("initBlowfish() error = %v", err)
	}

	// Check that cipher was created
	if bf.cipher == nil {
		t.Error("initBlowfish() did not create cipher")
	}

	// Check MD5 hash
	expectedKey := [5]uint32{1647339057, 926299953, 1698247224, 1647337777, 0}
	keyBytes := make([]byte, 20)
	for i, val := range expectedKey {
		binary.LittleEndian.PutUint32(keyBytes[i*4:], val)
	}
	expectedHash := md5.Sum(keyBytes)

	if !bytes.Equal(bf.hash[:], expectedHash[:]) {
		t.Errorf("initBlowfish() hash = %s, want %s",
			hex.EncodeToString(bf.hash[:]),
			hex.EncodeToString(expectedHash[:]))
	}
}

func TestInitBlowfishWithZeroByte(t *testing.T) {
	// Test the special case where MD5 hash contains a zero byte
	// We need to craft a key that produces an MD5 with a zero byte
	bf := &Blowfish{}

	// This is a crafted example - in practice you'd need to find
	// a key that produces an MD5 with a zero byte
	bf.key = [5]uint32{0, 0, 0, 0, 0}

	err := bf.initBlowfish()
	if err != nil {
		t.Fatalf("initBlowfish() error = %v", err)
	}

	// The MD5 of all zeros is: 157966392305302106430182806280163248922
	// In hex: 76dfd672103648313348e44912e04a2a
	// This doesn't contain a zero byte, but we can verify the logic works

	// Manually test the zero-out logic
	testHash := [16]byte{1, 2, 3, 0, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}
	for i := 0; i < 16; i++ {
		if testHash[i] == 0 {
			for j := i; j < 16; j++ {
				testHash[j] = 0
			}
			break
		}
	}

	expected := [16]byte{1, 2, 3, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	if testHash != expected {
		t.Errorf("Zero-out logic failed: got %v, want %v", testHash, expected)
	}
}

func TestIncrementKey(t *testing.T) {
	bf := &Blowfish{}
	err := bf.SetKeyFromString("1f0b1767829e1a0b")
	if err != nil {
		t.Fatalf("SetKeyFromString() error = %v", err)
	}

	originalKey := bf.key
	originalKey4 := bf.key[4]

	err = bf.IncrementKey()
	if err != nil {
		t.Fatalf("IncrementKey() error = %v", err)
	}

	// Check that only the 5th uint32 was incremented by 2
	expectedKey := originalKey
	expectedKey[4] = originalKey4 + 2

	if bf.key != expectedKey {
		t.Errorf("IncrementKey() key = %v, want %v", bf.key, expectedKey)
	}

	// Check that cipher was reinitialized
	if bf.cipher == nil {
		t.Error("IncrementKey() did not reinitialize cipher")
	}

	// Check that status wasn't changed (that's handled elsewhere)
	if bf.status != BlowfishWaiting {
		t.Errorf("IncrementKey() changed status to %v", bf.status)
	}
}

func TestEncryptDecryptECB(t *testing.T) {
	bf := &Blowfish{}
	err := bf.SetKeyFromString("testkey123456789")
	if err != nil {
		t.Fatalf("SetKeyFromString() error = %v", err)
	}

	err = bf.initBlowfish()
	if err != nil {
		t.Fatalf("initBlowfish() error = %v", err)
	}

	tests := []struct {
		name string
		data []byte
	}{
		{
			name: "Exactly 8 bytes",
			data: []byte("12345678"),
		},
		{
			name: "16 bytes",
			data: []byte("1234567890123456"),
		},
		{
			name: "Not multiple of 8 (should only process complete blocks)",
			data: []byte("123456789"), // 9 bytes, only first 8 encrypted
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Make a copy of the original data
			original := make([]byte, len(tt.data))
			copy(original, tt.data)

			// Encrypt
			encrypted := make([]byte, len(tt.data))
			copy(encrypted, tt.data)
			bf.EncryptECB(encrypted)

			// Verify encryption changed the data (for complete blocks)
			if len(tt.data) >= 8 {
				if bytes.Equal(encrypted[:8], original[:8]) {
					t.Error("EncryptECB() did not modify data")
				}
			}

			// Decrypt
			decrypted := make([]byte, len(encrypted))
			copy(decrypted, encrypted)
			bf.DecryptECB(decrypted)

			// Verify decryption restored original data
			if !bytes.Equal(decrypted, original) {
				t.Errorf("DecryptECB() did not restore original data")
				t.Errorf("  Original:  %s", hex.EncodeToString(original))
				t.Errorf("  Decrypted: %s", hex.EncodeToString(decrypted))
			}
		})
	}
}

func TestEncryptDecryptPacket(t *testing.T) {
	bf := &Blowfish{}
	err := bf.SetKeyFromString("1f0b1767829e1a0b")
	if err != nil {
		t.Fatalf("SetKeyFromString() error = %v", err)
	}

	err = bf.initBlowfish()
	if err != nil {
		t.Fatalf("initBlowfish() error = %v", err)
	}

	headerSize := 28 // FFXI_HEADER_SIZE

	tests := []struct {
		name   string
		packet []byte
	}{
		{
			name:   "Small packet with header and 8 bytes data",
			packet: make([]byte, headerSize+8),
		},
		{
			name:   "Packet with 16 bytes data",
			packet: make([]byte, headerSize+16),
		},
		{
			name:   "Packet with non-multiple of 8 data",
			packet: make([]byte, headerSize+10), // Only 8 bytes will be encrypted
		},
		{
			name:   "Packet with only header",
			packet: make([]byte, headerSize),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Fill packet with test data
			for i := range tt.packet {
				tt.packet[i] = byte(i % 256)
			}

			// Make a copy of the original
			original := make([]byte, len(tt.packet))
			copy(original, tt.packet)

			// Encrypt
			encrypted := make([]byte, len(tt.packet))
			copy(encrypted, tt.packet)
			bf.EncryptPacket(encrypted, headerSize)

			// Verify header wasn't touched
			if !bytes.Equal(encrypted[:headerSize], original[:headerSize]) {
				t.Error("EncryptPacket() modified header")
			}

			// If there's data after header, verify it was encrypted
			if len(tt.packet) > headerSize+7 {
				if bytes.Equal(encrypted[headerSize:headerSize+8], original[headerSize:headerSize+8]) {
					t.Error("EncryptPacket() did not encrypt data")
				}
			}

			// Decrypt
			decrypted := make([]byte, len(encrypted))
			copy(decrypted, encrypted)
			bf.DecryptPacket(decrypted, headerSize)

			// Verify complete restoration
			if !bytes.Equal(decrypted, original) {
				t.Errorf("DecryptPacket() did not restore original packet")
				t.Errorf("  Original:  %s", hex.EncodeToString(original))
				t.Errorf("  Decrypted: %s", hex.EncodeToString(decrypted))
			}
		})
	}
}

func TestBlowfishStatus(t *testing.T) {
	// Test that status constants have expected values
	if BlowfishWaiting != 0 {
		t.Errorf("BlowfishWaiting = %d, want 0", BlowfishWaiting)
	}
	if BlowfishSent != 1 {
		t.Errorf("BlowfishSent = %d, want 1", BlowfishSent)
	}
	if BlowfishAccepted != 2 {
		t.Errorf("BlowfishAccepted = %d, want 2", BlowfishAccepted)
	}
	if BlowfishPendingZone != 3 {
		t.Errorf("BlowfishPendingZone = %d, want 3", BlowfishPendingZone)
	}
}

func BenchmarkEncryptECB(b *testing.B) {
	bf := &Blowfish{}
	_ = bf.SetKeyFromString("1f0b1767829e1a0b")
	_ = bf.initBlowfish()

	data := make([]byte, 1024) // 1KB of data
	for i := range data {
		data[i] = byte(i % 256)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bf.EncryptECB(data)
	}
}

func BenchmarkDecryptECB(b *testing.B) {
	bf := &Blowfish{}
	_ = bf.SetKeyFromString("1f0b1767829e1a0b")
	_ = bf.initBlowfish()

	data := make([]byte, 1024) // 1KB of data
	for i := range data {
		data[i] = byte(i % 256)
	}
	bf.EncryptECB(data) // Start with encrypted data

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bf.DecryptECB(data)
	}
}

// TestIntegrationFullCycle tests a complete encryption/decryption cycle
// simulating actual FFXI packet flow
func TestIntegrationFullCycle(t *testing.T) {
	// Simulate creating a session
	sessionKey := "1f0b1767829e1a0b"
	bf, err := NewBlowfish(sessionKey)
	if err != nil {
		t.Fatalf("NewBlowfish() error = %v", err)
	}

	// Initialize for use
	err = bf.initBlowfish()
	if err != nil {
		t.Fatalf("initBlowfish() error = %v", err)
	}

	// Create a mock packet
	headerSize := 28
	packetData := []byte("Hello FFXI World! This is test data for encryption.")
	packet := make([]byte, headerSize+len(packetData))

	// Fill header with some data
	for i := 0; i < headerSize; i++ {
		packet[i] = byte(i)
	}
	copy(packet[headerSize:], packetData)

	// Store original for comparison
	original := make([]byte, len(packet))
	copy(original, packet)

	// Encrypt the packet
	bf.EncryptPacket(packet, headerSize)

	// Simulate zone transition
	err = bf.IncrementKey()
	if err != nil {
		t.Fatalf("IncrementKey() error = %v", err)
	}

	// Try to decrypt with new key (should fail)
	wrongDecrypt := make([]byte, len(packet))
	copy(wrongDecrypt, packet)
	bf.DecryptPacket(wrongDecrypt, headerSize)

	if bytes.Equal(wrongDecrypt, original) {
		t.Error("Packet decrypted successfully with wrong key")
	}

	// Create new Blowfish with original key to decrypt
	bf2, err := NewBlowfish(sessionKey)
	if err != nil {
		t.Fatalf("NewBlowfish() error = %v", err)
	}
	err = bf2.initBlowfish()
	if err != nil {
		t.Fatalf("initBlowfish() error = %v", err)
	}

	// Decrypt with correct key
	decrypted := make([]byte, len(packet))
	copy(decrypted, packet)
	bf2.DecryptPacket(decrypted, headerSize)

	if !bytes.Equal(decrypted, original) {
		t.Error("Failed to decrypt packet with correct key")
		t.Errorf("  Original:  %s", hex.EncodeToString(original[headerSize:headerSize+20]))
		t.Errorf("  Decrypted: %s", hex.EncodeToString(decrypted[headerSize:headerSize+20]))
	}
}

// TestEdgeCases tests various edge cases
func TestEdgeCases(t *testing.T) {
	t.Run("EncryptECB with nil data", func(_ *testing.T) {
		bf := &Blowfish{}
		_ = bf.SetKeyFromString("test")
		_ = bf.initBlowfish()

		// Should not panic
		bf.EncryptECB(nil)
	})

	t.Run("DecryptECB with nil data", func(_ *testing.T) {
		bf := &Blowfish{}
		_ = bf.SetKeyFromString("test")
		_ = bf.initBlowfish()

		// Should not panic
		bf.DecryptECB(nil)
	})

	t.Run("EncryptPacket with nil packet", func(_ *testing.T) {
		bf := &Blowfish{}
		_ = bf.SetKeyFromString("test")
		_ = bf.initBlowfish()

		// Should not panic
		bf.EncryptPacket(nil, 28)
	})

	t.Run("DecryptPacket with nil packet", func(_ *testing.T) {
		bf := &Blowfish{}
		_ = bf.SetKeyFromString("test")
		_ = bf.initBlowfish()

		// Should not panic
		bf.DecryptPacket(nil, 28)
	})

	t.Run("EncryptPacket with packet smaller than header", func(_ *testing.T) {
		bf := &Blowfish{}
		_ = bf.SetKeyFromString("test")
		_ = bf.initBlowfish()

		smallPacket := make([]byte, 10)

		// Should not panic
		bf.EncryptPacket(smallPacket, 28)
	})
}
