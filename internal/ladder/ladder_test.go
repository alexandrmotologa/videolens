package ladder

import (
	"testing"
)

func TestOptimalLadder(t *testing.T) {
	rungs := []LadderRung{
		{ID: "1080p", Resolution: "1920x1080", Width: 1920, Height: 1080, FPS: 30, Bitrate: 5500000, Codec: "avc1.640028"},
		{ID: "720p", Resolution: "1280x720", Width: 1280, Height: 720, FPS: 30, Bitrate: 3000000, Codec: "avc1.4d401f"},
		{ID: "480p", Resolution: "854x480", Width: 854, Height: 480, FPS: 30, Bitrate: 1600000, Codec: "avc1.4d401f"},
		{ID: "360p", Resolution: "640x360", Width: 640, Height: 360, FPS: 30, Bitrate: 800000, Codec: "avc1.4d401e"},
	}

	analysis := AnalyzeLadder(rungs)

	if analysis.HealthScore < 90 {
		t.Errorf("expected high health score for balanced ladder, got %d", analysis.HealthScore)
	}

	if len(analysis.Rungs) != 4 {
		t.Fatalf("expected 4 rungs, got %d", len(analysis.Rungs))
	}

	// Verify sorting (lowest bitrate first)
	if analysis.Rungs[0].ID != "360p" || analysis.Rungs[3].ID != "1080p" {
		t.Errorf("expected rungs sorted ascending by bitrate, got first=%s last=%s", analysis.Rungs[0].ID, analysis.Rungs[3].ID)
	}

	for _, r := range analysis.Rungs {
		if r.BPPStatus != "OPTIMAL" {
			t.Errorf("expected rung %s to have OPTIMAL BPP, got %s (bpp=%.4f)", r.ID, r.BPPStatus, r.BitsPerPixel)
		}
	}
}

func TestInvertedResolutionAndHarshStep(t *testing.T) {
	rungs := []LadderRung{
		{ID: "tier-1", Resolution: "1920x1080", Width: 1920, Height: 1080, FPS: 30, Bitrate: 1000000, Codec: "avc1.640028"},
		{ID: "tier-2", Resolution: "640x360", Width: 640, Height: 360, FPS: 30, Bitrate: 4000000, Codec: "avc1.4d401e"},
	}

	analysis := AnalyzeLadder(rungs)

	hasInverted := false
	hasHarsh := false
	hasStarved := false

	for _, issue := range analysis.Issues {
		if issue.Code == "ERR_INVERTED_RESOLUTION" {
			hasInverted = true
		}
		if issue.Code == "WARN_HARSH_STEP_RATIO" {
			hasHarsh = true
		}
		if issue.Code == "WARN_STARVED_BITRATE" {
			hasStarved = true
		}
	}

	if !hasInverted {
		t.Error("expected ERR_INVERTED_RESOLUTION")
	}
	if !hasHarsh {
		t.Error("expected WARN_HARSH_STEP_RATIO (4.0x step)")
	}
	if !hasStarved {
		t.Error("expected WARN_STARVED_BITRATE for 1080p at 1Mbps")
	}
	if analysis.HealthScore > 60 {
		t.Errorf("expected low health score, got %d", analysis.HealthScore)
	}
}
