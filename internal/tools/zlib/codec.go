package zlib

import (
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"unsafe"
)

type jumpEntry struct {
	ptr      unsafe.Pointer
	value    byte
	hasValue bool
}

// Codec encapsulates the LandSandBoat compression tables and provides
// methods that match the original FFXI behaviour.
const (
	CompressFileName   = "compress.dat"
	DecompressFileName = "decompress.dat"
)

type FFXICodec struct {
	resourcePath string

	initOnce sync.Once
	initErr  error

	encTable  []uint32
	jumpTable []jumpEntry
	jumpRoot  *jumpEntry
}

// NewCodec creates a codec instance that loads compress.dat and decompress.dat
// from the given resourcePath directory.
func NewCodec(resourcePath string) *FFXICodec {
	return &FFXICodec{
		resourcePath: resourcePath,
	}
}

// EnsureInitialized loads the compression tables if they haven't been loaded yet.
func (c *FFXICodec) EnsureInitialized() error {
	c.initOnce.Do(func() {
		c.initErr = c.loadTables()
	})

	return c.initErr
}

// Compress writes the custom FFXI compressed data into dst and returns the bit count.
func (c *FFXICodec) Compress(src []byte, dst []byte) (uint32, error) {
	if c == nil {
		return 0, errors.New("codec is not initialized")
	}

	if err := c.EnsureInitialized(); err != nil {
		return 0, err
	}

	if len(dst) == 0 {
		return 0, errors.New("destination buffer is empty")
	}

	read := uint32(0)
	maxBits := uint32(len(dst)-1) * 8

	for _, b := range src {
		index := int(int8(b))
		if index+0x180 >= len(c.encTable) || index+0x80 >= len(c.encTable) {
			return 0, fmt.Errorf("encoding table index out of range for byte %d", b)
		}

		elem := c.encTable[index+0x180]
		if elem+read >= maxBits {
			if len(src)+1 >= len(dst) {
				fillOverflowOutput(dst, len(src))
				return uint32(len(src)), nil
			}

			return 0, fmt.Errorf("insufficient space in destination buffer (needed %d bits, have %d)", elem+read, maxBits)
		}

		pattern := c.encTable[index+0x80]
		var bitPattern [4]byte
		binary.LittleEndian.PutUint32(bitPattern[:], pattern)

		if err := compressSub(bitPattern[:], read, elem, dst[1:]); err != nil {
			return 0, err
		}

		read += elem
	}

	dst[0] = 1
	return read + 8, nil
}

// Decompress expands the provided compressed payload into dst and returns the number of bytes written.
func (c *FFXICodec) Decompress(in []byte, bitCount uint32, dst []byte) (int, error) {
	if c == nil {
		return 0, errors.New("codec is not initialized")
	}

	if err := c.EnsureInitialized(); err != nil {
		return 0, err
	}

	if len(in) == 0 {
		return 0, errors.New("empty compressed input")
	}

	if bitCount == 0 {
		return 0, errors.New("invalid compressed size (0 bits)")
	}

	if in[0] != 1 {
		return 0, errors.New("invalid compressed header")
	}

	if c.jumpRoot == nil {
		return 0, errors.New("jump table not initialized")
	}

	written := 0
	dataBits := bitCount
	data := in[1:]
	if uint32(len(data))*8 < dataBits {
		return 0, fmt.Errorf("compressed input shorter than expected (%d bits needed, %d bytes provided)", dataBits, len(data))
	}

	jmp := c.jumpRoot

	for i := uint32(0); i < dataBits && written < len(dst); i++ {
		byteIndex := i / 8
		if int(byteIndex) >= len(data) {
			return 0, fmt.Errorf("compressed input overflow at bit %d", i)
		}

		bit := int((data[byteIndex] >> (i & 7)) & 1)
		child := jumpEntryAt(jmp, bit)
		if child == nil || child.ptr == nil {
			return 0, errors.New("invalid jump pointer")
		}

		next := (*jumpEntry)(child.ptr)
		if next == nil {
			return 0, errors.New("nil jump target during decompression")
		}

		jmp = next

		left := jumpEntryAt(jmp, 0)
		right := jumpEntryAt(jmp, 1)
		if (left != nil && left.ptr != nil) || (right != nil && right.ptr != nil) {
			continue
		}

		valueEntry := jumpEntryAt(jmp, 3)
		if valueEntry == nil || !valueEntry.hasValue {
			return 0, errors.New("invalid value entry")
		}

		val := valueEntry.value
		dst[written] = val
		written++
		jmp = c.jumpRoot
	}

	return written, nil
}

func fillOverflowOutput(dst []byte, inputLen int) {
	if len(dst) == 0 {
		return
	}

	zeroLen := len(dst)/4 + (inputLen & 3)
	if zeroLen > len(dst) {
		zeroLen = len(dst)
	}
	for i := 0; i < zeroLen; i++ {
		dst[i] = 0
	}

	if len(dst) <= 1 {
		return
	}

	secondLen := inputLen / 4
	if secondLen > len(dst)-1 {
		secondLen = len(dst) - 1
	}
	for i := 0; i < secondLen; i++ {
		dst[1+i] = byte(inputLen)
	}

	offset := 1 + inputLen/4
	if offset > len(dst) {
		offset = len(dst)
	}

	thirdLen := inputLen & 3
	if offset+thirdLen > len(dst) {
		thirdLen = len(dst) - offset
	}

	for i := 0; i < thirdLen; i++ {
		dst[offset+i] = byte((inputLen + 1) * 8)
	}
}

func (c *FFXICodec) loadTables() error {
	compressData, err := c.readResource(CompressFileName)
	if err != nil {
		return err
	}

	decompressData, err := c.readResource(DecompressFileName)
	if err != nil {
		return err
	}

	c.encTable, err = bytesToUint32(compressData)
	if err != nil {
		return err
	}

	decValues, err := bytesToUint32(decompressData)
	if err != nil {
		return err
	}

	if err := c.populateJumpTable(decValues); err != nil {
		return err
	}

	return nil
}

func (c *FFXICodec) readResource(name string) ([]byte, error) {
	base := c.resourcePath
	if base == "" {
		base = "."
	}

	path := filepath.Join(base, name)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("resource %s not found at %s: %w", name, path, err)
	}

	return data, nil
}

func (c *FFXICodec) populateJumpTable(dec []uint32) error {
	if len(dec) == 0 {
		return errors.New("empty decompression table")
	}

	c.jumpTable = make([]jumpEntry, len(dec))
	base := dec[0] - 4

	for i, entry := range dec {
		if entry > 0xFF {
			offset := (entry - base) / 4
			if offset >= uint32(len(c.jumpTable)) {
				return fmt.Errorf("jump pointer out of range (%d >= %d)", offset, len(c.jumpTable))
			}
			c.jumpTable[i].ptr = unsafe.Pointer(&c.jumpTable[offset])
			c.jumpTable[i].hasValue = false
		} else {
			c.jumpTable[i].ptr = nil
			c.jumpTable[i].hasValue = true
			c.jumpTable[i].value = byte(entry)
		}
	}

	root := c.jumpTable[0].ptr
	if root == nil {
		return errors.New("invalid decompression root pointer")
	}

	c.jumpRoot = (*jumpEntry)(root)
	return nil
}
