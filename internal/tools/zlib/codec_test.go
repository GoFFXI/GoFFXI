package zlib

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
)

const (
	testSymbolA = byte('A')
	testSymbolB = byte('B')
)

func TestBytesToUint32(t *testing.T) {
	t.Run("valid data", func(t *testing.T) {
		data := []byte{1, 0, 0, 0, 2, 0, 0, 0}
		vals, err := bytesToUint32(data)
		if err != nil {
			t.Fatalf("bytesToUint32() error = %v", err)
		}

		expected := []uint32{1, 2}
		if !bytes.Equal(uint32SliceToBytes(vals), uint32SliceToBytes(expected)) {
			t.Fatalf("bytesToUint32() = %v, want %v", vals, expected)
		}
	})

	t.Run("invalid length", func(t *testing.T) {
		if _, err := bytesToUint32([]byte{1, 2, 3}); err == nil {
			t.Fatal("bytesToUint32() succeeded on malformed input")
		}
	})
}

func TestCompressSub(t *testing.T) {
	pattern := []byte{0b00000110, 0, 0, 0}
	out := make([]byte, 1)
	if err := compressSub(pattern, 0, 3, out); err != nil {
		t.Fatalf("compressSub() error = %v", err)
	}
	if got := out[0]; got != 0b00000110 {
		t.Fatalf("compressSub() wrote %08b, want %08b", got, 0b00000110)
	}

	t.Run("respects bit offset", func(t *testing.T) {
		out := make([]byte, 1)
		if err := compressSub([]byte{1, 0, 0, 0}, 2, 1, out); err != nil {
			t.Fatalf("compressSub() error = %v", err)
		}
		if got := out[0]; got != 0b00000100 {
			t.Fatalf("compressSub() wrote %08b, want %08b", got, 0b00000100)
		}
	})

	t.Run("errors on overflow", func(t *testing.T) {
		if err := compressSub(pattern, 0, 9, out); err == nil {
			t.Fatal("compressSub() did not report overflow")
		}
	})
}

func TestPopulateJumpTableErrors(t *testing.T) {
	tests := []struct {
		name string
		data []uint32
	}{
		{name: "empty table", data: nil},
		{name: "pointer out of range", data: []uint32{0x404}},
		{name: "root pointer missing", data: []uint32{0x10}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			codec := &FFXICodec{}
			if err := codec.populateJumpTable(tc.data); err == nil {
				t.Fatal("populateJumpTable() unexpectedly succeeded")
			}
		})
	}
}

func TestFFXICodecCompressDecompress(t *testing.T) {
	codec := newTestCodec(t)

	input := []byte{testSymbolA, testSymbolB, testSymbolA, testSymbolA}
	compressed := make([]byte, len(input)+2)
	bitCount, err := codec.Compress(input, compressed)
	if err != nil {
		t.Fatalf("Compress() error = %v", err)
	}

	if compressed[0] != 1 {
		t.Fatalf("Compress() header byte = %d, want 1", compressed[0])
	}

	payloadBits := bitCount - 8
	payloadBytes := int((payloadBits + 7) / 8)
	compressedPayload := compressed[:payloadBytes+1]

	output := make([]byte, len(input))
	written, err := codec.Decompress(compressedPayload, payloadBits, output)
	if err != nil {
		t.Fatalf("Decompress() error = %v", err)
	}
	if written != len(input) {
		t.Fatalf("Decompress() wrote %d bytes, want %d", written, len(input))
	}
	if !bytes.Equal(output[:written], input) {
		t.Fatalf("Decompress() = %v, want %v", output[:written], input)
	}
}

func TestFFXICodecCompressOverflowFallback(t *testing.T) {
	codec := newTestCodec(t)

	input := bytes.Repeat([]byte{testSymbolA}, 10)
	dst := make([]byte, 2) // Forces fallback path

	bitCount, err := codec.Compress(input, dst)
	if err != nil {
		t.Fatalf("Compress() error = %v", err)
	}

	if bitCount != uint32(len(input)) {
		t.Fatalf("Compress() fallback returned %d bits, want %d", bitCount, len(input))
	}

	if dst[0] != 0 {
		t.Fatalf("fallback output byte[0] = %d, want 0", dst[0])
	}
	if dst[1] != byte(len(input)) {
		t.Fatalf("fallback output byte[1] = %d, want %d", dst[1], len(input))
	}
}

func newTestCodec(t *testing.T) *FFXICodec {
	t.Helper()

	dir := t.TempDir()
	writeUint32Resource(t, dir, CompressFileName, testEncTable())
	writeUint32Resource(t, dir, DecompressFileName, testDecTable())

	codec := NewCodec(dir)
	if err := codec.EnsureInitialized(); err != nil {
		t.Fatalf("EnsureInitialized() error = %v", err)
	}
	return codec
}

func writeUint32Resource(t *testing.T, dir, name string, values []uint32) {
	t.Helper()

	data := make([]byte, len(values)*4)
	for i, v := range values {
		binary.LittleEndian.PutUint32(data[i*4:], v)
	}

	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("failed to write %s: %v", name, err)
	}
}

func testEncTable() []uint32 {
	const tableSize = 0x200
	table := make([]uint32, tableSize)

	setEncEntry(table, testSymbolA, 0, 1)
	setEncEntry(table, testSymbolB, 1, 1)

	return table
}

func setEncEntry(table []uint32, symbol byte, pattern, length uint32) {
	index := int(int8(symbol))
	table[0x80+index] = pattern
	table[0x180+index] = length
}

func testDecTable() []uint32 {
	const base = uint32(0x400)
	table := make([]uint32, 13)
	table[0] = base + 4   // root pointer
	table[1] = base + 5*4 // bit 0 -> symbol A node
	table[2] = base + 9*4 // bit 1 -> symbol B node
	table[8] = uint32(testSymbolA)
	table[12] = uint32(testSymbolB)
	return table
}

func uint32SliceToBytes(vals []uint32) []byte {
	buf := make([]byte, len(vals)*4)
	for i, v := range vals {
		binary.LittleEndian.PutUint32(buf[i*4:], v)
	}
	return buf
}
