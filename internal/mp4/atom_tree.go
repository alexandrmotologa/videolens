package mp4

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"strings"
)

// BoxNode represents a parsed ISO box in a hierarchical tree.
type BoxNode struct {
	Type        string                 `json:"type"`
	Offset      int64                  `json:"offset"`
	HeaderSize  uint32                 `json:"headerSize"`
	PayloadSize uint64                 `json:"payloadSize"`
	TotalSize   uint64                 `json:"totalSize"`
	Children    []*BoxNode             `json:"children,omitempty"`
	Details     map[string]interface{} `json:"details,omitempty"`
}

// TrackSummary summarizes detected video or audio track properties.
type TrackSummary struct {
	TrackID         uint32  `json:"trackId"`
	Type            string  `json:"type"` // "video", "audio", "hint", "meta", "unknown"
	Codec           string  `json:"codec"`
	DurationSeconds float64 `json:"durationSeconds"`
	Timescale       uint32  `json:"timescale"`
	Width           float64 `json:"width,omitempty"`
	Height          float64 `json:"height,omitempty"`
	SampleRate      uint32  `json:"sampleRate,omitempty"`
	Channels        uint16  `json:"channels,omitempty"`
	KeyframeCount   int     `json:"keyframeCount,omitempty"`
	TotalFrames     int     `json:"totalFrames,omitempty"`
	AvgKeyframeSec  float64 `json:"avgKeyframeSec,omitempty"`
}

// MP4Profile represents the complete diagnostic profile of an MP4 file.
type MP4Profile struct {
	FileSize        int64            `json:"fileSize"`
	DurationSeconds float64          `json:"durationSeconds"`
	Timescale       uint32           `json:"timescale"`
	MajorBrand      string           `json:"majorBrand"`
	Compatible      []string         `json:"compatibleBrands"`
	FastStart       *FastStartStatus `json:"fastStart"`
	Tracks          []TrackSummary   `json:"tracks"`
	RootBoxes       []*BoxNode       `json:"rootBoxes"`
}

var containerBoxes = map[string]bool{
	"moov": true,
	"trak": true,
	"edts": true,
	"mdia": true,
	"minf": true,
	"dinf": true,
	"stbl": true,
	"mvex": true,
	"moof": true,
	"traf": true,
	"udta": true,
}

// ParseMP4 parses an MP4 stream from an io.ReaderAt with known total file size.
func ParseMP4(r io.ReaderAt, fileSize int64) (*MP4Profile, error) {
	profile := &MP4Profile{
		FileSize:  fileSize,
		RootBoxes: make([]*BoxNode, 0),
		Tracks:    make([]TrackSummary, 0),
	}

	var offset int64
	for offset < fileSize {
		sectionReader := io.NewSectionReader(r, offset, fileSize-offset)
		header, err := ReadBoxHeader(sectionReader, offset)
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("read box header at offset %d: %w", offset, err)
		}

		if header.Size == 0 {
			header.Size = uint64(fileSize - offset)
			header.PayloadSize = header.Size - uint64(header.HeaderSize)
		}

		node := &BoxNode{
			Type:        header.Type,
			Offset:      header.Offset,
			HeaderSize:  header.HeaderSize,
			PayloadSize: header.PayloadSize,
			TotalSize:   header.Size,
			Details:     make(map[string]interface{}),
		}

		// Read payload if small enough or container
		if containerBoxes[header.Type] {
			children, err := parseContainerChildren(r, offset+int64(header.HeaderSize), int64(header.PayloadSize))
			if err == nil {
				node.Children = children
			}
		} else {
			parseLeafBox(r, header, node, profile)
		}

		profile.RootBoxes = append(profile.RootBoxes, node)

		if header.Size == 0 {
			break
		}
		offset += int64(header.Size)
	}

	// Extract track summaries and calculate duration from parsed boxes
	extractTracksAndDuration(profile)

	// Evaluate Fast-Start status
	profile.FastStart = EvaluateFastStart(profile.RootBoxes, fileSize)

	return profile, nil
}

