package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/alexandrmotologa/videolens/internal/hls"
	"github.com/alexandrmotologa/videolens/internal/ladder"
	"github.com/alexandrmotologa/videolens/internal/mp4"
	"github.com/alexandrmotologa/videolens/internal/server"
)

const Version = "1.0.0"

// RunServe starts the HTTP server.
func RunServe(port int) error {
	srv := server.NewServer(port)
	return srv.ListenAndServe()
}

// CheckOptions defines CLI reporting and exit conditions.
type CheckOptions struct {
	Format string // "text", "json", "markdown"
	FailOn string // "error", "warning", ""
}

// RunCheck inspects an HLS, DASH, or MP4 resource with formatting and CI options.
func RunCheck(target string, opts CheckOptions) error {
	data, err := readTargetData(target)
	if err != nil {
		return fmt.Errorf("read target %s: %w", target, err)
	}

	trimmed := strings.TrimSpace(string(data))

	// 1. Check if MP4
	if isMP4(data) {
		return checkMP4(data, target, opts)
	}

	// 2. Check if DASH (.mpd or XML with MPD root)
	if strings.Contains(trimmed, "<MPD") || strings.HasSuffix(strings.ToLower(target), ".mpd") {
		return checkDASH(trimmed, target, opts)
	}

	// 3. Default to HLS
	return checkHLS(trimmed, target, opts)
}

// RunFix repairs an HLS media playlist and outputs the compliant M3U8.
func RunFix(inputFile, outputFile string) error {
	data, err := os.ReadFile(inputFile)
	if err != nil {
		return fmt.Errorf("read input playlist %s: %w", inputFile, err)
	}

	res, err := hls.FixMediaPlaylist(string(data))
	if err != nil {
		return fmt.Errorf("remediate playlist: %w", err)
	}

	if outputFile == "" {
		outputFile = strings.TrimSuffix(inputFile, filepath.Ext(inputFile)) + "_fixed.m3u8"
	}

	if err := os.WriteFile(outputFile, []byte(res.SanitizedContent), 0644); err != nil {
		return fmt.Errorf("write sanitized playlist: %w", err)
	}

	fmt.Printf("✓ Successfully fixed HLS playlist: %s -> %s\n", inputFile, outputFile)
	fmt.Printf("  Original Health Score:  %d/100\n", res.OriginalScore)
	fmt.Printf("  Sanitized Health Score: %d/100\n", res.SanitizedScore)
	fmt.Printf("  Fixes Applied (%d):\n", len(res.FixesApplied))
	for _, f := range res.FixesApplied {
		fmt.Printf("   - [%s] %s\n", f.Code, f.Description)
	}
	return nil
}

// RunFastStart remuxes an MP4 to place moov before mdat.
func RunFastStart(inputFile, outputFile string) error {
	if inputFile == "" {
		return fmt.Errorf("input file path required")
	}
	if outputFile == "" {
		outputFile = strings.TrimSuffix(inputFile, filepath.Ext(inputFile)) + "_faststart.mp4"
	}

	data, err := os.ReadFile(inputFile)
	if err != nil {
		return fmt.Errorf("read input file: %w", err)
	}

	reader := bytes.NewReader(data)
	remediated, err := mp4.RelocateMoov(reader, int64(len(data)))
	if err != nil {
		return fmt.Errorf("relocate moov: %w", err)
	}

	if err := os.WriteFile(outputFile, remediated, 0644); err != nil {
		return fmt.Errorf("write output file: %w", err)
	}

	fmt.Printf("✓ Successfully remuxed %s -> %s\n", inputFile, outputFile)
	fmt.Printf("  Original size:    %d bytes\n", len(data))
	fmt.Printf("  Remediated size:  %d bytes\n", len(remediated))
	fmt.Printf("  Fast-Start Status: OPTIMIZED (moov atom placed at head of file)\n")
	return nil
}

