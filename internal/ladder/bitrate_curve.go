package ladder

import (
	"fmt"
	"sort"
	"strings"
)

// LadderRung represents a single stream tier in an adaptive bitrate ladder.
type LadderRung struct {
	ID           string  `json:"id"`
	Resolution   string  `json:"resolution"`
	Width        int     `json:"width"`
	Height       int     `json:"height"`
	FPS          float64 `json:"fps"`
	Bitrate      uint64  `json:"bitrate"`      // bits per second
	BitrateKbps  float64 `json:"bitrateKbps"`  // kilobits per second
	BitrateMbps  float64 `json:"bitrateMbps"`  // megabits per second
	Codec        string  `json:"codec"`
	CodecFamily  string  `json:"codecFamily"`  // "H264", "HEVC", "VP9", "AV1", "OTHER"
	BitsPerPixel float64 `json:"bitsPerPixel"` // BPP
	StepRatio    float64 `json:"stepRatio"`    // Ratio vs previous lower rung
	BPPStatus    string  `json:"bppStatus"`    // "OPTIMAL", "STARVED", "SATURATED"
	Evaluation   string  `json:"evaluation"`
}

// LadderIssue represents a defect or optimization finding in the bitrate ladder.
type LadderIssue struct {
	Code           string `json:"code"`
	Severity       string `json:"severity"` // "ERROR", "WARNING", "INFO"
	RungID         string `json:"rungId,omitempty"`
	Message        string `json:"message"`
	Recommendation string `json:"recommendation"`
}

// LadderAnalysis encapsulates diagnostic metrics and health scoring for an ABR ladder.
type LadderAnalysis struct {
	Rungs           []LadderRung   `json:"rungs"`
	RungCount       int            `json:"rungCount"`
	MinBitrate      uint64         `json:"minBitrate"`
	MaxBitrate      uint64         `json:"maxBitrate"`
	DynamicRange    float64        `json:"dynamicRange"` // Max / Min
	HealthScore     int            `json:"healthScore"`
	Issues          []LadderIssue  `json:"issues"`
	Recommendations []string       `json:"recommendations"`
}

// DetectCodecFamily classifies a codec string into an ABR codec generation.
func DetectCodecFamily(codec string) string {
	c := strings.ToLower(codec)
	if strings.HasPrefix(c, "avc1") || strings.Contains(c, "h264") {
		return "H264"
	}
	if strings.HasPrefix(c, "hvc1") || strings.HasPrefix(c, "hev1") || strings.Contains(c, "h265") || strings.Contains(c, "hevc") {
		return "HEVC"
	}
	if strings.HasPrefix(c, "vp09") || strings.Contains(c, "vp9") {
		return "VP9"
	}
	if strings.HasPrefix(c, "av01") || strings.Contains(c, "av1") {
		return "AV1"
	}
	return "OTHER"
}

// CalculateBPP computes Bits Per Pixel: Bitrate / (Width * Height * FPS).
func CalculateBPP(bitrate uint64, width, height int, fps float64) float64 {
	if width <= 0 || height <= 0 {
		return 0
	}
	if fps <= 0 {
		fps = 30.0 // Default to 30fps if unspecified
	}
	pixelsPerSec := float64(width*height) * fps
	if pixelsPerSec <= 0 {
		return 0
	}
	return float64(bitrate) / pixelsPerSec
}

