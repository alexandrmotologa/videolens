package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/alexandrmotologa/videolens/internal/fixtures"
	"github.com/alexandrmotologa/videolens/internal/hls"
	"github.com/alexandrmotologa/videolens/internal/ladder"
	"github.com/alexandrmotologa/videolens/internal/mp4"
)

type LintRequest struct {
	URL     string `json:"url,omitempty"`
	Content string `json:"content,omitempty"`
}

type MP4SampleRequest struct {
	Sample string `json:"sample,omitempty"` // "faststart" or "unoptimized"
}

type LadderAnalyzeRequest struct {
	Rungs []ladder.LadderRung `json:"rungs"`
}

// RegisterAPIRoutes configures all REST API routes on a ServeMux.
func RegisterAPIRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/proxy", HandleProxy)
	mux.HandleFunc("/api/hls/lint", handleHLSLint)
	mux.HandleFunc("/api/hls/fix", handleHLSFix)
	mux.HandleFunc("/api/dash/lint", handleDASHLint)
	mux.HandleFunc("/api/mp4/inspect", handleMP4Inspect)
	mux.HandleFunc("/api/mp4/remediate", handleMP4Remediate)
	mux.HandleFunc("/api/ladder/analyze", handleLadderAnalyze)
	mux.HandleFunc("/api/ladder/align", handleLadderAlign)
	mux.HandleFunc("/api/samples", handleSamples)
}

func handleHLSLint(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"POST required"}`, http.StatusMethodNotAllowed)
		return
	}

	var req LintRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"invalid JSON payload: %v"}`, err), http.StatusBadRequest)
		return
	}

	content := req.Content
	if req.URL != "" && strings.TrimSpace(content) == "" {
		fetched, err := fetchURL(req.URL)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"failed to fetch manifest URL: %v"}`, err), http.StatusBadGateway)
			return
		}
		content = fetched
	}

	res, err := hls.LintHLS(content)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"linting failed: %v"}`, err), http.StatusInternalServerError)
		return
	}

	// If master playlist with variants, also attach ABR analysis
	type CombinedResult struct {
		*hls.HLSLintResult
		Ladder *ladder.LadderAnalysis `json:"ladder,omitempty"`
	}

	combined := CombinedResult{HLSLintResult: res}
	if res.IsMaster && len(res.Variants) > 0 {
		var rungs []ladder.LadderRung
		for i, v := range res.Variants {
			rungID := fmt.Sprintf("variant-%d", i+1)
			if v.Resolution != "" {
				rungID = v.Resolution
			}
			rungs = append(rungs, ladder.LadderRung{
				ID:         rungID,
				Resolution: v.Resolution,
				Width:      v.Width,
				Height:     v.Height,
				FPS:        v.FrameRate,
				Bitrate:    v.Bandwidth,
				Codec:      v.Codecs,
			})
		}
		combined.Ladder = ladder.AnalyzeLadder(rungs)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(combined)
}

func handleHLSFix(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"POST required"}`, http.StatusMethodNotAllowed)
		return
	}

	var req LintRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"invalid JSON: %v"}`, err), http.StatusBadRequest)
		return
	}

	content := req.Content
	if req.URL != "" && strings.TrimSpace(content) == "" {
		fetched, err := fetchURL(req.URL)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"fetch failed: %v"}`, err), http.StatusBadGateway)
			return
		}
		content = fetched
	}

	fixResult, err := hls.FixMediaPlaylist(content)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"fix failed: %v"}`, err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(fixResult)
}

func handleLadderAlign(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"POST required"}`, http.StatusMethodNotAllowed)
		return
	}

	type AlignRequest struct {
		Variants []ladder.VariantTrack `json:"variants"`
	}

	var req AlignRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"invalid JSON: %v"}`, err), http.StatusBadRequest)
		return
	}

	matrix := ladder.CompareVariantAlignment(req.Variants)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(matrix)
}

func handleDASHLint(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"POST required"}`, http.StatusMethodNotAllowed)
		return
	}

	var req LintRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"invalid JSON payload: %v"}`, err), http.StatusBadRequest)
		return
	}

	content := req.Content
	if req.URL != "" && strings.TrimSpace(content) == "" {
		fetched, err := fetchURL(req.URL)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"failed to fetch manifest URL: %v"}`, err), http.StatusBadGateway)
			return
		}
		content = fetched
	}

	res, err := hls.LintDASH(content)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"dash linting failed: %v"}`, err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}

