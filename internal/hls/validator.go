package hls

import (
	"bytes"
	"fmt"
	"math"
	"regexp"
	"strings"
)

// IssueSeverity levels for manifest diagnostics.
type IssueSeverity string

const (
	SeverityError   IssueSeverity = "ERROR"
	SeverityWarning IssueSeverity = "WARNING"
	SeverityInfo    IssueSeverity = "INFO"
)

// LintIssue represents a specific violation or warning in a manifest.
type LintIssue struct {
	Code           string        `json:"code"`
	Severity       IssueSeverity `json:"severity"`
	Line           int           `json:"line"`
	Message        string        `json:"message"`
	Recommendation string        `json:"recommendation"`
}

// DurationMetrics holds statistical variance metrics for segment durations.
type DurationMetrics struct {
	MinDuration  float64 `json:"minDuration"`
	MaxDuration  float64 `json:"maxDuration"`
	MeanDuration float64 `json:"meanDuration"`
	Variance     float64 `json:"variance"`
	StdDev       float64 `json:"stdDev"`
	DriftPercent float64 `json:"driftPercent"`
}

// HLSLintResult contains complete diagnostic outcome for an HLS playlist.
type HLSLintResult struct {
	IsMaster           bool             `json:"isMaster"`
	Version            int              `json:"version"`
	TargetDuration     float64          `json:"targetDuration,omitempty"`
	MediaSequence      int64            `json:"mediaSequence,omitempty"`
	TotalDuration      float64          `json:"totalDuration,omitempty"`
	SegmentCount       int              `json:"segmentCount"`
	DiscontinuityCount int              `json:"discontinuityCount"`
	PlaylistType       string           `json:"playlistType,omitempty"`
	HasEndList         bool             `json:"hasEndList"`
	IsLowLatency       bool             `json:"isLowLatency"`
	PartTarget         float64          `json:"partTarget,omitempty"`
	TotalPartsCount    int              `json:"totalPartsCount,omitempty"`
	DurationMetrics    *DurationMetrics `json:"durationMetrics,omitempty"`
	Variants           []VariantStream  `json:"variants,omitempty"`
	Renditions         []MediaRendition `json:"renditions,omitempty"`
	Segments           []Segment        `json:"segments,omitempty"`
	RawLines           []string         `json:"rawLines"`
	Issues             []LintIssue      `json:"issues"`
	HealthScore        int              `json:"healthScore"`
}

var codecRegex = regexp.MustCompile(`^[a-zA-Z0-9_\-]+(\.[a-zA-Z0-9_\-]+)*$`)

// LintHLS parses and validates an HLS playlist according to RFC 8216 rules.
func LintHLS(content string) (*HLSLintResult, error) {
	trimmed := strings.TrimSpace(content)
	if !strings.HasPrefix(trimmed, "#EXTM3U") {
		return &HLSLintResult{
			Issues: []LintIssue{
				{
					Code:           "ERR_MISSING_EXTM3U",
					Severity:       SeverityError,
					Line:           1,
					Message:        "Manifest does not start with mandatory #EXTM3U header.",
					Recommendation: "Prepend #EXTM3U to line 1 of the playlist file.",
				},
			},
			HealthScore: 0,
			RawLines:    strings.Split(content, "\n"),
		}, nil
	}

	isMaster := strings.Contains(content, "#EXT-X-STREAM-INF")

	if isMaster {
		return lintMasterPlaylist(content)
	}
	return lintMediaPlaylist(content)
}

