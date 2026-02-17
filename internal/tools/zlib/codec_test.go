package zlib

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"os"
	"path/filepath"
	"runtime"
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

func TestFFXICodecMatchesReferenceCompressor(t *testing.T) {
	cases := []struct {
		name        string
		payloadFile string
		bits        uint32
		hexOutput   string
	}{
		{
			name:        "sample_payload",
			payloadFile: "sample_payload.bin",
			bits:        2701,
			hexOutput:   "0195EB3401F73A22A1351B054C7E83470011F9EF07DF084BA6934B64681C8A107A7FFA1E2809657008D44646DCCF2072FD256ABD72F785D4FCEE41C0A12D31204BC38F65B15DFFD2C2145DF2B47A7A1C251D823D2C5AFF3A5C52890418FB341E0632F7B414FD83067480A43F68F9091BC645A02F219E44808FB08223C3D7223C84264F4949DA0EF3188012F01516F58358990146BF40333E20A46C449E03BA3FAC5FC59EC0AEE7B1E03B91200C04A03261295FE88C832114FC06801C8D6E3428A1EA433910958F3958E265A4C7469031344B9D078CD8DF222D41EC34A647D2176569A23BB22559F2322E55D1941E666144EA258775397906E6FA841A1FC7A683624B0A31EA208871406EA2C31ED6220B3B90294A804437A8603626440A8DC88120E8411E267B16DBA402BDD8C724AC4A81E89153E2A08F3550940F49481266E413F972286751F805B046F2233D78A60F4403",
		},
		{
			name:        "sample_payload2",
			payloadFile: "sample_payload2.bin",
			bits:        5394,
			hexOutput:   "014EDFA006B268F5872640002A2A3D9213F63059925C3F3844621E0692878C0F841C18C3BAE8902709E5BF0CD046303A228086B00FB1537272831EF9884842C0A54383AFF52B12A84A50EA1D401014450320501E44F9071980F11D94228D11C42007D6F8C023DFFBDDE352C4658605A3F91BB31C048DD0F74C833FA5A6A7A7B18258CF1BDD88A587519192383DC88FDE175A99FB483FD895A30744530A31819C7E040C21B9CBC3000158F404005227556C61B6E88D649443F5A2CB789EF495D86F68967107858A1400D60C8DA831F50991805FA1206392C726BA61D52F6845F697B430F6258016432348B28F0312934481241772F52F40A0C7FC611C3D46B64694B02F67471D88ED1219D76140874E46E82E4426F4CAA157323FCB4A8A0D697B4EBE122F4D7DD82115E4038711F76387FB494A4684A58345197364619B7C9A686469F8175A9E92B24C3E267DCFC05ACF62C6E91BD44016ADFED004084045A54772C21E264B92EB078748CCC340F290F18190036358171DF224A1FC9701DA08464704D010F621764A4E6ED0231F114908B87468F0B57E450255094ABD030882A2680004CA8328FF200330BE8352A4318218E4C01A1F78E47BBF7B5C8AB8CCB060347F639683A011FA9E69F0A7D4F4F4345610EB79A31BB1F4302A5212A707F9D1FB422B731FE907BB72F480684A212690D38F80212477791820008B9E0040EAA48A2DCC16BD918C72A85E7419CF93BE12FB0DCD32EEA0509102C09AA11135A63E2112F02B14644CF2D84437ACFA05ADC8FE9216C6BE04D062680449F6714062922890E442AEFE0508F4983F8CA3C7C8D68812F6E5ECA803B15D22E33A0CE8D0C908DD85C8845E39F44AE6675949B1216DCFC957E2A5A90F3BA4827CE030E27EEC703F49C988B074B028638E2C6C934F138D2C0DFF42CB535296C9C7A4EF1958EB59CC00",
		},
	}

	resourcePath := resourcesDir(t)
	codec := NewCodec(resourcePath)
	if err := codec.EnsureInitialized(); err != nil {
		t.Fatalf("EnsureInitialized() error = %v", err)
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			payload := readIntegrationPayload(t, tc.payloadFile)

			compressed := make([]byte, len(payload)*2+64)
			bitCount, err := codec.Compress(payload, compressed)
			if err != nil {
				t.Fatalf("Compress() error = %v", err)
			}

			if bitCount != tc.bits {
				t.Fatalf("Compress() bitCount = %d, want %d", bitCount, tc.bits)
			}

			compressedBytes := compressed[:compressedSize(bitCount)]
			wantCompressed := decodeHex(t, tc.hexOutput)
			if !bytes.Equal(compressedBytes, wantCompressed) {
				t.Fatalf("Compress() output mismatch\n got: %X\nwant: %X", compressedBytes, wantCompressed)
			}

			decompressed := make([]byte, len(payload))
			written, err := codec.Decompress(compressedBytes, bitCount, decompressed)
			if err != nil {
				t.Fatalf("Decompress() error = %v", err)
			}

			if !bytes.Equal(decompressed[:written], payload) {
				t.Fatalf("Decompress() = %X, want %X", decompressed[:written], payload)
			}
		})
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

func decodeHex(t *testing.T, hexStr string) []byte {
	t.Helper()

	data, err := hex.DecodeString(hexStr)
	if err != nil {
		t.Fatalf("failed to decode hex reference: %v", err)
	}
	return data
}

func readIntegrationPayload(t *testing.T, name string) []byte {
	t.Helper()

	path := filepath.Join(zlibTestDir(t), "testdata", name)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read %s: %v", path, err)
	}
	return data
}

func resourcesDir(t *testing.T) string {
	t.Helper()
	root := filepath.Clean(filepath.Join(zlibTestDir(t), "..", "..", ".."))
	return filepath.Join(root, "resources")
}

func zlibTestDir(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("unable to determine test file directory")
	}
	return filepath.Dir(filename)
}