func parseContainerChildren(r io.ReaderAt, startOffset int64, length int64) ([]*BoxNode, error) {
	children := make([]*BoxNode, 0)
	var relOffset int64

	for relOffset < length {
		absOffset := startOffset + relOffset
		remaining := length - relOffset
		if remaining < 8 {
			break
		}

		section := io.NewSectionReader(r, absOffset, remaining)
		header, err := ReadBoxHeader(section, absOffset)
		if err != nil {
			break
		}

		if header.Size == 0 || header.Size > uint64(remaining) {
			header.Size = uint64(remaining)
			header.PayloadSize = header.Size - uint64(header.HeaderSize)
		}

		childNode := &BoxNode{
			Type:        header.Type,
			Offset:      header.Offset,
			HeaderSize:  header.HeaderSize,
			PayloadSize: header.PayloadSize,
			TotalSize:   header.Size,
			Details:     make(map[string]interface{}),
		}

		if containerBoxes[header.Type] {
			subChildren, err := parseContainerChildren(r, absOffset+int64(header.HeaderSize), int64(header.PayloadSize))
			if err == nil {
				childNode.Children = subChildren
			}
		} else {
			parseLeafBox(r, header, childNode, nil)
		}

		children = append(children, childNode)
		relOffset += int64(header.Size)
	}

	return children, nil
}

