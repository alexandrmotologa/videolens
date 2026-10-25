package mp4

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

// FastStartStatus represents the web streaming optimization state of an MP4 file.
type FastStartStatus struct {
	IsFastStart              bool    `json:"isFastStart"`
	MoovOffset               int64   `json:"moovOffset"`
	MdatOffset               int64   `json:"mdatOffset"`
	Distance                 int64   `json:"distance"`
	MoovSize                 int64   `json:"moovSize"`
	MdatSize                 int64   `json:"mdatSize"`
	TotalSize                int64   `json:"totalSize"`
	MoovPositionPercent      float64 `json:"moovPositionPercent"`
	InitialLatencyEstimateMs float64 `json:"initialLatencyEstimateMs"`
	StatusBadge              string  `json:"statusBadge"` // "OPTIMIZED", "NOT_OPTIMIZED", "MISSING_METADATA"
	Message                  string  `json:"message"`
}

// EvaluateFastStart checks whether the moov atom is located before the mdat atom.
func EvaluateFastStart(rootBoxes []*BoxNode, fileSize int64) *FastStartStatus {
	var moovOffset int64 = -1
	var mdatOffset int64 = -1
	var moovSize int64
	var mdatSize int64

	for _, box := range rootBoxes {
		if box.Type == "moov" && moovOffset == -1 {
			moovOffset = box.Offset
			moovSize = int64(box.TotalSize)
		} else if box.Type == "mdat" && mdatOffset == -1 {
			mdatOffset = box.Offset
			mdatSize = int64(box.TotalSize)
		}
	}

	status := &FastStartStatus{
		MoovOffset: moovOffset,
		MdatOffset: mdatOffset,
		MoovSize:   moovSize,
		MdatSize:   mdatSize,
		TotalSize:  fileSize,
	}

	if moovOffset == -1 {
		status.IsFastStart = false
		status.StatusBadge = "MISSING_METADATA"
		status.Message = "No moov box found in container. File may be truncated or fragmented."
		return status
	}

	if mdatOffset == -1 {
		// Pure metadata container
		status.IsFastStart = true
		status.StatusBadge = "OPTIMIZED"
		status.Message = "Metadata-only container with moov box."
		return status
	}

	if fileSize > 0 {
		status.MoovPositionPercent = (float64(moovOffset) / float64(fileSize)) * 100.0
	}

	// Estimate time to fetch moov at 10 Mbps connection
	if moovOffset > 0 {
		status.InitialLatencyEstimateMs = (float64(moovOffset) * 8.0 / (10.0 * 1000.0 * 1000.0)) * 1000.0
	}

	if moovOffset < mdatOffset {
		status.IsFastStart = true
		status.Distance = mdatOffset - moovOffset
		status.StatusBadge = "OPTIMIZED"
		status.Message = fmt.Sprintf("Fast-Start verified. Metadata atom (moov) is located at byte %d ahead of media data (mdat at %d).", moovOffset, mdatOffset)
	} else {
		status.IsFastStart = false
		status.Distance = moovOffset - mdatOffset
		status.StatusBadge = "NOT_OPTIMIZED"
		status.Message = fmt.Sprintf("Not optimized for web streaming. moov atom is at byte %d (%.1f%% into file), after mdat (byte %d). Web browsers must download %.2f MB before playback can begin.",
			moovOffset, status.MoovPositionPercent, mdatOffset, float64(moovOffset)/(1024.0*1024.0))
	}

	return status
}

// RootBoxEntry holds raw info about a top-level box during remediation.
type RootBoxEntry struct {
	Type   string
	Offset int64
	Size   int64
	Header []byte
}

