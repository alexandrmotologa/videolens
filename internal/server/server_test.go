package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alexandrmotologa/videolens/internal/fixtures"
	"github.com/alexandrmotologa/videolens/internal/ladder"
	"github.com/alexandrmotologa/videolens/internal/mp4"
)

func TestAPISamples(t *testing.T) {
	srv := NewServer(8850)
	req := httptest.NewRequest(http.MethodGet, "/api/samples", nil)
	w := httptest.NewRecorder()

	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var samples []map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &samples); err != nil {
		t.Fatalf("failed to decode samples: %v", err)
	}

	if len(samples) < 3 {
		t.Errorf("expected at least 3 samples, got %d", len(samples))
	}
}

func TestAPIHLSLint(t *testing.T) {
	srv := NewServer(8850)
	payload := map[string]string{
		"content": fixtures.ValidHLSMaster,
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/hls/lint", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var res map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if res["isMaster"] != true {
		t.Error("expected isMaster=true")
	}
	if res["ladder"] == nil {
		t.Error("expected ladder analysis attached to master playlist response")
	}
}

func TestAPIMP4InspectAndRemediate(t *testing.T) {
	srv := NewServer(8850)

	// 1. Inspect unoptimized sample
	reqBody, _ := json.Marshal(map[string]string{"sample": "unoptimized"})
	req := httptest.NewRequest(http.MethodPost, "/api/mp4/inspect", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	srv.Handler().ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("inspect failed: %d: %s", w.Code, w.Body.String())
	}

	var profile mp4.MP4Profile
	if err := json.Unmarshal(w.Body.Bytes(), &profile); err != nil {
		t.Fatalf("failed to decode profile: %v", err)
	}
	if profile.FastStart.IsFastStart {
		t.Error("expected unoptimized sample to have isFastStart=false")
	}

	// 2. Remediate unoptimized sample
	remReq := httptest.NewRequest(http.MethodPost, "/api/mp4/remediate", bytes.NewReader(reqBody))
	remReq.Header.Set("Content-Type", "application/json")
	remW := httptest.NewRecorder()

	srv.Handler().ServeHTTP(remW, remReq)
	if remW.Code != http.StatusOK {
		t.Fatalf("remediate failed: %d", remW.Code)
	}

	if remW.Header().Get("Content-Type") != "video/mp4" {
		t.Errorf("expected video/mp4, got %s", remW.Header().Get("Content-Type"))
	}

	remediatedData := remW.Body.Bytes()
	remReader := bytes.NewReader(remediatedData)
	verifiedProfile, err := mp4.ParseMP4(remReader, int64(len(remediatedData)))
	if err != nil {
		t.Fatalf("failed to parse remediated output: %v", err)
	}

	if !verifiedProfile.FastStart.IsFastStart {
		t.Error("expected remediated output to have isFastStart=true")
	}
}

func TestAPILadderAnalyze(t *testing.T) {
	srv := NewServer(8850)
	payload := map[string]interface{}{
		"rungs": []ladder.LadderRung{
			{ID: "1080p", Width: 1920, Height: 1080, FPS: 30, Bitrate: 5000000, Codec: "avc1.640028"},
			{ID: "720p", Width: 1280, Height: 720, FPS: 30, Bitrate: 2800000, Codec: "avc1.4d401f"},
		},
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/ladder/analyze", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	srv.Handler().ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("ladder analyze failed: %d", w.Code)
	}

	var analysis ladder.LadderAnalysis
	if err := json.Unmarshal(w.Body.Bytes(), &analysis); err != nil {
		t.Fatalf("failed to decode analysis: %v", err)
	}
	if len(analysis.Rungs) != 2 {
		t.Errorf("expected 2 rungs, got %d", len(analysis.Rungs))
	}
}
