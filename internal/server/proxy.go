package server

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var httpClient = &http.Client{
	Timeout: 30 * time.Second,
}

// HandleProxy streams an external media file with permissive CORS headers.
func HandleProxy(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	targetURL := r.URL.Query().Get("url")
	if targetURL == "" {
		http.Error(w, `{"error":"missing required 'url' parameter"}`, http.StatusBadRequest)
		return
	}

	parsedURL, err := url.Parse(targetURL)
	if err != nil || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") {
		http.Error(w, `{"error":"invalid URL scheme; must be http or https"}`, http.StatusBadRequest)
		return
	}

	req, err := http.NewRequestWithContext(r.Context(), r.Method, targetURL, r.Body)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"create upstream request: %v"}`, err), http.StatusInternalServerError)
		return
	}

	// Forward Range header if present for video streaming
	if rangeHdr := r.Header.Get("Range"); rangeHdr != "" {
		req.Header.Set("Range", rangeHdr)
	}
	req.Header.Set("User-Agent", "VideoLens-Media-Inspector/1.0")

	resp, err := httpClient.Do(req)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"upstream fetch failed: %v"}`, err), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// Forward pertinent headers
	for _, h := range []string{"Content-Type", "Content-Range", "Accept-Ranges", "Content-Length"} {
		if val := resp.Header.Get(h); val != "" {
			w.Header().Set(h, val)
		}
	}

	// If it is an HLS playlist, rewrite relative segment paths to route through proxy
	contentType := resp.Header.Get("Content-Type")
	isM3U8 := strings.Contains(contentType, "mpegurl") || strings.HasSuffix(parsedURL.Path, ".m3u8")

	if isM3U8 && resp.StatusCode == http.StatusOK {
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			http.Error(w, "read playlist body", http.StatusInternalServerError)
			return
		}

		rewritten := rewritePlaylistURLs(string(bodyBytes), parsedURL)
		w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(rewritten)))
		w.WriteHeader(resp.StatusCode)
		w.Write([]byte(rewritten))
		return
	}

	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

func rewritePlaylistURLs(content string, baseURL *url.URL) string {
	lines := strings.Split(content, "\n")
	var out []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			// Check for URI inside tags like #EXT-X-KEY or #EXT-X-MEDIA
			if strings.HasPrefix(trimmed, "#EXT-X-MEDIA:") || strings.HasPrefix(trimmed, "#EXT-X-KEY:") {
				// Rewrite URI attribute if present
				rewrittenTag := rewriteURIAttribute(trimmed, baseURL)
				out = append(out, rewrittenTag)
			} else {
				out = append(out, line)
			}
			continue
		}

		// Relative or absolute segment URI
		segURL, err := url.Parse(trimmed)
		if err != nil {
			out = append(out, line)
			continue
		}

		resolvedURL := baseURL.ResolveReference(segURL).String()
		proxiedURL := fmt.Sprintf("/api/proxy?url=%s", url.QueryEscape(resolvedURL))
		out = append(out, proxiedURL)
	}

	return strings.Join(out, "\n")
}

func rewriteURIAttribute(tag string, baseURL *url.URL) string {
	idx := strings.Index(tag, `URI="`)
	if idx == -1 {
		return tag
	}
	start := idx + 5
	end := strings.Index(tag[start:], `"`)
	if end == -1 {
		return tag
	}
	rawURI := tag[start : start+end]
	u, err := url.Parse(rawURI)
	if err != nil {
		return tag
	}
	resolved := baseURL.ResolveReference(u).String()
	proxied := fmt.Sprintf("/api/proxy?url=%s", url.QueryEscape(resolved))
	return tag[:start] + proxied + tag[start+end:]
}

func enableCORS(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS, HEAD")
	w.Header().Set("Access-Control-Allow-Headers", "Range, Content-Type, Authorization, Accept")
	w.Header().Set("Access-Control-Expose-Headers", "Content-Range, Content-Length, Accept-Ranges")
}