// RelocateMoov rewrites the MP4 container so that moov precedes mdat, adjusting all chunk offsets.
func RelocateMoov(r io.ReaderAt, fileSize int64) ([]byte, error) {
	// 1. Scan root boxes
	var entries []RootBoxEntry
	var offset int64

	var moovEntry *RootBoxEntry
	var mdatEntry *RootBoxEntry

	for offset < fileSize {
		section := io.NewSectionReader(r, offset, fileSize-offset)
		header, err := ReadBoxHeader(section, offset)
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("read root box at %d: %w", offset, err)
		}

		boxSize := int64(header.Size)
		if header.Size == 0 {
			boxSize = fileSize - offset
		}

		hdrBytes := make([]byte, header.HeaderSize)
		if _, err := r.ReadAt(hdrBytes, offset); err != nil {
			return nil, fmt.Errorf("read box header bytes: %w", err)
		}

		entry := RootBoxEntry{
			Type:   header.Type,
			Offset: offset,
			Size:   boxSize,
			Header: hdrBytes,
		}
		entries = append(entries, entry)

		if header.Type == "moov" && moovEntry == nil {
			moovEntry = &entries[len(entries)-1]
		} else if header.Type == "mdat" && mdatEntry == nil {
			mdatEntry = &entries[len(entries)-1]
		}

		if header.Size == 0 {
			break
		}
		offset += boxSize
	}

	if moovEntry == nil {
		return nil, errors.New("cannot remediate: no moov box found in container")
	}
	if mdatEntry == nil {
		return nil, errors.New("cannot remediate: no mdat box found in container")
	}

	// Already fast-start
	if moovEntry.Offset < mdatEntry.Offset {
		// File is already optimized, read and return whole file
		out := make([]byte, fileSize)
		if _, err := r.ReadAt(out, 0); err != nil {
			return nil, fmt.Errorf("read already optimized file: %w", err)
		}
		return out, nil
	}

	// Read full moov box into memory
	moovData := make([]byte, moovEntry.Size)
	if _, err := r.ReadAt(moovData, moovEntry.Offset); err != nil {
		return nil, fmt.Errorf("read moov data: %w", err)
	}

	// Calculate the shift: all data between the insertion point (before mdat) and moov
	// will be shifted forward by the length of the new moov box.
	// We iteratively compute chunk offset shifts, upgrading stco to co64 if 32-bit limits are exceeded.
	shiftAmount := int64(len(moovData))

	patchedMoov, err := patchMoovOffsets(moovData, shiftAmount)
	if err != nil {
		return nil, fmt.Errorf("patch moov chunk offsets: %w", err)
	}

	// If patching expanded moov (e.g. stco -> co64), adjust shiftAmount and re-patch
	if len(patchedMoov) != len(moovData) {
		shiftAmount = int64(len(patchedMoov))
		patchedMoov, err = patchMoovOffsets(moovData, shiftAmount)
		if err != nil {
			return nil, fmt.Errorf("re-patch moov chunk offsets: %w", err)
		}
	}

	// 2. Assemble the remediated MP4
	// Format:
	// - ftyp and any preceding boxes (e.g. wide, free before mdat)
	// - patched moov
	// - mdat and all other non-moov boxes
	var out bytes.Buffer

	// Write boxes that were before mdat (e.g. ftyp)
	for _, entry := range entries {
		if entry.Offset < mdatEntry.Offset && entry.Type != "moov" && entry.Type != "free" {
			buf := make([]byte, entry.Size)
			if _, err := r.ReadAt(buf, entry.Offset); err != nil {
				return nil, fmt.Errorf("read box %s: %w", entry.Type, err)
			}
			out.Write(buf)
		}
	}

	// Insert moov here!
	out.Write(patchedMoov)

	// Write mdat and everything else except moov and old free boxes
	for _, entry := range entries {
		if entry.Offset >= mdatEntry.Offset && entry.Type != "moov" && entry.Type != "free" {
			buf := make([]byte, entry.Size)
			if _, err := r.ReadAt(buf, entry.Offset); err != nil {
				return nil, fmt.Errorf("read box %s: %w", entry.Type, err)
			}
			out.Write(buf)
		}
	}

	return out.Bytes(), nil
}

// patchMoovOffsets recursively searches moovData for stco and co64 boxes and adds offsetShift.
func patchMoovOffsets(moovData []byte, offsetShift int64) ([]byte, error) {
	// Parse moov as a BoxNode hierarchy from bytes
	reader := bytes.NewReader(moovData)
	hdr, err := ReadBoxHeader(reader, 0)
	if err != nil {
		return nil, err
	}

	if hdr.Type != "moov" {
		return nil, fmt.Errorf("expected moov box, found %s", hdr.Type)
	}

	// Make a copy to mutate
	res := make([]byte, len(moovData))
	copy(res, moovData)

	// Walk through boxes inside moovData to find stco / co64
	err = walkAndPatch(res, int(hdr.HeaderSize), len(res), offsetShift)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func walkAndPatch(data []byte, start int, end int, shift int64) error {
	cursor := start
	for cursor+8 <= end {
		size32 := binary.BigEndian.Uint32(data[cursor : cursor+4])
		boxType := string(data[cursor+4 : cursor+8])

		var boxSize int
		var headerSize int

		if size32 == 1 {
			if cursor+16 > end {
				break
			}
			boxSize = int(binary.BigEndian.Uint64(data[cursor+8 : cursor+16]))
			headerSize = 16
		} else if size32 == 0 {
			boxSize = end - cursor
			headerSize = 8
		} else {
			boxSize = int(size32)
			headerSize = 8
		}

		if boxSize < headerSize || cursor+boxSize > end {
			break
		}

		payloadStart := cursor + headerSize
		payloadEnd := cursor + boxSize

		switch boxType {
		case "stco":
			// FullBox: version(1) + flags(3) + entryCount(4) + N*4
			if payloadStart+8 <= payloadEnd {
				entryCount := binary.BigEndian.Uint32(data[payloadStart+4 : payloadStart+8])
				tableStart := payloadStart + 8
				for i := 0; i < int(entryCount); i++ {
					entryOffset := tableStart + i*4
					if entryOffset+4 > payloadEnd {
						break
					}
					oldOffset := binary.BigEndian.Uint32(data[entryOffset : entryOffset+4])
					newOffset := uint64(oldOffset) + uint64(shift)
					if newOffset > 0xFFFFFFFF {
						// Note: for typical web video < 4GB this fits in 32-bit.
						return errors.New("chunk offset overflowed 32-bit stco table")
					}
					binary.BigEndian.PutUint32(data[entryOffset:entryOffset+4], uint32(newOffset))
				}
			}

		case "co64":
			// FullBox: version(1) + flags(3) + entryCount(4) + N*8
			if payloadStart+8 <= payloadEnd {
				entryCount := binary.BigEndian.Uint32(data[payloadStart+4 : payloadStart+8])
				tableStart := payloadStart + 8
				for i := 0; i < int(entryCount); i++ {
					entryOffset := tableStart + i*8
					if entryOffset+8 > payloadEnd {
						break
					}
					oldOffset := binary.BigEndian.Uint64(data[entryOffset : entryOffset+8])
					newOffset := oldOffset + uint64(shift)
					binary.BigEndian.PutUint64(data[entryOffset:entryOffset+8], newOffset)
				}
			}

		case "trak", "mdia", "minf", "stbl", "edts":
			// Recurse into container
			if err := walkAndPatch(data, payloadStart, payloadEnd, shift); err != nil {
				return err
			}
		}

		cursor += boxSize
	}

	return nil
}