func parseLeafBox(r io.ReaderAt, header *BoxHeader, node *BoxNode, profile *MP4Profile) {
	payloadLen := header.PayloadSize
	if payloadLen == 0 {
		return
	}

	// Limit in-memory payload reads to 64KB for inspection
	readCap := payloadLen
	if readCap > 65536 {
		readCap = 65536
	}

	buf := make([]byte, readCap)
	n, err := r.ReadAt(buf, header.Offset+int64(header.HeaderSize))
	if err != nil && err != io.EOF {
		return
	}
	buf = buf[:n]

	switch header.Type {
	case "ftyp":
		if len(buf) >= 8 {
			major := string(buf[0:4])
			minor := binary.BigEndian.Uint32(buf[4:8])
			node.Details["majorBrand"] = strings.TrimRight(major, "\x00 ")
			node.Details["minorVersion"] = minor

			var brands []string
			for i := 8; i+4 <= len(buf); i += 4 {
				b := strings.TrimRight(string(buf[i:i+4]), "\x00 ")
				if b != "" {
					brands = append(brands, b)
				}
			}
			node.Details["compatibleBrands"] = brands

			if profile != nil {
				profile.MajorBrand = node.Details["majorBrand"].(string)
				profile.Compatible = brands
			}
		}

	case "mvhd":
		if len(buf) >= 20 {
			version := buf[0]
			var timescale uint32
			var duration uint64

			if version == 1 && len(buf) >= 32 {
				timescale = binary.BigEndian.Uint32(buf[20:24])
				duration = binary.BigEndian.Uint64(buf[24:32])
			} else if len(buf) >= 20 {
				timescale = binary.BigEndian.Uint32(buf[12:16])
				duration = uint64(binary.BigEndian.Uint32(buf[16:20]))
			}

			node.Details["timescale"] = timescale
			node.Details["duration"] = duration
			if timescale > 0 {
				durSec := float64(duration) / float64(timescale)
				node.Details["durationSeconds"] = durSec
				if profile != nil {
					profile.Timescale = timescale
					profile.DurationSeconds = durSec
				}
			}
		}

	case "tkhd":
		if len(buf) >= 24 {
			version := buf[0]
			var trackID uint32
			var duration uint64
			var widthVal, heightVal uint32

			if version == 1 && len(buf) >= 88 {
				trackID = binary.BigEndian.Uint32(buf[20:24])
				duration = binary.BigEndian.Uint64(buf[28:36])
				widthVal = binary.BigEndian.Uint32(buf[80:84])
				heightVal = binary.BigEndian.Uint32(buf[84:88])
			} else if len(buf) >= 84 {
				trackID = binary.BigEndian.Uint32(buf[12:16])
				duration = uint64(binary.BigEndian.Uint32(buf[20:24]))
				widthVal = binary.BigEndian.Uint32(buf[76:80])
				heightVal = binary.BigEndian.Uint32(buf[80:84])
			}

			node.Details["trackId"] = trackID
			node.Details["duration"] = duration
			node.Details["width"] = Fixed16_16ToFloat(widthVal)
			node.Details["height"] = Fixed16_16ToFloat(heightVal)
		}

	case "mdhd":
		if len(buf) >= 20 {
			version := buf[0]
			var timescale uint32
			var duration uint64
			var langCode uint16

			if version == 1 && len(buf) >= 34 {
				timescale = binary.BigEndian.Uint32(buf[20:24])
				duration = binary.BigEndian.Uint64(buf[24:32])
				langCode = binary.BigEndian.Uint16(buf[32:34])
			} else if len(buf) >= 22 {
				timescale = binary.BigEndian.Uint32(buf[12:16])
				duration = uint64(binary.BigEndian.Uint32(buf[16:20]))
				langCode = binary.BigEndian.Uint16(buf[20:22])
			}

			node.Details["timescale"] = timescale
			node.Details["duration"] = duration
			if timescale > 0 {
				node.Details["durationSeconds"] = float64(duration) / float64(timescale)
			}
			// Decode ISO-639-2 language code
			c1 := string(rune(((langCode >> 10) & 0x1F) + 0x60))
			c2 := string(rune(((langCode >> 5) & 0x1F) + 0x60))
			c3 := string(rune((langCode & 0x1F) + 0x60))
			node.Details["language"] = c1 + c2 + c3
		}

	case "hdlr":
		if len(buf) >= 16 {
			handlerType := string(buf[8:12])
			node.Details["handlerType"] = handlerType

			name := ""
			if len(buf) > 24 {
				name = strings.TrimRight(string(buf[24:]), "\x00 ")
			}
			node.Details["name"] = name
		}

	case "stsd":
		if len(buf) >= 8 {
			entryCount := binary.BigEndian.Uint32(buf[4:8])
			node.Details["entryCount"] = entryCount

			var codecs []string
			cursor := 8
			for i := 0; i < int(entryCount) && cursor+8 <= len(buf); i++ {
				entrySize := binary.BigEndian.Uint32(buf[cursor : cursor+4])
				codecFourCC := string(buf[cursor+4 : cursor+8])
				codecs = append(codecs, codecFourCC)

				// If visual sample entry (avc1, hvc1, etc.)
				if cursor+36 <= len(buf) {
					w := binary.BigEndian.Uint16(buf[cursor+24 : cursor+26])
					h := binary.BigEndian.Uint16(buf[cursor+26 : cursor+28])
					if w > 0 && h > 0 {
						node.Details["width"] = w
						node.Details["height"] = h
					}
				}
				// If audio sample entry (mp4a)
				if cursor+32 <= len(buf) {
					channels := binary.BigEndian.Uint16(buf[cursor+16 : cursor+18])
					sr := binary.BigEndian.Uint32(buf[cursor+22 : cursor+26]) >> 16
					if channels > 0 && sr > 0 {
						node.Details["channels"] = channels
						node.Details["sampleRate"] = sr
					}
				}

				if entrySize == 0 || int(entrySize) > len(buf)-cursor {
					break
				}
				cursor += int(entrySize)
			}
			node.Details["codecs"] = codecs
		}

	case "stts":
		if len(buf) >= 8 {
			entryCount := binary.BigEndian.Uint32(buf[4:8])
			node.Details["entryCount"] = entryCount
		}

	case "stss":
		if len(buf) >= 8 {
			entryCount := binary.BigEndian.Uint32(buf[4:8])
			node.Details["keyframeCount"] = entryCount
		}

	case "stsz":
		if len(buf) >= 12 {
			sampleSize := binary.BigEndian.Uint32(buf[4:8])
			sampleCount := binary.BigEndian.Uint32(buf[8:12])
			node.Details["sampleSize"] = sampleSize
			node.Details["sampleCount"] = sampleCount
		}

	case "stco":
		if len(buf) >= 8 {
			entryCount := binary.BigEndian.Uint32(buf[4:8])
			node.Details["chunkCount"] = entryCount
		}

	case "co64":
		if len(buf) >= 8 {
			entryCount := binary.BigEndian.Uint32(buf[4:8])
			node.Details["chunkCount64"] = entryCount
		}
	}
}

