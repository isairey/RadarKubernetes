package server

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"path/filepath"
	"strings"
)

// ErrSaveCancelled is returned by saveFileFunc when the user dismisses the
// native save dialog. The handler returns 204 No Content in this case.
var ErrSaveCancelled = errors.New("save dialog cancelled")

// handleDesktopSaveFile shows the native OS save dialog and writes a file.
// In the desktop app, blob URL downloads are silently swallowed by WKWebView
// (macOS) and other embedded webviews, so the frontend calls this endpoint
// to trigger a native save dialog instead.
// Returns 204 if the user cancels the dialog, or {"path": "..."} on success.
// POST /api/desktop/save-file
func (s *Server) handleDesktopSaveFile(w http.ResponseWriter, r *http.Request) {
	if s.saveFileFunc == nil {
		s.writeError(w, http.StatusNotFound, "not available")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 100<<20) // 100 MB limit

	var req struct {
		Filename      string `json:"filename"`
		Content       string `json:"content"`
		ContentBase64 string `json:"contentBase64"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Sanitize filename: strip path components, reject empty
	req.Filename = filepath.Base(req.Filename)
	req.Filename = strings.ReplaceAll(req.Filename, "\x00", "")
	if req.Filename == "" || req.Filename == "." || req.Filename == ".." {
		s.writeError(w, http.StatusBadRequest, "invalid filename")
		return
	}

	// Decode content — exactly one of content or contentBase64 must be provided
	var data []byte
	switch {
	case req.Content != "" && req.ContentBase64 != "":
		s.writeError(w, http.StatusBadRequest, "provide content or contentBase64, not both")
		return
	case req.Content != "":
		data = []byte(req.Content)
	case req.ContentBase64 != "":
		var err error
		data, err = base64.StdEncoding.DecodeString(req.ContentBase64)
		if err != nil {
			s.writeError(w, http.StatusBadRequest, "invalid base64 content")
			return
		}
	default:
		s.writeError(w, http.StatusBadRequest, "content or contentBase64 is required")
		return
	}

	path, err := s.saveFileFunc(req.Filename, data)
	if err != nil {
		if errors.Is(err, ErrSaveCancelled) {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		log.Printf("[desktop] Failed to save file %q: %v", req.Filename, err)
		s.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]string{"path": path}); err != nil {
		log.Printf("[desktop] Failed to write save-file response: %v", err)
	}
}
