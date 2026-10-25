package mp4

import (
	"bytes"
	"testing"
)

func TestParseFastStartOptimized(t *testing.T) {
	data := GenerateSampleMP4(true)
	reader := bytes.NewReader(data)

	profile, err := ParseMP4(reader, int64(len(data)))
	if err != nil {
		t.Fatalf("ParseMP4 failed: %v", err)
	}

	if profile.FastStart == nil {
		t.Fatal("expected FastStart status to be populated")
	}

	if !profile.FastStart.IsFastStart {
		t.Errorf("expected IsFastStart=true, got false. Message: %s", profile.FastStart.Message)
	}

	if profile.FastStart.StatusBadge != "OPTIMIZED" {
		t.Errorf("expected statusBadge=OPTIMIZED, got %s", profile.FastStart.StatusBadge)
	}

	if len(profile.Tracks) == 0 {
		t.Fatal("expected at least 1 track, got 0")
	}

	track := profile.Tracks[0]
	if track.Type != "video" {
		t.Errorf("expected video track, got %s", track.Type)
	}
	if track.Width != 1920 || track.Height != 1080 {
		t.Errorf("expected 1920x1080, got %.0fx%.0f", track.Width, track.Height)
	}
	if track.KeyframeCount != 5 {
		t.Errorf("expected 5 keyframes, got %d", track.KeyframeCount)
	}
}

func TestParseNotFastStartAndRemediate(t *testing.T) {
	unoptimizedData := GenerateSampleMP4(false)
	reader := bytes.NewReader(unoptimizedData)

	profile, err := ParseMP4(reader, int64(len(unoptimizedData)))
	if err != nil {
		t.Fatalf("ParseMP4 unoptimized failed: %v", err)
	}

	if profile.FastStart.IsFastStart {
		t.Error("expected unoptimized sample to have IsFastStart=false")
	}
	if profile.FastStart.StatusBadge != "NOT_OPTIMIZED" {
		t.Errorf("expected NOT_OPTIMIZED, got %s", profile.FastStart.StatusBadge)
	}

	// Remediate with RelocateMoov
	remediatedBytes, err := RelocateMoov(reader, int64(len(unoptimizedData)))
	if err != nil {
		t.Fatalf("RelocateMoov failed: %v", err)
	}

	// Verify remediated MP4
	remReader := bytes.NewReader(remediatedBytes)
	remProfile, err := ParseMP4(remReader, int64(len(remediatedBytes)))
	if err != nil {
		t.Fatalf("ParseMP4 remediated failed: %v", err)
	}

	if !remProfile.FastStart.IsFastStart {
		t.Errorf("expected remediated file to be FastStart=true, message: %s", remProfile.FastStart.Message)
	}
	if remProfile.FastStart.StatusBadge != "OPTIMIZED" {
		t.Errorf("expected OPTIMIZED, got %s", remProfile.FastStart.StatusBadge)
	}

	// Verify tracks are intact
	if len(remProfile.Tracks) != 1 {
		t.Fatalf("expected 1 track in remediated file, got %d", len(remProfile.Tracks))
	}
	if remProfile.Tracks[0].Width != 1920 || remProfile.Tracks[0].Height != 1080 {
		t.Errorf("expected 1920x1080, got %.0fx%.0f", remProfile.Tracks[0].Width, remProfile.Tracks[0].Height)
	}

	// Verify GOP Cadence is populated
	if remProfile.Tracks[0].GOP == nil {
		t.Fatal("expected GOP profile to be populated")
	}
	if remProfile.Tracks[0].GOP.CadenceType != "FIXED" {
		t.Errorf("expected FIXED GOP, got %s", remProfile.Tracks[0].GOP.CadenceType)
	}
	if remProfile.Tracks[0].GOP.TotalKeyframes != 5 {
		t.Errorf("expected 5 keyframes, got %d", remProfile.Tracks[0].GOP.TotalKeyframes)
	}
}