func extractTracksAndDuration(profile *MP4Profile) {
	// Look through moov children for trak
	for _, rootBox := range profile.RootBoxes {
		if rootBox.Type == "moov" {
			for _, child := range rootBox.Children {
				if child.Type == "trak" {
					track := parseTrackSummary(child)
					profile.Tracks = append(profile.Tracks, track)
				}
			}
		}
	}
}

func parseTrackSummary(trakNode *BoxNode) TrackSummary {
	track := TrackSummary{
		Type:  "unknown",
		Codec: "unknown",
	}

	findBoxes(trakNode, func(node *BoxNode) {
		switch node.Type {
		case "tkhd":
			if tid, ok := node.Details["trackId"].(uint32); ok {
				track.TrackID = tid
			}
			if w, ok := node.Details["width"].(float64); ok && w > 0 {
				track.Width = w
			}
			if h, ok := node.Details["height"].(float64); ok && h > 0 {
				track.Height = h
			}

		case "mdhd":
			if ts, ok := node.Details["timescale"].(uint32); ok {
				track.Timescale = ts
			}
			if durSec, ok := node.Details["durationSeconds"].(float64); ok {
				track.DurationSeconds = durSec
			}

		case "hdlr":
			if ht, ok := node.Details["handlerType"].(string); ok {
				switch ht {
				case "vide":
					track.Type = "video"
				case "soun":
					track.Type = "audio"
				case "hint":
					track.Type = "hint"
				case "meta":
					track.Type = "meta"
				}
			}

		case "stsd":
			if codecs, ok := node.Details["codecs"].([]string); ok && len(codecs) > 0 {
				track.Codec = strings.Join(codecs, ", ")
			}
			if w, ok := node.Details["width"].(uint16); ok && track.Width == 0 {
				track.Width = float64(w)
			}
			if h, ok := node.Details["height"].(uint16); ok && track.Height == 0 {
				track.Height = float64(h)
			}
			if ch, ok := node.Details["channels"].(uint16); ok {
				track.Channels = ch
			}
			if sr, ok := node.Details["sampleRate"].(uint32); ok {
				track.SampleRate = sr
			}

		case "stss":
			if kf, ok := node.Details["keyframeCount"].(uint32); ok {
				track.KeyframeCount = int(kf)
			}

		case "stsz":
			if sc, ok := node.Details["sampleCount"].(uint32); ok {
				track.TotalFrames = int(sc)
			}
		}
	})

	if track.KeyframeCount > 0 && track.DurationSeconds > 0 {
		track.AvgKeyframeSec = track.DurationSeconds / float64(track.KeyframeCount)
	}

	return track
}

func findBoxes(root *BoxNode, cb func(*BoxNode)) {
	cb(root)
	for _, child := range root.Children {
		findBoxes(child, cb)
	}
}

// FindFirstBox searches the tree for the first box matching targetType.
func (b *BoxNode) FindFirstBox(targetType string) *BoxNode {
	if b.Type == targetType {
		return b
	}
	for _, child := range b.Children {
		if found := child.FindFirstBox(targetType); found != nil {
			return found
		}
	}
	return nil
}

// FindAllBoxes searches the tree for all boxes matching targetType.
func (b *BoxNode) FindAllBoxes(targetType string) []*BoxNode {
	var result []*BoxNode
	if b.Type == targetType {
		result = append(result, b)
	}
	for _, child := range b.Children {
		result = append(result, child.FindAllBoxes(targetType)...)
	}
	return result
}

// PrintTree returns an ASCII visualization of the atom tree.
func (p *MP4Profile) PrintTree() string {
	var sb strings.Builder
	for _, box := range p.RootBoxes {
		renderBoxTree(&sb, box, 0)
	}
	return sb.String()
}

func renderBoxTree(sb *strings.Builder, node *BoxNode, depth int) {
	indent := strings.Repeat("  ", depth)
	sb.WriteString(fmt.Sprintf("%s[%s] offset=%d size=%d\n", indent, node.Type, node.Offset, node.TotalSize))
	for _, child := range node.Children {
		renderBoxTree(sb, child, depth+1)
	}
}

// Ensure reader implements bytes buffer checks
var _ io.ReaderAt = (*bytes.Reader)(nil)
