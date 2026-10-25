package mp4

import (
	"encoding/binary"
	"fmt"
	"io"
)

// BoxHeader represents the 8-byte or 16-byte header of an ISO Base Media File Format box.
type BoxHeader struct {
	Type        string // 4-byte FourCC
	Size        uint64 // Total box size including header
	HeaderSize  uint32 // 8 for standard 32-bit size, 16 for 64-bit largesize
	PayloadSize uint64 // Size - HeaderSize
	Offset      int64  // File offset where box starts
}

// ReadBoxHeader reads an ISO box header at the current position.
func ReadBoxHeader(r io.Reader, currentOffset int64) (*BoxHeader, error) {
	var buf [8]byte
	if _, err := io.ReadFull(r, buf[:]); err != nil {
		return nil, err
	}

	size32 := binary.BigEndian.Uint32(buf[0:4])
	boxType := string(buf[4:8])

	header := &BoxHeader{
		Type:       boxType,
		Offset:     currentOffset,
		HeaderSize: 8,
	}

	if size32 == 1 {
		// 64-bit largesize follows
		var extBuf [8]byte
		if _, err := io.ReadFull(r, extBuf[:]); err != nil {
			return nil, fmt.Errorf("failed to read 64-bit box size for %s: %w", boxType, err)
		}
		header.Size = binary.BigEndian.Uint64(extBuf[:])
		header.HeaderSize = 16
	} else if size32 == 0 {
		// Box extends to end of file (typically trailing mdat)
		header.Size = 0
		header.HeaderSize = 8
	} else {
		header.Size = uint64(size32)
	}

	if header.Size >= uint64(header.HeaderSize) {
		header.PayloadSize = header.Size - uint64(header.HeaderSize)
	} else {
		header.PayloadSize = 0
	}

	return header, nil
}

// ReadFullBoxHeader reads version and flags for FullBox types.
func ReadFullBoxHeader(r io.Reader) (version uint8, flags uint32, err error) {
	var buf [4]byte
	if _, err := io.ReadFull(r, buf[:]); err != nil {
		return 0, 0, err
	}
	version = buf[0]
	flags = uint32(buf[1])<<16 | uint32(buf[2])<<8 | uint32(buf[3])
	return version, flags, nil
}

// Fixed16_16ToFloat converts 16.16 fixed-point number to float64.
func Fixed16_16ToFloat(val uint32) float64 {
	return float64(val) / 65536.0
}

// Fixed8_8ToFloat converts 8.8 fixed-point number to float64.
func Fixed8_8ToFloat(val uint16) float64 {
	return float64(val) / 256.0
}