func lintMasterPlaylist(content string) (*HLSLintResult, error) {
	master, err := ParseMasterPlaylist(bytes.NewBufferString(content))
	if err != nil {
		return nil, err
	}

	result := &HLSLintResult{
		IsMaster:    true,
		Version:     master.Version,
		Variants:    master.Variants,
		Renditions:  master.Renditions,
		RawLines:    master.RawLines,
		Issues:      make([]LintIssue, 0),
		HealthScore: 100,
	}

	if len(master.Variants) == 0 {
		result.Issues = append(result.Issues, LintIssue{
			Code:           "ERR_NO_VARIANTS",
			Severity:       SeverityError,
			Line:           1,
			Message:        "Master playlist contains no #EXT-X-STREAM-INF variant streams.",
			Recommendation: "Include at least one #EXT-X-STREAM-INF tag defining a stream rendition.",
		})
	}

	audioGroups := make(map[string]bool)
	subtitleGroups := make(map[string]bool)
	for _, rend := range master.Renditions {
		if rend.Type == "AUDIO" {
			audioGroups[rend.GroupID] = true
		} else if rend.Type == "SUBTITLES" {
			subtitleGroups[rend.GroupID] = true
		}
	}

	for _, v := range master.Variants {
		if v.Bandwidth == 0 {
			result.Issues = append(result.Issues, LintIssue{
				Code:           "ERR_ZERO_BANDWIDTH",
				Severity:       SeverityError,
				Line:           v.LineNumber,
				Message:        "Variant has BANDWIDTH=0.",
				Recommendation: "Specify positive integer peak bitrate in bits per second.",
			})
		}

		if v.Audio != "" && !audioGroups[v.Audio] {
			result.Issues = append(result.Issues, LintIssue{
				Code:           "WARN_UNMATCHED_AUDIO_GROUP",
				Severity:       SeverityWarning,
				Line:           v.LineNumber,
				Message:        fmt.Sprintf("Variant references audio group %q, but no matching #EXT-X-MEDIA:TYPE=AUDIO with GROUP-ID=%q exists.", v.Audio, v.Audio),
				Recommendation: "Add the corresponding #EXT-X-MEDIA audio rendition definition.",
			})
		}

		if v.Subtitles != "" && !subtitleGroups[v.Subtitles] {
			result.Issues = append(result.Issues, LintIssue{
				Code:           "WARN_UNMATCHED_SUBTITLE_GROUP",
				Severity:       SeverityWarning,
				Line:           v.LineNumber,
				Message:        fmt.Sprintf("Variant references subtitles group %q, but no matching #EXT-X-MEDIA:TYPE=SUBTITLES exists.", v.Subtitles),
				Recommendation: "Add the corresponding #EXT-X-MEDIA subtitles rendition definition.",
			})
		}

		if v.Codecs != "" {
			for _, c := range strings.Split(v.Codecs, ",") {
				c = strings.TrimSpace(c)
				if !codecRegex.MatchString(c) {
					result.Issues = append(result.Issues, LintIssue{
						Code:           "WARN_INVALID_CODEC_SYNTAX",
						Severity:       SeverityWarning,
						Line:           v.LineNumber,
						Message:        fmt.Sprintf("Codec %q does not match RFC 6381 syntax.", c),
						Recommendation: "Verify codec identifier matches standards such as avc1.4d401f or mp4a.40.2.",
					})
				}
			}
		}
	}

	computeHealthScore(result)
	return result, nil
}

