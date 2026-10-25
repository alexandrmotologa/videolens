package hls

import (
	"bufio"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
)

// VariantStream represents an #EXT-X-STREAM-INF entry in an HLS master playlist.
type VariantStream struct {
	Bandwidth        uint64  `json:"bandwidth"`
	AverageBandwidth uint64  `json:"averageBandwidth,omitempty"`
	Resolution       string  `json:"resolution,omitempty"`
	Width            int     `json:"width,omitempty"`
	Height           int     `json:"height,omitempty"`
	FrameRate        float64 `json:"frameRate,omitempty"`
	Codecs           string  `json:"codecs,omitempty"`
	Audio            string  `json:"audio,omitempty"`
	Subtitles        string  `json:"subtitles,omitempty"`
	URI              string  `json:"uri"`
	LineNumber       int     `json:"lineNumber"`
}

// MediaRendition represents an #EXT-X-MEDIA entry in an HLS master playlist.
type MediaRendition struct {
	Type       string `json:"type"` // "AUDIO", "VIDEO", "SUBTITLES", "CLOSED-CAPTIONS"
	GroupID    string `json:"groupId"`
	Name       string `json:"name"`
	Language   string `json:"language,omitempty"`
	Default    bool   `json:"default"`
	AutoSelect bool   `json:"autoSelect"`
	URI        string `json:"uri,omitempty"`
	LineNumber int    `json:"lineNumber"`
}

// MasterPlaylist holds parsed components of an HLS master playlist.
type MasterPlaylist struct {
	Version             int              `json:"version"`
	IndependentSegments bool             `json:"independentSegments"`
	Variants            []VariantStream  `json:"variants"`
	Renditions          []MediaRendition `json:"renditions"`
	RawLines            []string         `json:"rawLines"`
}

// ParseMasterPlaylist parses an HLS master playlist from a reader.
func ParseMasterPlaylist(r io.Reader) (*MasterPlaylist, error) {
	scanner := bufio.NewScanner(r)
	master := &MasterPlaylist{
		Version:  1,
		Variants: make([]VariantStream, 0),
	}

	var currentVariant *VariantStream
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		master.RawLines = append(master.RawLines, line)

		if line == "" {
			continue
		}

		if currentVariant != nil && !strings.HasPrefix(line, "#") {
			currentVariant.URI = line
			master.Variants = append(master.Variants, *currentVariant)
			currentVariant = nil
			continue
		}

		if strings.HasPrefix(line, "#EXT-X-VERSION:") {
			val := strings.TrimPrefix(line, "#EXT-X-VERSION:")
			if v, err := strconv.Atoi(strings.TrimSpace(val)); err == nil {
				master.Version = v
			}
		} else if strings.HasPrefix(line, "#EXT-X-INDEPENDENT-SEGMENTS") {
			master.IndependentSegments = true
		} else if strings.HasPrefix(line, "#EXT-X-STREAM-INF:") {
			attrStr := strings.TrimPrefix(line, "#EXT-X-STREAM-INF:")
			v := parseStreamInfAttributes(attrStr)
			v.LineNumber = lineNum
			currentVariant = &v
		} else if strings.HasPrefix(line, "#EXT-X-MEDIA:") {
			attrStr := strings.TrimPrefix(line, "#EXT-X-MEDIA:")
			m := parseMediaAttributes(attrStr)
			m.LineNumber = lineNum
			master.Renditions = append(master.Renditions, m)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read master playlist: %w", err)
	}

	return master, nil
}

// Attribute parser regex matching key=value or key="quoted value"
var attrRegex = regexp.MustCompile(`([A-Z0-9\-]+)=("[^"]*"|[^,]+)`)

func parseAttributes(attrStr string) map[string]string {
	res := make(map[string]string)
	matches := attrRegex.FindAllStringSubmatch(attrStr, -1)
	for _, m := range matches {
		key := strings.TrimSpace(m[1])
		val := strings.TrimSpace(m[2])
		val = strings.Trim(val, `"`)
		res[key] = val
	}
	return res
}

func parseStreamInfAttributes(attrStr string) VariantStream {
	attrs := parseAttributes(attrStr)
	v := VariantStream{
		Codecs:    attrs["CODECS"],
		Audio:     attrs["AUDIO"],
		Subtitles: attrs["SUBTITLES"],
	}

	if bwStr, ok := attrs["BANDWIDTH"]; ok {
		if bw, err := strconv.ParseUint(bwStr, 10, 64); err == nil {
			v.Bandwidth = bw
		}
	}
	if abwStr, ok := attrs["AVERAGE-BANDWIDTH"]; ok {
		if abw, err := strconv.ParseUint(abwStr, 10, 64); err == nil {
			v.AverageBandwidth = abw
		}
	}
	if resStr, ok := attrs["RESOLUTION"]; ok {
		v.Resolution = resStr
		parts := strings.Split(resStr, "x")
		if len(parts) == 2 {
			w, _ := strconv.Atoi(parts[0])
			h, _ := strconv.Atoi(parts[1])
			v.Width = w
			v.Height = h
		}
	}
	if fpsStr, ok := attrs["FRAME-RATE"]; ok {
		if fps, err := strconv.ParseFloat(fpsStr, 64); err == nil {
			v.FrameRate = fps
		}
	}

	return v
}

func parseMediaAttributes(attrStr string) MediaRendition {
	attrs := parseAttributes(attrStr)
	return MediaRendition{
		Type:       attrs["TYPE"],
		GroupID:    attrs["GROUP-ID"],
		Name:       attrs["NAME"],
		Language:   attrs["LANGUAGE"],
		Default:    strings.ToUpper(attrs["DEFAULT"]) == "YES",
		AutoSelect: strings.ToUpper(attrs["AUTOSELECT"]) == "YES",
		URI:        attrs["URI"],
	}
}
