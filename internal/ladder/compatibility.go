package ladder

import (
	"fmt"
	"strconv"
	"strings"
)

// DeviceCompatibility evaluates client playback readiness across major device ecosystems.
type DeviceCompatibility struct {
	CodecString  string   `json:"codecString"`
	ProfileName  string   `json:"profileName"`
	LevelNumber  float64  `json:"levelNumber"`
	IOS          bool     `json:"ios"`
	Android      bool     `json:"android"`
	SmartTV      bool     `json:"smartTv"`
	Roku         bool     `json:"roku"`
	DesktopWeb   bool     `json:"desktopWeb"`
	LegacyMobile bool     `json:"legacyMobile"`
	Notes        []string `json:"notes"`
}

// EvaluateCodecCompatibility decodes RFC 6381 codec strings and evaluates hardware compatibility.
func EvaluateCodecCompatibility(codec string) *DeviceCompatibility {
	c := strings.ToLower(strings.TrimSpace(codec))
	comp := &DeviceCompatibility{
		CodecString: codec,
		DesktopWeb:  true,
		IOS:         true,
		Android:     true,
		SmartTV:     true,
		Roku:        true,
		Notes:       make([]string, 0),
	}

	// 1. Check H.264 (avc1.PPCCLL)
	if strings.HasPrefix(c, "avc1") {
		parts := strings.Split(c, ".")
		if len(parts) >= 2 && len(parts[1]) == 6 {
			hexProfile := parts[1][0:2]
			hexLevel := parts[1][4:6]

			levelDec, _ := strconv.ParseInt(hexLevel, 16, 64)
			comp.LevelNumber = float64(levelDec) / 10.0

			switch hexProfile {
			case "42":
				comp.ProfileName = "Baseline Profile"
				comp.LegacyMobile = true
				comp.Notes = append(comp.Notes, "Universal baseline support across older mobile devices.")
			case "4d":
				comp.ProfileName = "Main Profile"
				comp.LegacyMobile = false
				comp.Notes = append(comp.Notes, "Standard broadcast profile supported by modern smartphones and web browsers.")
			case "64":
				comp.ProfileName = "High Profile"
				comp.LegacyMobile = false
				comp.Notes = append(comp.Notes, "High compression efficiency. Supported by modern devices.")
			default:
				comp.ProfileName = fmt.Sprintf("Profile 0x%s", hexProfile)
			}

			// Level evaluation
			if comp.LevelNumber > 4.2 {
				comp.SmartTV = false
				comp.Notes = append(comp.Notes, fmt.Sprintf("Level %.1f exceeds Level 4.2 ceiling for older Smart TVs and legacy streaming sticks.", comp.LevelNumber))
			}
			if comp.LevelNumber > 5.1 {
				comp.Roku = false
				comp.Notes = append(comp.Notes, fmt.Sprintf("Level %.1f may exceed hardware decode capabilities of budget streaming players.", comp.LevelNumber))
			}
		} else {
			comp.ProfileName = "H.264 (AVC)"
		}
		return comp
	}

	// 2. Check HEVC / H.265 (hvc1 or hev1)
	if strings.HasPrefix(c, "hvc1") || strings.HasPrefix(c, "hev1") {
		comp.ProfileName = "HEVC / H.265 Main"
		comp.LegacyMobile = false
		comp.Notes = append(comp.Notes, "Requires hardware HEVC decoder. Native on Apple devices and modern Android/Smart TVs; limited on older desktop Chrome without hardware acceleration.")
		return comp
	}

	// 3. Check AV1 (av01)
	if strings.HasPrefix(c, "av01") {
		comp.ProfileName = "AV1 Main Profile"
		comp.LegacyMobile = false
		comp.SmartTV = false
		comp.Notes = append(comp.Notes, "Next-generation royalty-free codec. Native on recent Apple (A17+/M3+), modern Android, and Chrome; requires hardware support on 4K TVs.")
		return comp
	}

	// 4. Check AAC audio (mp4a.40.2)
	if strings.HasPrefix(c, "mp4a") {
		comp.ProfileName = "AAC-LC Audio"
		comp.LegacyMobile = true
		comp.Notes = append(comp.Notes, "Universal stereo audio codec supported across all players.")
		return comp
	}

	comp.ProfileName = codec
	return comp
}
