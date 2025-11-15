package blowfish

import (
	"bytes"
	"crypto/md5"
	"encoding/binary"
	"encoding/hex"
	"testing"
)

func TestSetKeyFromStringAndGetters(t *testing.T) {
	const sessionKey = "ABCDEFGHIJKLMNOPQRST" // 20 bytes

	var bf Blowfish
	if err := bf.SetKeyFromString(sessionKey); err != nil {
		t.Fatalf("SetKeyFromString() error = %v", err)
	}

	expected := [5]uint32{
		binary.LittleEndian.Uint32([]byte("ABCD")),
		binary.LittleEndian.Uint32([]byte("EFGH")),
		binary.LittleEndian.Uint32([]byte("IJKL")),
		binary.LittleEndian.Uint32([]byte("MNOP")),
		binary.LittleEndian.Uint32([]byte("QRST")),
	}
	if bf.Key != expected {
		t.Fatalf("SetKeyFromString() = %#v, want %#v", bf.Key, expected)
	}

	if got := bf.GetKeyAsString(); got != sessionKey {
		t.Fatalf("GetKeyAsString() = %q, want %q", got, sessionKey)
	}

	keyBytes := bf.GetKeyBytes()
	if len(keyBytes) != KeySize {
		t.Fatalf("GetKeyBytes() len = %d, want %d", len(keyBytes), KeySize)
	}
	if !bytes.Equal(keyBytes, []byte(sessionKey)) {
		t.Fatalf("GetKeyBytes() = %v, want %v", keyBytes, []byte(sessionKey))
	}
}

func TestGetKeyAsStringEmptyAndPartial(t *testing.T) {
	var bf Blowfish
	if got := bf.GetKeyAsString(); got != "" {
		t.Fatalf("GetKeyAsString() = %q for zero key, want empty", got)
	}

	const sessionKey = "abcde"
	if err := bf.SetKeyFromString(sessionKey); err != nil {
		t.Fatalf("SetKeyFromString() error = %v", err)
	}

	expected := [5]uint32{
		0x64636261, // "abcd" little-endian
		0x00000065,
		0, 0, 0,
	}
	if bf.Key != expected {
		t.Fatalf("partial SetKeyFromString() = %#v, want %#v", bf.Key, expected)
	}

	if got := bf.GetKeyAsString(); got != sessionKey {
		t.Fatalf("GetKeyAsString() = %q, want %q", got, sessionKey)
	}

	keyBytes := bf.GetKeyBytes()
	wantBytes := append([]byte(sessionKey), make([]byte, KeySize-len(sessionKey))...)
	if !bytes.Equal(keyBytes, wantBytes) {
		t.Fatalf("GetKeyBytes() = %v, want %v", keyBytes, wantBytes)
	}
}

func TestSetKeyBytesMatchesString(t *testing.T) {
	raw := []byte("ABCDEFGHIJKLMNOPQRST")

	var fromBytes, fromString Blowfish
	fromBytes.SetKeyBytes(raw)
	if err := fromString.SetKeyFromString(string(raw)); err != nil {
		t.Fatalf("SetKeyFromString() error = %v", err)
	}

	if fromBytes.Key != fromString.Key {
		t.Fatalf("SetKeyBytes result %#v differs from string %#v", fromBytes.Key, fromString.Key)
	}
}

func TestInitBlowfishHashAndHex(t *testing.T) {
	var bf Blowfish
	if err := bf.SetKeyFromString("0123456789ABCDEF0123"); err != nil {
		t.Fatalf("SetKeyFromString() error = %v", err)
	}

	if err := bf.initBlowfish(); err != nil {
		t.Fatalf("initBlowfish() error = %v", err)
	}

	keyBytes := bf.GetKeyBytes()
	expectedHash := md5.Sum(keyBytes)
	for i := 0; i < len(expectedHash); i++ {
		if expectedHash[i] == 0 {
			for j := i; j < len(expectedHash); j++ {
				expectedHash[j] = 0
			}
			break
		}
	}

	if bf.Hash != expectedHash {
		t.Fatalf("Hash = %x, want %x", bf.Hash, expectedHash)
	}

	if bf.Cipher == nil {
		t.Fatal("Cipher is nil after init")
	}

	if got := bf.HashHex(); got != hex.EncodeToString(expectedHash[:]) {
		t.Fatalf("HashHex() = %q, want %q", got, hex.EncodeToString(expectedHash[:]))
	}
}

func TestIncrementKey(t *testing.T) {
	bf := mustNewBlowfish(t, "IncrementKeyExample")

	original := bf.Key
	originalHash := bf.Hash

	if err := bf.IncrementKey(); err != nil {
		t.Fatalf("IncrementKey() error = %v", err)
	}

	if bf.Key[4] != original[4]+2 {
		t.Fatalf("Key[4] = %d, want %d", bf.Key[4], original[4]+2)
	}

	if bytes.Equal(bf.Hash[:], originalHash[:]) {
		t.Fatal("Hash did not change after IncrementKey()")
	}
}

func TestEncryptDecryptECB(t *testing.T) {
	bf := mustNewBlowfish(t, "0123456789abcdef")

	plain := []byte("01234567abcdefgh")
	buf := append([]byte(nil), plain...)

	bf.EncryptECB(buf)
	if bytes.Equal(buf, plain) {
		t.Fatal("EncryptECB() left data unchanged")
	}

	bf.DecryptECB(buf)
	if !bytes.Equal(buf, plain) {
		t.Fatalf("DecryptECB() = %v, want %v", buf, plain)
	}
}

func TestEncryptDecryptPacket(t *testing.T) {
	bf := mustNewBlowfish(t, "packetKeyExample")

	header := []byte{0xAA, 0xBB, 0xCC, 0xDD}
	body := []byte("abcdefghijkl") // 12 bytes (not multiple of 8)
	packet := append(append([]byte{}, header...), body...)
	original := append([]byte{}, packet...)

	bf.EncryptPacket(packet, len(header))

	if !bytes.Equal(packet[:len(header)], header) {
		t.Fatal("EncryptPacket() modified header bytes")
	}

	if bytes.Equal(packet[len(header):len(header)+8], body[:8]) {
		t.Fatal("EncryptPacket() did not encrypt the first 8 bytes")
	}

	if !bytes.Equal(packet[len(packet)-4:], original[len(original)-4:]) {
		t.Fatal("EncryptPacket() should leave trailing partial block untouched")
	}

	bf.DecryptPacket(packet, len(header))
	if !bytes.Equal(packet, original) {
		t.Fatalf("DecryptPacket() = %v, want %v", packet, original)
	}
}

func TestNewFromKeyBytes(t *testing.T) {
	key := []byte("qrstuvwxABCDEFGHIJKLMNOP")
	bf, err := NewFromKeyBytes(key)
	if err != nil {
		t.Fatalf("NewFromKeyBytes() error = %v", err)
	}

	if got := bf.GetKeyAsString(); got != string(key[:KeySize]) {
		t.Fatalf("GetKeyAsString() = %q, want %q", got, string(key[:KeySize]))
	}
}

func mustNewBlowfish(t *testing.T, key string) *Blowfish {
	t.Helper()

	bf, err := NewBlowfish(key)
	if err != nil {
		t.Fatalf("NewBlowfish(%q) error = %v", key, err)
	}
	return bf
}
