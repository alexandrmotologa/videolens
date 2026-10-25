package hls

import (
	"bufio"
	"bytes"
	"fmt"
	"math"
	"strconv"
	"strings"
)

// FixAction records an automated modification made to sanitize a playlist.
type FixAction struct {
	Code        string `json:"code"`
	Description string `json:"description"`
}

// FixResult contains the remediated manifest text and the list of applied fixes.
type FixResult struct {
	OriginalContent  string      `json:"originalContent"`
	SanitizedContent string      `json:"sanitizedContent"`
	FixesApplied     []FixAction `json:"fixesApplied"`
	OriginalScore    int         `json:"originalScore"`
	SanitizedScore   int         `json:"sanitizedScore"`
}

// FixMediaPlaylist takes an invalid or suboptimal HLS media playlist and produces a compliant version.
func FixMediaPlaylist(content string) (*FixResult, error) {
	origLint, _ := LintHLS(content)
	origScore := 0
	if origLint != nil {
		origScore = origLint.HealthScore
	}

	res := &FixResult{
		OriginalContent: content,
		FixesApplied:    make([]FixAction, 0),
		OriginalScore:   origScore,
	}

	scanner := bufio.NewScanner(bytes.NewBufferString(content))
	var lines []string
	var maxSegmentDur float64
	var hasTargetDuration bool
	var targetDurationLineIdx = -1
	var hasEndList bool
	var hasExtM3U bool
	var isVOD bool

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if line == "#EXTM3U" {
			hasExtM3U = true
		}
		if strings.HasPrefix(line, "#EXT-X-TARGETDURATION:") {
			hasTargetDuration = true
			targetDurationLineIdx = len(lines)
		}
		if strings.HasPrefix(line, "#EXT-X-PLAYLIST-TYPE:") {
			val := strings.ToUpper(strings.TrimSpace(strings.TrimPrefix(line, "#EXT-X-PLAYLIST-TYPE:")))
			if val == "VOD" {
				isVOD = true
			}
		}
		if strings.HasPrefix(line, "#EXT-X-ENDLIST") {
			hasEndList = true
		}

		if strings.HasPrefix(line, "#EXTINF:") {
			inf := strings.TrimPrefix(line, "#EXTINF:")
			parts := strings.SplitN(inf, ",", 2)
			if dur, err := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64); err == nil {
				if dur > maxSegmentDur {
					maxSegmentDur = dur
				}
			}
		}

		lines = append(lines, line)
	}

	var outputLines []string

	// Rule 1: Prepend #EXTM3U if missing
	if !hasExtM3U {
		outputLines = append(outputLines, "#EXTM3U")
		res.FixesApplied = append(res.FixesApplied, FixAction{
			Code:        "FIX_PREPEND_EXTM3U",
			Description: "Prepended mandatory #EXTM3U header to line 1.",
		})
	}

	// Calculate recommended ceiling target duration
	recommendedTargetDuration := int(math.Ceil(maxSegmentDur))
	if recommendedTargetDuration < 1 {
		recommendedTargetDuration = 6
	}

	// Rebuild lines, patching target duration
	for i, line := range lines {
		if strings.HasPrefix(line, "#EXT-X-TARGETDURATION:") {
			valStr := strings.TrimSpace(strings.TrimPrefix(line, "#EXT-X-TARGETDURATION:"))
			curTarget, _ := strconv.Atoi(valStr)
			if curTarget < recommendedTargetDuration {
				outputLines = append(outputLines, fmt.Sprintf("#EXT-X-TARGETDURATION:%d", recommendedTargetDuration))
				res.FixesApplied = append(res.FixesApplied, FixAction{
					Code:        "FIX_TARGET_DURATION_OVERFLOW",
					Description: fmt.Sprintf("Updated #EXT-X-TARGETDURATION from %d to %d to accommodate maximum segment duration (%.3fs).", curTarget, recommendedTargetDuration, maxSegmentDur),
				})
				continue
			}
		}
		outputLines = append(outputLines, line)
		_ = i
	}

	// If #EXT-X-TARGETDURATION was missing entirely, insert it after version or header
	if !hasTargetDuration {
		insertIdx := 1
		if len(outputLines) > 1 && strings.HasPrefix(outputLines[1], "#EXT-X-VERSION") {
			insertIdx = 2
		}
		newLines := make([]string, 0, len(outputLines)+1)
		newLines = append(newLines, outputLines[:insertIdx]...)
		newLines = append(newLines, fmt.Sprintf("#EXT-X-TARGETDURATION:%d", recommendedTargetDuration))
		newLines = append(newLines, outputLines[insertIdx:]...)
		outputLines = newLines
		res.FixesApplied = append(res.FixesApplied, FixAction{
			Code:        "FIX_INSERT_TARGET_DURATION",
			Description: fmt.Sprintf("Inserted missing #EXT-X-TARGETDURATION:%d.", recommendedTargetDuration),
		})
	}
	_ = targetDurationLineIdx

	// Rule 3: Append #EXT-X-ENDLIST for VOD if absent
	if isVOD && !hasEndList {
		outputLines = append(outputLines, "#EXT-X-ENDLIST")
		res.FixesApplied = append(res.FixesApplied, FixAction{
			Code:        "FIX_APPEND_ENDLIST",
			Description: "Appended terminating #EXT-X-ENDLIST tag for VOD playlist.",
		})
	}

	res.SanitizedContent = strings.Join(outputLines, "\n") + "\n"

	// Re-lint sanitized version to verify health score improvement
	sanLint, _ := LintHLS(res.SanitizedContent)
	if sanLint != nil {
		res.SanitizedScore = sanLint.HealthScore
	}

	return res, nil
}
