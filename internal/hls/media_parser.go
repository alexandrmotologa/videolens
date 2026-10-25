package hls

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// PartialSegment represents an #EXT-X-PART in Low-Latency HLS (LL-HLS).
type PartialSegment struct {
	Index       int     `json:"index"`
	Duration    float64 `json:"duration"`
	URI         string  `json:"uri"`
	Independent bool    `json:"independent"`
	ByteRange   string  `json:"byteRange,omitempty"`
	LineNumber  int     `json:"lineNumber"`
}

// Segment represents a media segment in an HLS media playlist.
type Segment struct {
	Index           int              `json:"index"`
	Duration        float64          `json:"duration"`
	Title           string           `json:"title,omitempty"`
	URI             string           `json:"uri"`
	LineNumber      int              `json:"lineNumber"`
	ByteRangeLength int64            `json:"byteRangeLength,omitempty"`
	ByteRangeOffset int64            `json:"byteRangeOffset,omitempty"`
	Discontinuity   bool             `json:"discontinuity"`
	ProgramDateTime string           `json:"programDateTime,omitempty"`
	KeyMethod       string           `json:"keyMethod,omitempty"`
	KeyURI          string           `json:"keyUri,omitempty"`
	Parts           []PartialSegment `json:"parts,omitempty"`
}

// MediaPlaylist holds parsed components of an HLS media playlist.
type MediaPlaylist struct {
	Version               int              `json:"version"`
	TargetDuration        float64          `json:"targetDuration"`
	MediaSequence         int64            `json:"mediaSequence"`
	DiscontinuitySequence int64            `json:"discontinuitySequence"`
	PlaylistType          string           `json:"playlistType,omitempty"` // "VOD", "EVENT"
	HasEndList            bool             `json:"hasEndList"`
	IFrameOnly            bool             `json:"iFrameOnly"`
	TotalDuration         float64          `json:"totalDuration"`
	IsLowLatency          bool             `json:"isLowLatency"`
	PartTarget            float64          `json:"partTarget,omitempty"`
	ServerControl         map[string]string `json:"serverControl,omitempty"`
	PreloadHint           map[string]string `json:"preloadHint,omitempty"`
	TotalPartsCount       int              `json:"totalPartsCount"`
	Segments              []Segment        `json:"segments"`
	RawLines              []string         `json:"rawLines"`
}

