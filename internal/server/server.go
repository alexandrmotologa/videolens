package server

import (
	"embed"
	"fmt"
	"io/fs"
	"net/http"
	"strings"
)

//go:embed all:dist
var distFS embed.FS

func getMimeType(p string) string {
	lower := strings.ToLower(p)
	switch {
	case strings.HasSuffix(lower, ".js") || strings.HasSuffix(lower, ".mjs"):
		return "application/javascript; charset=utf-8"
	case strings.HasSuffix(lower, ".css"):
		return "text/css; charset=utf-8"
	case strings.HasSuffix(lower, ".html"):
		return "text/html; charset=utf-8"
	case strings.HasSuffix(lower, ".svg"):
		return "image/svg+xml"
	case strings.HasSuffix(lower, ".json"):
		return "application/json; charset=utf-8"
	case strings.HasSuffix(lower, ".png"):
		return "image/png"
	case strings.HasSuffix(lower, ".ico"):
		return "image/x-icon"
	default:
		return "application/octet-stream"
	}
}

// Server encapsulates the HTTP server for VideoLens studio.
type Server struct {
	mux  *http.ServeMux
	port int
}

// NewServer initializes a new VideoLens HTTP server.
func NewServer(port int) *Server {
	mux := http.NewServeMux()
	s := &Server{
		mux:  mux,
		port: port,
	}

	RegisterAPIRoutes(mux)
	s.registerStaticHandlers()

	return s
}

func (s *Server) registerStaticHandlers() {
	sub, err := fs.Sub(distFS, "dist")
	if err != nil {
		return
	}

	s.mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			http.NotFound(w, r)
			return
		}

		filePath := strings.TrimPrefix(r.URL.Path, "/")
		if filePath == "" {
			filePath = "index.html"
		}

		// Try opening static asset in embedded dist
		data, err := fs.ReadFile(sub, filePath)
		if err == nil {
			w.Header().Set("Content-Type", getMimeType(filePath))
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(data)
			return
		}

		// Fallback to index.html for SPA router
		indexBytes, err := fs.ReadFile(sub, "index.html")
		if err != nil {
			http.NotFound(w, r)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(indexBytes)
	})
}

// ListenAndServe runs the HTTP server on the configured port.
func (s *Server) ListenAndServe() error {
	addr := fmt.Sprintf(":%d", s.port)
	fmt.Printf("✓ VideoLens Web Studio listening at http://localhost:%d\n", s.port)
	return http.ListenAndServe(addr, s.mux)
}

// Handler returns the underlying http.Handler for testing.
func (s *Server) Handler() http.Handler {
	return s.mux
}