// EvaluateBPP returns status and text explanation based on codec family.
func EvaluateBPP(bpp float64, codecFamily string) (string, string) {
	if bpp <= 0 {
		return "UNKNOWN", "Insufficient resolution data to calculate BPP."
	}

	switch codecFamily {
	case "H264":
		if bpp < 0.05 {
			return "STARVED", fmt.Sprintf("BPP %.4f is starved for H.264 (optimal: 0.07-0.15). Risk of macroblocking and compression artifacts.", bpp)
		} else if bpp > 0.20 {
			return "SATURATED", fmt.Sprintf("BPP %.4f is saturated for H.264 (optimal: 0.07-0.15). Wastes bandwidth with diminishing visual return.", bpp)
		}
		return "OPTIMAL", fmt.Sprintf("BPP %.4f is within optimal H.264 range (0.07-0.15).", bpp)

	case "HEVC", "VP9":
		if bpp < 0.03 {
			return "STARVED", fmt.Sprintf("BPP %.4f is starved for %s (optimal: 0.04-0.09).", bpp, codecFamily)
		} else if bpp > 0.12 {
			return "SATURATED", fmt.Sprintf("BPP %.4f is saturated for %s (optimal: 0.04-0.09).", bpp, codecFamily)
		}
		return "OPTIMAL", fmt.Sprintf("BPP %.4f is within optimal %s range (0.04-0.09).", bpp, codecFamily)

	case "AV1":
		if bpp < 0.025 {
			return "STARVED", fmt.Sprintf("BPP %.4f is starved for AV1 (optimal: 0.03-0.07).", bpp)
		} else if bpp > 0.10 {
			return "SATURATED", fmt.Sprintf("BPP %.4f is saturated for AV1 (optimal: 0.03-0.07).", bpp)
		}
		return "OPTIMAL", fmt.Sprintf("BPP %.4f is within optimal AV1 range (0.03-0.07).", bpp)

	default:
		if bpp < 0.05 {
			return "STARVED", fmt.Sprintf("BPP %.4f is low.", bpp)
		} else if bpp > 0.18 {
			return "SATURATED", fmt.Sprintf("BPP %.4f is high.", bpp)
		}
		return "OPTIMAL", fmt.Sprintf("BPP %.4f is acceptable.", bpp)
	}
}

