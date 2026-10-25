package hls

import (
	"strings"
	"testing"

	"github.com/alexandrmotologa/videolens/internal/fixtures"
)

func TestFixBrokenHLSMedia(t *testing.T) {
	fixRes, err := FixMediaPlaylist(fixtures.BrokenHLSMedia)
	if err != nil {
		t.Fatalf("FixMediaPlaylist failed: %v", err)
	}

	if len(fixRes.FixesApplied) == 0 {
		t.Fatal("expected fixes to be applied to broken HLS media playlist")
	}

	hasTargetFix := false
	hasEndListFix := false

	for _, fix := range fixRes.FixesApplied {
		if fix.Code == "FIX_TARGET_DURATION_OVERFLOW" {
			hasTargetFix = true
		}
		if fix.Code == "FIX_APPEND_ENDLIST" {
			hasEndListFix = true
		}
	}

	if !hasTargetFix {
		t.Error("expected FIX_TARGET_DURATION_OVERFLOW to increase target duration")
	}
	if !hasEndListFix {
		t.Error("expected FIX_APPEND_ENDLIST to terminate VOD playlist")
	}

	// Verify sanitized content contains #EXT-X-TARGETDURATION:9 (since max segment is 8.2s)
	if !strings.Contains(fixRes.SanitizedContent, "#EXT-X-TARGETDURATION:9") {
		t.Errorf("expected sanitized content to contain #EXT-X-TARGETDURATION:9, got:\n%s", fixRes.SanitizedContent)
	}
	if !strings.HasSuffix(strings.TrimSpace(fixRes.SanitizedContent), "#EXT-X-ENDLIST") {
		t.Errorf("expected sanitized content to end with #EXT-X-ENDLIST, got:\n%s", fixRes.SanitizedContent)
	}

	// Sanitized score must be higher than original
	if fixRes.SanitizedScore <= fixRes.OriginalScore {
		t.Errorf("expected sanitized score (%d) > original score (%d)", fixRes.SanitizedScore, fixRes.OriginalScore)
	}
}
