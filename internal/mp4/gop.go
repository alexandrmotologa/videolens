package mp4

import (
	"math"
)

// GOPInterval represents a single Group of Pictures between consecutive keyframes.
type GOPInterval struct {
	Index           int     `json:"index"`
	StartFrame      uint32  `json:"startFrame"`
	DurationFrames  uint32  `json:"durationFrames"`
	DurationSeconds float64 `json:"durationSeconds"`
	StartSeconds    float64 `json:"startSeconds"`
}

// GOPProfile summarizes the keyframe distribution and cadence stability.
type GOPProfile struct {
	TotalKeyframes  int           `json:"totalKeyframes"`
	TotalFrames     int           `json:"totalFrames"`
	CadenceType     string        `json:"cadenceType"` // "FIXED", "VARIABLE", "SINGLE_KEYFRAME"
	MeanGOPDuration float64       `json:"meanGOPDuration"`
	MinGOPDuration  float64       `json:"minGOPDuration"`
	MaxGOPDuration  float64       `json:"maxGOPDuration"`
	IsFixedGOP      bool          `json:"isFixedGOP"`
	SpikeCount      int           `json:"spikeCount"`
	Intervals       []GOPInterval `json:"intervals"`
	Recommendation  string        `json:"recommendation"`
}

// AnalyzeGOPCadence computes GOP structure from sync samples (stss) and time-to-sample (stts).
func AnalyzeGOPCadence(keyframes []uint32, totalFrames int, timescale uint32, defaultFrameDuration uint32) *GOPProfile {
	profile := &GOPProfile{
		TotalKeyframes: len(keyframes),
		TotalFrames:    totalFrames,
		Intervals:      make([]GOPInterval, 0),
		MinGOPDuration: math.MaxFloat64,
	}

	if len(keyframes) == 0 {
		profile.CadenceType = "SINGLE_KEYFRAME"
		profile.Recommendation = "No sync sample table (stss) found. Stream may use all intra frames or fragmented MP4 indexing."
		profile.MinGOPDuration = 0
		return profile
	}

	if len(keyframes) == 1 {
		profile.CadenceType = "SINGLE_KEYFRAME"
		profile.MeanGOPDuration = float64(totalFrames*int(defaultFrameDuration)) / float64(timescale)
		profile.MinGOPDuration = profile.MeanGOPDuration
		profile.MaxGOPDuration = profile.MeanGOPDuration
		profile.Recommendation = "Single initial keyframe detected. Video lacks periodic seekable recovery points."
		return profile
	}

	if timescale == 0 {
		timescale = 1000
	}
	if defaultFrameDuration == 0 {
		defaultFrameDuration = timescale / 30 // 30fps default
	}

	var sumDur float64

	for i := 0; i < len(keyframes)-1; i++ {
		curFrame := keyframes[i]
		nextFrame := keyframes[i+1]
		framesInGOP := nextFrame - curFrame

		startSec := float64((curFrame-1)*defaultFrameDuration) / float64(timescale)
		durSec := float64(framesInGOP*defaultFrameDuration) / float64(timescale)

		sumDur += durSec
		if durSec < profile.MinGOPDuration {
			profile.MinGOPDuration = durSec
		}
		if durSec > profile.MaxGOPDuration {
			profile.MaxGOPDuration = durSec
		}

		profile.Intervals = append(profile.Intervals, GOPInterval{
			Index:           i + 1,
			StartFrame:      curFrame,
			DurationFrames:  framesInGOP,
			DurationSeconds: durSec,
			StartSeconds:    startSec,
		})
	}

	if len(profile.Intervals) > 0 {
		profile.MeanGOPDuration = sumDur / float64(len(profile.Intervals))

		// Check if fixed GOP
		isFixed := true
		for _, interval := range profile.Intervals {
			diff := math.Abs(interval.DurationSeconds - profile.MeanGOPDuration)
			if diff > 0.15 {
				isFixed = false
			}
			if interval.DurationSeconds > profile.MeanGOPDuration*1.5 {
				profile.SpikeCount++
			}
		}

		profile.IsFixedGOP = isFixed
		if isFixed {
			profile.CadenceType = "FIXED"
			profile.Recommendation = "Fixed GOP structure verified. Consistent keyframe intervals allow seamless ABR variant switching."
		} else {
			profile.CadenceType = "VARIABLE"
			profile.Recommendation = "Variable GOP detected (scene cut encoding). Ensure segment boundaries in HLS/DASH align exactly with keyframe intervals."
		}
	}

	return profile
}