// RunABR validates an ABR bitrate ladder from a master playlist.
func RunABR(target string) error {
	data, err := readTargetData(target)
	if err != nil {
		return fmt.Errorf("read target %s: %w", target, err)
	}

	content := string(data)
	res, err := hls.LintHLS(content)
	if err != nil {
		return fmt.Errorf("parse manifest: %w", err)
	}

	if !res.IsMaster || len(res.Variants) == 0 {
		return fmt.Errorf("target is not an HLS master playlist with variants")
	}

	var rungs []ladder.LadderRung
	for i, v := range res.Variants {
		name := v.Resolution
		if name == "" {
			name = fmt.Sprintf("variant-%d", i+1)
		}
		rungs = append(rungs, ladder.LadderRung{
			ID:         name,
			Resolution: v.Resolution,
			Width:      v.Width,
			Height:     v.Height,
			FPS:        v.FrameRate,
			Bitrate:    v.Bandwidth,
			Codec:      v.Codecs,
		})
	}

	analysis := ladder.AnalyzeLadder(rungs)

	fmt.Println("================================================================================")
	fmt.Printf(" VIDEOLENS ABR LADDER ANALYSIS: %s\n", target)
	fmt.Printf(" Health Score: %d/100 | Dynamic Range: %.1fx\n", analysis.HealthScore, analysis.DynamicRange)
	fmt.Println("================================================================================")
	fmt.Printf("%-12s | %-12s | %-10s | %-10s | %-12s | %s\n", "RESOLUTION", "BITRATE", "FPS", "STEP", "BPP", "STATUS")
	fmt.Println("--------------------------------------------------------------------------------")

	for _, r := range analysis.Rungs {
		fmt.Printf("%-12s | %-12s | %-10.1f | %-10.2fx | %-12.4f | %s\n",
			r.Resolution, fmt.Sprintf("%.2f Mbps", r.BitrateMbps), r.FPS, r.StepRatio, r.BitsPerPixel, r.BPPStatus)
	}
	fmt.Println("--------------------------------------------------------------------------------")

	if len(analysis.Issues) > 0 {
		fmt.Println("Issues:")
		for _, issue := range analysis.Issues {
			fmt.Printf(" [%s] %s: %s\n", issue.Severity, issue.Code, issue.Message)
		}
	} else {
		fmt.Println("✓ No ladder anomalies detected. Bitrate progression is clean.")
	}
	return nil
}

func checkMP4(data []byte, target string, opts CheckOptions) error {
	reader := bytes.NewReader(data)
	profile, err := mp4.ParseMP4(reader, int64(len(data)))
	if err != nil {
		return err
	}

	if opts.Format == "json" {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(profile)
	}

	fmt.Println("================================================================================")
	fmt.Printf(" VIDEOLENS MP4 ATOM PROFILER: %s\n", target)
	fmt.Println("================================================================================")
	fmt.Printf(" File Size:        %d bytes (%.2f MB)\n", profile.FileSize, float64(profile.FileSize)/(1024*1024))
	fmt.Printf(" Duration:         %.2f seconds\n", profile.DurationSeconds)
	fmt.Printf(" Major Brand:      %s (Compatible: %s)\n", profile.MajorBrand, strings.Join(profile.Compatible, ", "))
	fmt.Println("--------------------------------------------------------------------------------")
	fmt.Println(" FAST-START WEB STREAMING AUDIT:")
	if profile.FastStart != nil {
		fmt.Printf("  Status:          [%s]\n", profile.FastStart.StatusBadge)
		fmt.Printf("  moov Offset:     %d bytes\n", profile.FastStart.MoovOffset)
		fmt.Printf("  mdat Offset:     %d bytes\n", profile.FastStart.MdatOffset)
		fmt.Printf("  Summary:         %s\n", profile.FastStart.Message)
	}
	fmt.Println("--------------------------------------------------------------------------------")
	fmt.Printf(" TRACKS DETECTED (%d):\n", len(profile.Tracks))
	for _, trk := range profile.Tracks {
		dim := ""
		if trk.Width > 0 && trk.Height > 0 {
			dim = fmt.Sprintf("%.0fx%.0f, ", trk.Width, trk.Height)
		}
		sr := ""
		if trk.SampleRate > 0 {
			sr = fmt.Sprintf("%d Hz, ", trk.SampleRate)
		}
		gopStr := ""
		if trk.GOP != nil {
			gopStr = fmt.Sprintf("GOP: %s (~%.1fs), ", trk.GOP.CadenceType, trk.GOP.MeanGOPDuration)
		}
		fmt.Printf("  #%d [%s] Codec: %s (%s%s%sDuration: %.2fs)\n", trk.TrackID, trk.Type, trk.Codec, dim, sr, gopStr, trk.DurationSeconds)
	}
	fmt.Println("--------------------------------------------------------------------------------")
	fmt.Println(" ATOM TREE HIERARCHY:")
	fmt.Print(profile.PrintTree())
	fmt.Println("================================================================================")

	if opts.FailOn == "error" && profile.FastStart != nil && !profile.FastStart.IsFastStart {
		return fmt.Errorf("CI check failed: MP4 is not Fast-Start optimized")
	}
	return nil
}

