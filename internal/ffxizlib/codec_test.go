package ffxizlib

import (
	"bytes"
	"crypto/rand"
	"path/filepath"
	"runtime"
	"testing"
)

const testBufferSize = 4096

func TestCompressDecompressRoundTrip(t *testing.T) {
	codec := newTestCodec(t)
	if err := codec.EnsureInitialized(); err != nil {
		t.Fatalf("EnsureInitialized failed: %v", err)
	}

	randomData := make([]byte, 512)
	_, err := rand.Read(randomData)
	if err != nil {
		t.Fatalf("failed to generate random data: %v", err)
	}

	testCases := []struct {
		name string
		data []byte
	}{
		{name: "ShortASCII", data: []byte("Hello, Vana'diel!")},
		{name: "Zeroes", data: bytes.Repeat([]byte{0}, 128)},
		{name: "Incrementing", data: incrementingBytes(300)},
		{name: "Random", data: randomData},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			roundTripCompressDecompress(t, codec, tc.data)
		})
	}
}

func TestDecompressInvalidHeader(t *testing.T) {
	codec := newTestCodec(t)
	if err := codec.EnsureInitialized(); err != nil {
		t.Fatalf("EnsureInitialized failed: %v", err)
	}

	input := []byte("test payload")
	compBuf := make([]byte, testBufferSize)
	bitCount, err := codec.Compress(input, compBuf)
	if err != nil {
		t.Fatalf("Compress failed: %v", err)
	}

	compressedLen := CompressedSize(bitCount)
	corrupted := make([]byte, compressedLen)
	copy(corrupted, compBuf[:compressedLen])
	corrupted[0] = 0 // invalid header marker

	_, err = codec.Decompress(corrupted, bitCount, make([]byte, len(input)))
	if err == nil {
		t.Fatal("expected error when header is invalid, got nil")
	}
}

func roundTripCompressDecompress(t *testing.T, codec *Codec, data []byte) {
	t.Helper()

	compBuf := make([]byte, testBufferSize)
	bitCount, err := codec.Compress(data, compBuf)
	if err != nil {
		t.Fatalf("Compress failed: %v", err)
	}

	compressedLen := CompressedSize(bitCount)
	decompBuf := make([]byte, len(data)+16) // allow for padding

	size, err := codec.Decompress(compBuf[:compressedLen], bitCount, decompBuf)
	if err != nil {
		t.Fatalf("Decompress failed: %v", err)
	}

	if size != len(data) {
		t.Fatalf("unexpected decompressed size: got %d want %d", size, len(data))
	}

	if !bytes.Equal(data, decompBuf[:size]) {
		t.Fatalf("round trip mismatch:\n got: %x\nwant: %x", decompBuf[:size], data)
	}
}

func incrementingBytes(n int) []byte {
	buf := make([]byte, n)
	for i := 0; i < n; i++ {
		buf[i] = byte(i % 256)
	}
	return buf
}

func newTestCodec(t *testing.T) *Codec {
	t.Helper()

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}

	root := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
	resDir := filepath.Join(root, "resources")
	return NewCodec(resDir)
}
