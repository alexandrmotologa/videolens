package hls

import (
	"encoding/xml"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// XML structures for MPEG-DASH MPD
type MPDXML struct {
	XMLName                   xml.Name           `xml:"MPD"`
	Profiles                  string             `xml:"profiles,attr"`
	Type                      string             `xml:"type,attr"`
	MinBufferTime             string             `xml:"minBufferTime,attr"`
	MediaPresentationDuration string             `xml:"mediaPresentationDuration,attr"`
	MaxSegmentDuration        string             `xml:"maxSegmentDuration,attr"`
	Periods                   []PeriodXML        `xml:"Period"`
}

type PeriodXML struct {
	ID             string             `xml:"id,attr"`
	Duration       string             `xml:"duration,attr"`
	AdaptationSets []AdaptationSetXML `xml:"AdaptationSet"`
}

type AdaptationSetXML struct {
	ID                  string              `xml:"id,attr"`
	MimeType            string              `xml:"mimeType,attr"`
	ContentType         string              `xml:"contentType,attr"`
	Codecs              string              `xml:"codecs,attr"`
	SubsegmentAlignment string              `xml:"subsegmentAlignment,attr"`
	Representations     []RepresentationXML `xml:"Representation"`
	SegmentTemplate     *SegmentTemplateXML `xml:"SegmentTemplate"`
}

type RepresentationXML struct {
	ID                string              `xml:"id,attr"`
	Bandwidth         uint64              `xml:"bandwidth,attr"`
	Width             int                 `xml:"width,attr"`
	Height            int                 `xml:"height,attr"`
	FrameRate         string              `xml:"frameRate,attr"`
	Codecs            string              `xml:"codecs,attr"`
	AudioSamplingRate string              `xml:"audioSamplingRate,attr"`
	SegmentTemplate   *SegmentTemplateXML `xml:"SegmentTemplate"`
}

type SegmentTemplateXML struct {
	Timescale      uint32              `xml:"timescale,attr"`
	Media          string              `xml:"media,attr"`
	Initialization string              `xml:"initialization,attr"`
	Duration       uint64              `xml:"duration,attr"`
	Timeline       *SegmentTimelineXML `xml:"SegmentTimeline"`
}

type SegmentTimelineXML struct {
	Segments []SegmentSXML `xml:"S"`
}

type SegmentSXML struct {
	T uint64 `xml:"t,attr"`
	D uint64 `xml:"d,attr"`
	R int    `xml:"r,attr"`
}

// DASHRepresentation summarizes an analyzed representation.
type DASHRepresentation struct {
	ID         string  `json:"id"`
	MimeType   string  `json:"mimeType"`
	Bandwidth  uint64  `json:"bandwidth"`
	Width      int     `json:"width,omitempty"`
	Height     int     `json:"height,omitempty"`
	FrameRate  float64 `json:"frameRate,omitempty"`
	Codecs     string  `json:"codecs,omitempty"`
	SampleRate string  `json:"sampleRate,omitempty"`
}

// DASHAdaptationSet summarizes an analyzed adaptation set.
type DASHAdaptationSet struct {
	ID                  string               `json:"id"`
	ContentType         string               `json:"contentType"`
	MimeType            string               `json:"mimeType"`
	SubsegmentAlignment bool                 `json:"subsegmentAlignment"`
	Representations     []DASHRepresentation `json:"representations"`
}

// DASHLintResult contains outcome of MPEG-DASH manifest analysis.
type DASHLintResult struct {
	Profiles           string              `json:"profiles"`
	Type               string              `json:"type"` // "static" or "dynamic"
	MinBufferTimeSec   float64             `json:"minBufferTimeSec"`
	DurationSec        float64             `json:"durationSec"`
	PeriodCount        int                 `json:"periodCount"`
	AdaptationSets     []DASHAdaptationSet `json:"adaptationSets"`
	TotalVariantsCount int                 `json:"totalVariantsCount"`
	Issues             []LintIssue         `json:"issues"`
	HealthScore        int                 `json:"healthScore"`
	RawLines           []string            `json:"rawLines"`
}

// ParseISODuration parses ISO 8601 duration string like "PT1H20M30.5S" to seconds.
func ParseISODuration(durStr string) float64 {
	if durStr == "" {
		return 0
	}
	r := regexp.MustCompile(`PT(?:(\d+)H)?(?:(\d+)M)?(?:([\d.]+)S)?`)
	m := r.FindStringSubmatch(durStr)
	if m == nil {
		return 0
	}

	var total float64
	if m[1] != "" {
		h, _ := strconv.ParseFloat(m[1], 64)
		total += h * 3600
	}
	if m[2] != "" {
		min, _ := strconv.ParseFloat(m[2], 64)
		total += min * 60
	}
	if m[3] != "" {
		sec, _ := strconv.ParseFloat(m[3], 64)
		total += sec
	}
	return total
}

// LintDASH parses and lints an MPEG-DASH XML manifest.
func LintDASH(content string) (*DASHLintResult, error) {
	var mpd MPDXML
	if err := xml.Unmarshal([]byte(content), &mpd); err != nil {
		return &DASHLintResult{
			Issues: []LintIssue{
				{
					Code:           "ERR_XML_PARSE",
					Severity:       SeverityError,
					Line:           1,
					Message:        fmt.Sprintf("Failed to parse MPEG-DASH XML: %v", err),
					Recommendation: "Verify XML syntax, closing tags, and namespace declarations.",
				},
			},
			HealthScore: 0,
			RawLines:    strings.Split(content, "\n"),
		}, nil
	}

	result := &DASHLintResult{
		Profiles:         mpd.Profiles,
		Type:             mpd.Type,
		MinBufferTimeSec: ParseISODuration(mpd.MinBufferTime),
		DurationSec:      ParseISODuration(mpd.MediaPresentationDuration),
		PeriodCount:      len(mpd.Periods),
		AdaptationSets:   make([]DASHAdaptationSet, 0),
		Issues:           make([]LintIssue, 0),
		HealthScore:      100,
		RawLines:         strings.Split(content, "\n"),
	}

	if result.Type == "" {
		result.Type = "static" // Default to static (VOD)
	}

	if mpd.MinBufferTime == "" {
		result.Issues = append(result.Issues, LintIssue{
			Code:           "WARN_MISSING_MIN_BUFFER_TIME",
			Severity:       SeverityWarning,
			Line:           1,
			Message:        "Mandatory minBufferTime attribute is missing from MPD root.",
			Recommendation: "Specify minBufferTime (e.g. minBufferTime=\"PT2.0S\") to guide player buffer management.",
		})
	}

	for _, period := range mpd.Periods {
		for _, as := range period.AdaptationSets {
			adSet := DASHAdaptationSet{
				ID:                  as.ID,
				ContentType:         as.ContentType,
				MimeType:            as.MimeType,
				SubsegmentAlignment: strings.ToLower(as.SubsegmentAlignment) == "true",
				Representations:     make([]DASHRepresentation, 0),
			}

			if adSet.ContentType == "" {
				if strings.HasPrefix(as.MimeType, "video/") {
					adSet.ContentType = "video"
				} else if strings.HasPrefix(as.MimeType, "audio/") {
					adSet.ContentType = "audio"
				}
			}

			for _, rep := range as.Representations {
				mime := as.MimeType
				codec := rep.Codecs
				if codec == "" {
					codec = as.Codecs
				}

				var fps float64
				if rep.FrameRate != "" {
					if strings.Contains(rep.FrameRate, "/") {
						parts := strings.Split(rep.FrameRate, "/")
						num, _ := strconv.ParseFloat(parts[0], 64)
						den, _ := strconv.ParseFloat(parts[1], 64)
						if den > 0 {
							fps = num / den
						}
					} else {
						fps, _ = strconv.ParseFloat(rep.FrameRate, 64)
					}
				}

				dRep := DASHRepresentation{
					ID:         rep.ID,
					MimeType:   mime,
					Bandwidth:  rep.Bandwidth,
					Width:      rep.Width,
					Height:     rep.Height,
					FrameRate:  fps,
					Codecs:     codec,
					SampleRate: rep.AudioSamplingRate,
				}

				if dRep.Bandwidth == 0 {
					result.Issues = append(result.Issues, LintIssue{
						Code:           "ERR_ZERO_BANDWIDTH",
						Severity:       SeverityError,
						Line:           1,
						Message:        fmt.Sprintf("Representation %q specifies bandwidth=0.", rep.ID),
						Recommendation: "Assign non-zero peak bitrate in bits per second.",
					})
				}

				adSet.Representations = append(adSet.Representations, dRep)
				result.TotalVariantsCount++
			}

			result.AdaptationSets = append(result.AdaptationSets, adSet)
		}
	}

	// Score calculation
	score := 100
	for _, issue := range result.Issues {
		switch issue.Severity {
		case SeverityError:
			score -= 25
		case SeverityWarning:
			score -= 10
		case SeverityInfo:
			score -= 2
		}
	}
	if score < 0 {
		score = 0
	}
	result.HealthScore = score

	return result, nil
}