// AnalyzeLadder evaluates an adaptive bitrate ladder from an input slice of rungs.
func AnalyzeLadder(inputRungs []LadderRung) *LadderAnalysis {
	if len(inputRungs) == 0 {
		return &LadderAnalysis{
			HealthScore: 0,
			Issues: []LadderIssue{
				{
					Code:           "ERR_EMPTY_LADDER",
					Severity:       "ERROR",
					Message:        "Bitrate ladder is empty.",
					Recommendation: "Supply at least one video variant stream.",
				},
			},
		}
	}

	// Sort rungs ascending by bitrate
	rungs := make([]LadderRung, len(inputRungs))
	copy(rungs, inputRungs)
	sort.Slice(rungs, func(i, j int) bool {
		return rungs[i].Bitrate < rungs[j].Bitrate
	})

	analysis := &LadderAnalysis{
		RungCount:       len(rungs),
		MinBitrate:      rungs[0].Bitrate,
		MaxBitrate:      rungs[len(rungs)-1].Bitrate,
		Issues:          make([]LadderIssue, 0),
		Recommendations: make([]string, 0),
		HealthScore:     100,
	}

	if analysis.MinBitrate > 0 {
		analysis.DynamicRange = float64(analysis.MaxBitrate) / float64(analysis.MinBitrate)
	}

	for i := range rungs {
		r := &rungs[i]
		if r.FPS <= 0 {
			r.FPS = 30.0
		}
		if r.Resolution == "" && r.Width > 0 && r.Height > 0 {
			r.Resolution = fmt.Sprintf("%dx%d", r.Width, r.Height)
		}
		r.BitrateKbps = float64(r.Bitrate) / 1000.0
		r.BitrateMbps = float64(r.Bitrate) / 1000000.0

		r.CodecFamily = DetectCodecFamily(r.Codec)
		r.BitsPerPixel = CalculateBPP(r.Bitrate, r.Width, r.Height, r.FPS)
		r.BPPStatus, r.Evaluation = EvaluateBPP(r.BitsPerPixel, r.CodecFamily)

		if i > 0 && rungs[i-1].Bitrate > 0 {
			r.StepRatio = float64(r.Bitrate) / float64(rungs[i-1].Bitrate)
		} else {
			r.StepRatio = 1.0
		}

		// Check BPP status issues
		if r.BPPStatus == "STARVED" {
			analysis.Issues = append(analysis.Issues, LadderIssue{
				Code:           "WARN_STARVED_BITRATE",
				Severity:       "WARNING",
				RungID:         r.ID,
				Message:        fmt.Sprintf("Variant %s (%s, %.1f Mbps) has insufficient bitrate (BPP=%.4f).", r.ID, r.Resolution, r.BitrateMbps, r.BitsPerPixel),
				Recommendation: fmt.Sprintf("Increase bitrate for %s or downscale resolution.", r.Resolution),
			})
		} else if r.BPPStatus == "SATURATED" {
			analysis.Issues = append(analysis.Issues, LadderIssue{
				Code:           "WARN_SATURATED_BITRATE",
				Severity:       "WARNING",
				RungID:         r.ID,
				Message:        fmt.Sprintf("Variant %s (%s, %.1f Mbps) has surplus bitrate (BPP=%.4f).", r.ID, r.Resolution, r.BitrateMbps, r.BitsPerPixel),
				Recommendation: fmt.Sprintf("Lower bitrate for %s to conserve CDN bandwidth.", r.Resolution),
			})
		}

		// Check step ratio against previous rung
		if i > 0 {
			prev := &rungs[i-1]
			// Inverted tier check: resolution smaller but bitrate higher
			curPixels := r.Width * r.Height
			prevPixels := prev.Width * prev.Height

			if curPixels > 0 && prevPixels > 0 && curPixels < prevPixels {
				analysis.Issues = append(analysis.Issues, LadderIssue{
					Code:           "ERR_INVERTED_RESOLUTION",
					Severity:       "ERROR",
					RungID:         r.ID,
					Message:        fmt.Sprintf("Inverted resolution: variant %s (%s) has smaller pixel area than lower-bitrate variant %s (%s).", r.ID, r.Resolution, prev.ID, prev.Resolution),
					Recommendation: "Align resolution ascending with bitrate.",
				})
			}

			// Harsh step ratio check (> 2.5x)
			if r.StepRatio > 2.5 {
				analysis.Issues = append(analysis.Issues, LadderIssue{
					Code:           "WARN_HARSH_STEP_RATIO",
					Severity:       "WARNING",
					RungID:         r.ID,
					Message:        fmt.Sprintf("Harsh step ratio (%.2fx) between %s (%.1f Mbps) and %s (%.1f Mbps).", r.StepRatio, prev.ID, prev.BitrateMbps, r.ID, r.BitrateMbps),
					Recommendation: "Insert an intermediate bitrate rung or reduce step ratio to 1.5x-2.0x.",
				})
			} else if r.StepRatio < 1.3 && r.StepRatio > 1.0 {
				analysis.Issues = append(analysis.Issues, LadderIssue{
					Code:           "WARN_REDUNDANT_STEP_RATIO",
					Severity:       "INFO",
					RungID:         r.ID,
					Message:        fmt.Sprintf("Close step ratio (%.2fx) between %s and %s.", r.StepRatio, prev.ID, r.ID),
					Recommendation: "Consider consolidating these tiers to eliminate redundant encoding compute.",
				})
			}
		}
	}

	analysis.Rungs = rungs

	// Calculate overall health score
	score := 100
	for _, issue := range analysis.Issues {
		switch issue.Severity {
		case "ERROR":
			score -= 25
		case "WARNING":
			score -= 10
		case "INFO":
			score -= 3
		}
	}
	if score < 0 {
		score = 0
	}
	analysis.HealthScore = score

	// Generate synthesis recommendations
	if analysis.DynamicRange < 3.0 && len(rungs) > 1 {
		analysis.Recommendations = append(analysis.Recommendations, "Dynamic range is narrow. Consider adding a low-bitrate rung (< 600 kbps) for constrained cellular networks.")
	}
	if len(rungs) < 3 {
		analysis.Recommendations = append(analysis.Recommendations, "Ladder has fewer than 3 variants. Adding 360p, 720p, and 1080p steps provides better adaptation across clients.")
	}
	if len(analysis.Issues) == 0 {
		analysis.Recommendations = append(analysis.Recommendations, "Bitrate ladder follows recommended resolution steps, BPP targets, and scaling intervals.")
	}

	return analysis
}
