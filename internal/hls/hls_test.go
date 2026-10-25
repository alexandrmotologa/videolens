package hls

import (
	"testing"

	"github.com/alexandrmotologa/videolens/internal/fixtures"
)

func TestLintValidHLSMaster(t *testing.T) {
	res, err := LintHLS(fixtures.ValidHLSMaster)
	if err != nil {
		t.Fatalf("LintHLS failed: %v", err)
	}

	if !res.IsMaster {
		t.Error("expected IsMaster=true")
	}
	if len(res.Variants) != 4 {
		t.Errorf("expected 4 variants, got %d", len(res.Variants))
	}
	if len(res.Renditions) != 2 {
		t.Errorf("expected 2 renditions, got %d", len(res.Renditions))
	}
	if res.HealthScore < 90 {
		t.Errorf("expected high health score, got %d", res.HealthScore)
	}
}

func TestLintValidHLSMedia(t *testing.T) {
	res, err := LintHLS(fixtures.ValidHLSMedia)
	if err != nil {
		t.Fatalf("LintHLS failed: %v", err)
	}

	if res.IsMaster {
		t.Error("expected IsMaster=false")
	}
	if res.TargetDuration != 6 {
		t.Errorf("expected TargetDuration=6, got %f", res.TargetDuration)
	}
	if len(res.Segments) != 6 {
		t.Errorf("expected 6 segments, got %d", len(res.Segments))
	}
	if !res.HasEndList {
		t.Error("expected HasEndList=true")
	}
	if res.HealthScore != 100 {
		t.Errorf("expected HealthScore=100, got %d", res.HealthScore)
	}
}

func TestLintBrokenHLSMedia(t *testing.T) {
	res, err := LintHLS(fixtures.BrokenHLSMedia)
	if err != nil {
		t.Fatalf("LintHLS failed: %v", err)
	}

	if res.HealthScore > 50 {
		t.Errorf("expected low health score due to violations, got %d", res.HealthScore)
	}

	hasExceeded := false
	hasDrift := false
	hasEndList := false

	for _, issue := range res.Issues {
		if issue.Code == "ERR_TARGET_DURATION_EXCEEDED" {
			hasExceeded = true
		}
		if issue.Code == "WARN_DURATION_DRIFT" {
			hasDrift = true
		}
		if issue.Code == "ERR_MISSING_ENDLIST" {
			hasEndList = true
		}
	}

	if !hasExceeded {
		t.Error("expected ERR_TARGET_DURATION_EXCEEDED issue")
	}
	if !hasDrift {
		t.Error("expected WARN_DURATION_DRIFT issue")
	}
	if !hasEndList {
		t.Error("expected ERR_MISSING_ENDLIST issue")
	}
}

func TestLintValidDASH(t *testing.T) {
	res, err := LintDASH(fixtures.ValidDASHManifest)
	if err != nil {
		t.Fatalf("LintDASH failed: %v", err)
	}

	if res.MinBufferTimeSec != 2.0 {
		t.Errorf("expected MinBufferTimeSec=2.0, got %f", res.MinBufferTimeSec)
	}
	if len(res.AdaptationSets) != 2 {
		t.Fatalf("expected 2 adaptation sets, got %d", len(res.AdaptationSets))
	}
	if res.TotalVariantsCount != 4 {
		t.Errorf("expected 4 representations across sets, got %d", res.TotalVariantsCount)
	}
	if res.HealthScore != 100 {
		t.Errorf("expected HealthScore=100, got %d", res.HealthScore)
	}
}