func handleMP4Inspect(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	var data []byte
	var err error

	contentType := r.Header.Get("Content-Type")
	if strings.HasPrefix(contentType, "multipart/form-data") {
		// Handle file upload (up to 500MB)
		if err := r.ParseMultipartForm(500 << 20); err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"parse multipart: %v"}`, err), http.StatusBadRequest)
			return
		}
		file, _, err := r.FormFile("file")
		if err != nil {
			http.Error(w, `{"error":"missing 'file' field in multipart form"}`, http.StatusBadRequest)
			return
		}
		defer file.Close()
		data, err = io.ReadAll(file)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"read uploaded file: %v"}`, err), http.StatusInternalServerError)
			return
		}
	} else {
		// Handle JSON sample request
		var req MP4SampleRequest
		json.NewDecoder(r.Body).Decode(&req)
		if req.Sample == "unoptimized" {
			data = mp4.GenerateSampleMP4(false)
		} else {
			data = mp4.GenerateSampleMP4(true)
		}
	}

	reader := bytes.NewReader(data)
	profile, err := mp4.ParseMP4(reader, int64(len(data)))
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"parse MP4 container: %v"}`, err), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(profile)
}

func handleMP4Remediate(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	var data []byte
	var err error

	contentType := r.Header.Get("Content-Type")
	if strings.HasPrefix(contentType, "multipart/form-data") {
		if err := r.ParseMultipartForm(500 << 20); err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"parse multipart: %v"}`, err), http.StatusBadRequest)
			return
		}
		file, _, err := r.FormFile("file")
		if err != nil {
			http.Error(w, `{"error":"missing 'file' in multipart form"}`, http.StatusBadRequest)
			return
		}
		defer file.Close()
		data, err = io.ReadAll(file)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"read uploaded file: %v"}`, err), http.StatusInternalServerError)
			return
		}
	} else {
		var req MP4SampleRequest
		json.NewDecoder(r.Body).Decode(&req)
		data = mp4.GenerateSampleMP4(false)
	}

	reader := bytes.NewReader(data)
	remediatedBytes, err := mp4.RelocateMoov(reader, int64(len(data)))
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"remux moov: %v"}`, err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "video/mp4")
	w.Header().Set("Content-Disposition", "attachment; filename=\"videolens_faststart.mp4\"")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(remediatedBytes)))
	w.WriteHeader(http.StatusOK)
	w.Write(remediatedBytes)
}

func handleLadderAnalyze(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"POST required"}`, http.StatusMethodNotAllowed)
		return
	}

	var req LadderAnalyzeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"invalid JSON: %v"}`, err), http.StatusBadRequest)
		return
	}

	analysis := ladder.AnalyzeLadder(req.Rungs)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(analysis)
}

func handleSamples(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)

	type SampleItem struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		Type        string `json:"type"` // "hls_master", "hls_media", "dash", "mp4"
		Description string `json:"description"`
		Content     string `json:"content,omitempty"`
		URL         string `json:"url,omitempty"`
	}

	samples := []SampleItem{
		{
			ID:          "hls-master-valid",
			Name:        "Apple Big Buck Bunny (Valid HLS Master)",
			Type:        "hls_master",
			Description: "RFC 8216 compliant 4-variant master playlist with audio and subtitle renditions.",
			Content:     fixtures.ValidHLSMaster,
		},
		{
			ID:          "hls-media-valid",
			Name:        "Compliant VOD Stream (Valid HLS Media)",
			Type:        "hls_media",
			Description: "Uniform 6s segments with correct target duration and terminating #EXT-X-ENDLIST.",
			Content:     fixtures.ValidHLSMedia,
		},
		{
			ID:          "hls-media-broken",
			Name:        "Target Duration & Drift Violations (Broken HLS Media)",
			Type:        "hls_media",
			Description: "Target duration overflow, high jitter drift, and missing #EXT-X-ENDLIST.",
			Content:     fixtures.BrokenHLSMedia,
		},
		{
			ID:          "dash-valid",
			Name:        "Multi-Bitrate DASH Manifest (Valid MPEG-DASH)",
			Type:        "dash",
			Description: "MPEG-DASH XML manifest with 3 video representations and 1 audio representation.",
			Content:     fixtures.ValidDASHManifest,
		},
		{
			ID:          "live-mux-test",
			Name:        "Live Remote HLS Test Stream (Mux Big Buck Bunny)",
			Type:        "hls_master",
			Description: "Live multi-variant HLS test stream hosted by Mux.",
			URL:         "https://test-streams.mux.dev/x36xhzz/x36xhzz.m3u8",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(samples)
}

func fetchURL(target string) (string, error) {
	resp, err := httpClient.Get(target)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}