func checkHLS(content string, target string, opts CheckOptions) error {
	res, err := hls.LintHLS(content)
	if err != nil {
		return err
	}

	if opts.Format == "json" {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(res)
	}

	fmt.Println("================================================================================")
	fmt.Printf(" VIDEOLENS HLS MANIFEST LINTER: %s\n", target)
	fmt.Printf(" Health Score: %d/100 | Is Master: %t | Version: %d\n", res.HealthScore, res.IsMaster, res.Version)
	if res.IsLowLatency {
		fmt.Printf(" LL-HLS Enabled: PART-TARGET=%.2fs | Total Parts: %d\n", res.PartTarget, res.TotalPartsCount)
	}
	fmt.Println("================================================================================")

	if res.IsMaster {
		fmt.Printf(" Variants (%d):\n", len(res.Variants))
		for _, v := range res.Variants {
			fmt.Printf("  - %-10s %-12s Codecs: %-20s URI: %s\n", v.Resolution, fmt.Sprintf("%.1f Mbps", float64(v.Bandwidth)/1000000.0), v.Codecs, v.URI)
		}
		if len(res.Renditions) > 0 {
			fmt.Printf(" Renditions (%d):\n", len(res.Renditions))
			for _, r := range res.Renditions {
				fmt.Printf("  - [%s] Group: %-12s Name: %-10s URI: %s\n", r.Type, r.GroupID, r.Name, r.URI)
			}
		}
	} else {
		fmt.Printf(" Segments: %d | Total Duration: %.2fs | Target Duration: %.1fs\n", res.SegmentCount, res.TotalDuration, res.TargetDuration)
		if res.DurationMetrics != nil {
			fmt.Printf(" Duration Metrics: Min=%.2fs Max=%.2fs Mean=%.2fs StdDev=%.2fs Drift=%.1f%%\n",
				res.DurationMetrics.MinDuration, res.DurationMetrics.MaxDuration, res.DurationMetrics.MeanDuration,
				res.DurationMetrics.StdDev, res.DurationMetrics.DriftPercent)
		}
	}

	fmt.Println("--------------------------------------------------------------------------------")
	if len(res.Issues) == 0 {
		fmt.Println("✓ Manifest passed all RFC 8216 validation checks without violations.")
	} else {
		fmt.Printf(" Violations and Warnings (%d):\n", len(res.Issues))
		for _, issue := range res.Issues {
			fmt.Printf("  [%s] Line %d (%s): %s\n", issue.Severity, issue.Line, issue.Code, issue.Message)
			if issue.Recommendation != "" {
				fmt.Printf("    -> Recommendation: %s\n", issue.Recommendation)
			}
		}
	}
	fmt.Println("================================================================================")

	errorCount := 0
	warningCount := 0
	for _, i := range res.Issues {
		if i.Severity == hls.SeverityError {
			errorCount++
		} else if i.Severity == hls.SeverityWarning {
			warningCount++
		}
	}

	if opts.FailOn == "error" && errorCount > 0 {
		return fmt.Errorf("CI check failed: %d errors encountered in manifest", errorCount)
	}
	if opts.FailOn == "warning" && (errorCount > 0 || warningCount > 0) {
		return fmt.Errorf("CI check failed: %d issues encountered in manifest", errorCount+warningCount)
	}
	return nil
}

func checkDASH(content string, target string, opts CheckOptions) error {
	res, err := hls.LintDASH(content)
	if err != nil {
		return err
	}

	if opts.Format == "json" {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(res)
	}

	fmt.Println("================================================================================")
	fmt.Printf(" VIDEOLENS MPEG-DASH MANIFEST LINTER: %s\n", target)
	fmt.Printf(" Health Score: %d/100 | Type: %s | MinBufferTime: %.1fs\n", res.HealthScore, res.Type, res.MinBufferTimeSec)
	fmt.Println("================================================================================")
	fmt.Printf(" Adaptation Sets (%d):\n", len(res.AdaptationSets))
	for _, as := range res.AdaptationSets {
		fmt.Printf("  - ID: %s [%s] Mime: %s\n", as.ID, as.ContentType, as.MimeType)
		for _, rep := range as.Representations {
			resStr := ""
			if rep.Width > 0 {
				resStr = fmt.Sprintf("%dx%d, ", rep.Width, rep.Height)
			}
			fmt.Printf("    * Rep %s: %s%.1f Mbps, Codecs: %s\n", rep.ID, resStr, float64(rep.Bandwidth)/1000000.0, rep.Codecs)
		}
	}
	fmt.Println("--------------------------------------------------------------------------------")
	if len(res.Issues) == 0 {
		fmt.Println("✓ Manifest passed all MPEG-DASH validation checks.")
	} else {
		for _, issue := range res.Issues {
			fmt.Printf("  [%s] %s: %s\n", issue.Severity, issue.Code, issue.Message)
		}
	}
	fmt.Println("================================================================================")

	if opts.FailOn == "error" && len(res.Issues) > 0 {
		for _, i := range res.Issues {
			if i.Severity == hls.SeverityError {
				return fmt.Errorf("CI check failed: errors detected in DASH manifest")
			}
		}
	}
	return nil
}

func readTargetData(target string) ([]byte, error) {
	if strings.HasPrefix(target, "http://") || strings.HasPrefix(target, "https://") {
		resp, err := http.Get(target)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		return io.ReadAll(resp.Body)
	}
	return os.ReadFile(target)
}

func isMP4(data []byte) bool {
	if len(data) < 8 {
		return false
	}
	// Check standard FourCC identifiers at offset 4
	fourCC := string(data[4:8])
	switch fourCC {
	case "ftyp", "moov", "mdat", "free", "skip", "wide":
		return true
	}
	return false
}