// ParseMediaPlaylist parses an HLS media playlist from a reader.
func ParseMediaPlaylist(r io.Reader) (*MediaPlaylist, error) {
	scanner := bufio.NewScanner(r)
	media := &MediaPlaylist{
		Version:  1,
		Segments: make([]Segment, 0),
	}

	var currentSegment *Segment
	var pendingDiscontinuity bool
	var pendingProgramDateTime string
	var currentKeyMethod string
	var currentKeyURI string
	var currentByteRangeOffset int64

	lineNum := 0
	segmentIndex := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		media.RawLines = append(media.RawLines, line)

		if line == "" {
			continue
		}

		// Segment URI line
		if currentSegment != nil && !strings.HasPrefix(line, "#") {
			currentSegment.URI = line
			media.TotalDuration += currentSegment.Duration
			media.Segments = append(media.Segments, *currentSegment)
			currentSegment = nil
			pendingDiscontinuity = false
			segmentIndex++
			continue
		}

		if strings.HasPrefix(line, "#EXT-X-VERSION:") {
			val := strings.TrimPrefix(line, "#EXT-X-VERSION:")
			if v, err := strconv.Atoi(strings.TrimSpace(val)); err == nil {
				media.Version = v
			}
		} else if strings.HasPrefix(line, "#EXT-X-TARGETDURATION:") {
			val := strings.TrimPrefix(line, "#EXT-X-TARGETDURATION:")
			if td, err := strconv.ParseFloat(strings.TrimSpace(val), 64); err == nil {
				media.TargetDuration = td
			}
		} else if strings.HasPrefix(line, "#EXT-X-MEDIA-SEQUENCE:") {
			val := strings.TrimPrefix(line, "#EXT-X-MEDIA-SEQUENCE:")
			if seq, err := strconv.ParseInt(strings.TrimSpace(val), 10, 64); err == nil {
				media.MediaSequence = seq
			}
		} else if strings.HasPrefix(line, "#EXT-X-DISCONTINUITY-SEQUENCE:") {
			val := strings.TrimPrefix(line, "#EXT-X-DISCONTINUITY-SEQUENCE:")
			if dseq, err := strconv.ParseInt(strings.TrimSpace(val), 10, 64); err == nil {
				media.DiscontinuitySequence = dseq
			}
		} else if strings.HasPrefix(line, "#EXT-X-PART-INF:") {
			media.IsLowLatency = true
			attrs := parseAttributes(strings.TrimPrefix(line, "#EXT-X-PART-INF:"))
			if ptStr, ok := attrs["PART-TARGET"]; ok {
				if pt, err := strconv.ParseFloat(ptStr, 64); err == nil {
					media.PartTarget = pt
				}
			}
		} else if strings.HasPrefix(line, "#EXT-X-SERVER-CONTROL:") {
			media.ServerControl = parseAttributes(strings.TrimPrefix(line, "#EXT-X-SERVER-CONTROL:"))
		} else if strings.HasPrefix(line, "#EXT-X-PRELOAD-HINT:") {
			media.PreloadHint = parseAttributes(strings.TrimPrefix(line, "#EXT-X-PRELOAD-HINT:"))
		} else if strings.HasPrefix(line, "#EXT-X-PART:") {
			media.IsLowLatency = true
			media.TotalPartsCount++
			attrs := parseAttributes(strings.TrimPrefix(line, "#EXT-X-PART:"))
			dur, _ := strconv.ParseFloat(attrs["DURATION"], 64)
			part := PartialSegment{
				Index:       media.TotalPartsCount,
				Duration:    dur,
				URI:         attrs["URI"],
				Independent: strings.ToUpper(attrs["INDEPENDENT"]) == "YES",
				ByteRange:   attrs["BYTERANGE"],
				LineNumber:  lineNum,
			}
			if currentSegment != nil {
				currentSegment.Parts = append(currentSegment.Parts, part)
			}
		} else if strings.HasPrefix(line, "#EXT-X-PLAYLIST-TYPE:") {
			media.PlaylistType = strings.TrimSpace(strings.TrimPrefix(line, "#EXT-X-PLAYLIST-TYPE:"))
		} else if strings.HasPrefix(line, "#EXT-X-ENDLIST") {
			media.HasEndList = true
		} else if strings.HasPrefix(line, "#EXT-X-I-FRAMES-ONLY") {
			media.IFrameOnly = true
		} else if strings.HasPrefix(line, "#EXT-X-DISCONTINUITY") {
			pendingDiscontinuity = true
		} else if strings.HasPrefix(line, "#EXT-X-PROGRAM-DATE-TIME:") {
			pendingProgramDateTime = strings.TrimSpace(strings.TrimPrefix(line, "#EXT-X-PROGRAM-DATE-TIME:"))
		} else if strings.HasPrefix(line, "#EXT-X-KEY:") {
			attrs := parseAttributes(strings.TrimPrefix(line, "#EXT-X-KEY:"))
			currentKeyMethod = attrs["METHOD"]
			currentKeyURI = attrs["URI"]
		} else if strings.HasPrefix(line, "#EXTINF:") {
			inf := strings.TrimPrefix(line, "#EXTINF:")
			parts := strings.SplitN(inf, ",", 2)
			durStr := strings.TrimSpace(parts[0])
			title := ""
			if len(parts) > 1 {
				title = strings.TrimSpace(parts[1])
			}

			dur, _ := strconv.ParseFloat(durStr, 64)
			currentSegment = &Segment{
				Index:           segmentIndex,
				Duration:        dur,
				Title:           title,
				LineNumber:      lineNum,
				Discontinuity:   pendingDiscontinuity,
				ProgramDateTime: pendingProgramDateTime,
				KeyMethod:       currentKeyMethod,
				KeyURI:          currentKeyURI,
			}
			pendingProgramDateTime = ""
		} else if strings.HasPrefix(line, "#EXT-X-BYTERANGE:") && currentSegment != nil {
			brStr := strings.TrimPrefix(line, "#EXT-X-BYTERANGE:")
			brParts := strings.SplitN(brStr, "@", 2)
			if len(brParts) >= 1 {
				length, _ := strconv.ParseInt(strings.TrimSpace(brParts[0]), 10, 64)
				currentSegment.ByteRangeLength = length
				if len(brParts) == 2 {
					offset, _ := strconv.ParseInt(strings.TrimSpace(brParts[1]), 10, 64)
					currentSegment.ByteRangeOffset = offset
					currentByteRangeOffset = offset + length
				} else {
					currentSegment.ByteRangeOffset = currentByteRangeOffset
					currentByteRangeOffset += length
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read media playlist: %w", err)
	}

	return media, nil
}