func lintMediaPlaylist(content string) (*HLSLintResult, error) {
	media, err := ParseMediaPlaylist(bytes.NewBufferString(content))
	if err != nil {
		return nil, err
	}

	result := &HLSLintResult{
		IsMaster:        false,
		Version:         media.Version,
		TargetDuration:  media.TargetDuration,
		MediaSequence:   media.MediaSequence,
		PlaylistType:    media.PlaylistType,
		HasEndList:      media.HasEndList,
		IsLowLatency:    media.IsLowLatency,
		PartTarget:      media.PartTarget,
		TotalPartsCount: media.TotalPartsCount,
		TotalDuration:   media.TotalDuration,
		SegmentCount:    len(media.Segments),
		Segments:        media.Segments,
		RawLines:        media.RawLines,
		Issues:          make([]LintIssue, 0),
		HealthScore:     100,
	}

	// Rule: LL-HLS checks
	if media.IsLowLatency {
		if media.PartTarget <= 0 {
			result.Issues = append(result.Issues, LintIssue{
				Code:           "ERR_MISSING_PART_TARGET",
				Severity:       SeverityError,
				Line:           1,
				Message:        "Low-Latency HLS playlist contains partial segments (#EXT-X-PART) but lacks mandatory #EXT-X-PART-INF:PART-TARGET.",
				Recommendation: "Declare #EXT-X-PART-INF:PART-TARGET with the maximum part duration in seconds.",
			})
		} else {
			for _, seg := range media.Segments {
				for _, part := range seg.Parts {
					if part.Duration > media.PartTarget+0.05 {
						result.Issues = append(result.Issues, LintIssue{
							Code:           "ERR_PART_TARGET_EXCEEDED",
							Severity:       SeverityError,
							Line:           part.LineNumber,
							Message:        fmt.Sprintf("Partial segment duration (%.3fs) exceeds PART-TARGET (%.2fs).", part.Duration, media.PartTarget),
							Recommendation: "Reduce low-latency chunk generation interval on the encoder.",
						})
					}
				}
			}
		}
	}

	if media.TargetDuration <= 0 {
		result.Issues = append(result.Issues, LintIssue{
			Code:           "ERR_MISSING_TARGETDURATION",
			Severity:       SeverityError,
			Line:           1,
			Message:        "Missing or non-positive #EXT-X-TARGETDURATION tag.",
			Recommendation: "Specify #EXT-X-TARGETDURATION with the ceiling duration of any segment in seconds.",
		})
	}

	if strings.ToUpper(media.PlaylistType) == "VOD" && !media.HasEndList {
		result.Issues = append(result.Issues, LintIssue{
			Code:           "ERR_MISSING_ENDLIST",
			Severity:       SeverityError,
			Line:           len(media.RawLines),
			Message:        "Playlist is marked as VOD (#EXT-X-PLAYLIST-TYPE:VOD) but lacks trailing #EXT-X-ENDLIST tag.",
			Recommendation: "Append #EXT-X-ENDLIST to terminate the VOD playlist.",
		})
	}

	var sumDur float64
	minDur := math.MaxFloat64
	maxDur := 0.0
	discontinuityCount := 0

	for _, seg := range media.Segments {
		sumDur += seg.Duration
		if seg.Duration < minDur {
			minDur = seg.Duration
		}
		if seg.Duration > maxDur {
			maxDur = seg.Duration
		}
		if seg.Discontinuity {
			discontinuityCount++
		}

		if media.TargetDuration > 0 {
			// RFC 8216 §4.3.3.1: Rounded to nearest integer MUST be <= targetDuration
			roundedDur := math.Round(seg.Duration)
			if seg.Duration > media.TargetDuration+0.5 {
				result.Issues = append(result.Issues, LintIssue{
					Code:           "ERR_TARGET_DURATION_EXCEEDED",
					Severity:       SeverityError,
					Line:           seg.LineNumber,
					Message:        fmt.Sprintf("Segment %d duration (%.3fs) exceeds #EXT-X-TARGETDURATION (%.1fs) by > 0.5s.", seg.Index, seg.Duration, media.TargetDuration),
					Recommendation: "Increase #EXT-X-TARGETDURATION or re-encode video with shorter keyframe intervals.",
				})
			} else if roundedDur > media.TargetDuration {
				result.Issues = append(result.Issues, LintIssue{
					Code:           "WARN_TARGET_DURATION_ROUNDING",
					Severity:       SeverityWarning,
					Line:           seg.LineNumber,
					Message:        fmt.Sprintf("Segment %d duration (%.3fs) rounds to %.0fs which exceeds #EXT-X-TARGETDURATION (%.1fs).", seg.Index, seg.Duration, roundedDur, media.TargetDuration),
					Recommendation: "Align target duration to ceiling value of segment durations.",
				})
			}
		}
	}

	result.DiscontinuityCount = discontinuityCount

	// Calculate drift across segments (excluding the final tail segment if shorter on VOD)
	n := len(media.Segments)
	if n > 1 {
		evalSegments := media.Segments
		if media.HasEndList && n >= 3 && media.Segments[n-1].Duration < media.Segments[n-2].Duration*0.8 {
			// Tail segment of VOD is expected to be short, evaluate steady-state segments for drift
			evalSegments = media.Segments[:n-1]
		}

		evalCount := float64(len(evalSegments))
		var evalSum float64
		for _, s := range evalSegments {
			evalSum += s.Duration
		}
		meanDur := evalSum / evalCount

		var sumSqDiff float64
		for _, s := range evalSegments {
			diff := s.Duration - meanDur
			sumSqDiff += diff * diff
		}
		variance := sumSqDiff / evalCount
		stdDev := math.Sqrt(variance)
		driftPercent := 0.0
		if meanDur > 0 {
			driftPercent = (stdDev / meanDur) * 100.0
		}

		result.DurationMetrics = &DurationMetrics{
			MinDuration:  minDur,
			MaxDuration:  maxDur,
			MeanDuration: meanDur,
			Variance:     variance,
			StdDev:       stdDev,
			DriftPercent: driftPercent,
		}

		if driftPercent > 15.0 {
			result.Issues = append(result.Issues, LintIssue{
				Code:           "WARN_DURATION_DRIFT",
				Severity:       SeverityWarning,
				Line:           1,
				Message:        fmt.Sprintf("High segment duration drift detected (drift=%.1f%%, stdDev=%.2fs, min=%.2fs, max=%.2fs).", driftPercent, stdDev, minDur, maxDur),
				Recommendation: "Ensure video encoder uses fixed GOP size (keyframe intervals) to produce consistent segment lengths.",
			})
		}
	}

	computeHealthScore(result)
	return result, nil
}

func computeHealthScore(result *HLSLintResult) {
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
}
