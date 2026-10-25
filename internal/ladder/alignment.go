package ladder

import (
	"fmt"
	"math"
)

// SegmentTiming holds timing information for a segment in a variant track.
type SegmentTiming struct {
	Index    int     `json:"index"`
	Duration float64 `json:"duration"`
	PTS      float64 `json:"pts"` // Presentation timestamp relative to start
}

// VariantTrack represents an individual variant stream's segment sequence for alignment comparison.
type VariantTrack struct {
	ID         string          `json:"id"`
	Resolution string          `json:"resolution"`
	Bitrate    uint64          `json:"bitrate"`
	Segments   []SegmentTiming `json:"segments"`
}

// AlignmentDiscrepancy flags a specific segment where variant timestamps drift apart.
type AlignmentDiscrepancy struct {
	SegmentIndex int     `json:"segmentIndex"`
	VariantA     string  `json:"variantA"`
	VariantB     string  `json:"variantB"`
	PtsA         float64 `json:"ptsA"`
	PtsB         float64 `json:"ptsB"`
	DriftMs      float64 `json:"driftMs"`
	Severity     string  `json:"severity"` // "ERROR", "WARNING"
	Message      string  `json:"message"`
}

// AlignmentMatrix encapsulates the cross-variant alignment audit.
type AlignmentMatrix struct {
	VariantCount  int                    `json:"variantCount"`
	SegmentCount  int                    `json:"segmentCount"`
	Aligned       bool                   `json:"aligned"`
	MaxDriftMs    float64                `json:"maxDriftMs"`
	Score         int                    `json:"score"` // 0 - 100
	Discrepancies []AlignmentDiscrepancy `json:"discrepancies"`
	Summary       string                 `json:"summary"`
}

// CompareVariantAlignment audits keyframe and segment boundary alignment across two or more variants.
func CompareVariantAlignment(variants []VariantTrack) *AlignmentMatrix {
	matrix := &AlignmentMatrix{
		VariantCount:  len(variants),
		Discrepancies: make([]AlignmentDiscrepancy, 0),
		Score:         100,
		Aligned:       true,
	}

	if len(variants) < 2 {
		matrix.Summary = "At least two variant streams are required for alignment comparison."
		return matrix
	}

	// 1. Check segment count parity
	baseCount := len(variants[0].Segments)
	matrix.SegmentCount = baseCount

	for i := 1; i < len(variants); i++ {
		curCount := len(variants[i].Segments)
		if curCount != baseCount {
			matrix.Aligned = false
			matrix.Score -= 20
			matrix.Discrepancies = append(matrix.Discrepancies, AlignmentDiscrepancy{
				SegmentIndex: -1,
				VariantA:     variants[0].ID,
				VariantB:     variants[i].ID,
				Severity:     "ERROR",
				Message:      fmt.Sprintf("Segment count mismatch: %s has %d segments while %s has %d segments.", variants[0].ID, baseCount, variants[i].ID, curCount),
			})
		}
	}

	// Compare PTS alignment across matching segment indexes
	minCommonCount := baseCount
	for _, v := range variants {
		if len(v.Segments) < minCommonCount {
			minCommonCount = len(v.Segments)
		}
	}

	for segIdx := 0; segIdx < minCommonCount; segIdx++ {
		for i := 0; i < len(variants); i++ {
			for j := i + 1; j < len(variants); j++ {
				vA := variants[i]
				vB := variants[j]

				ptsA := vA.Segments[segIdx].PTS
				ptsB := vB.Segments[segIdx].PTS

				driftMs := math.Abs(ptsA-ptsB) * 1000.0
				if driftMs > matrix.MaxDriftMs {
					matrix.MaxDriftMs = driftMs
				}

				if driftMs > 100.0 {
					matrix.Aligned = false
					sev := "WARNING"
					if driftMs > 250.0 {
						sev = "ERROR"
						matrix.Score -= 15
					} else {
						matrix.Score -= 5
					}

					matrix.Discrepancies = append(matrix.Discrepancies, AlignmentDiscrepancy{
						SegmentIndex: segIdx,
						VariantA:     vA.ID,
						VariantB:     vB.ID,
						PtsA:         ptsA,
						PtsB:         ptsB,
						DriftMs:      driftMs,
						Severity:     sev,
						Message:      fmt.Sprintf("Segment #%d timestamp drift is %.1f ms between %s and %s.", segIdx, driftMs, vA.ID, vB.ID),
					})
				}
			}
		}
	}

	if matrix.Score < 0 {
		matrix.Score = 0
	}

	if matrix.Aligned && len(matrix.Discrepancies) == 0 {
		matrix.Summary = fmt.Sprintf("Variants are perfectly aligned. Max boundary drift is %.2f ms across %d segments.", matrix.MaxDriftMs, minCommonCount)
	} else {
		matrix.Summary = fmt.Sprintf("Variants have timestamp misalignment (max drift: %.1f ms). Players will experience buffer freezes during ABR switching.", matrix.MaxDriftMs)
	}

	return matrix
}
