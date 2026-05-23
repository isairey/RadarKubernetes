package server

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/skyhook-io/radar/internal/updater"
)

// handleDesktopUpdateStart starts downloading the latest desktop release.
// POST /api/desktop/update → 202 Accepted
func (s *Server) handleDesktopUpdateStart(w http.ResponseWriter, r *http.Request) {
	if s.updater == nil {
		s.writeError(w, http.StatusNotImplemented, "desktop update not available in this build")
		return
	}

	// Use background context: the download runs asynchronously and must not be
	// cancelled when this HTTP response completes or the 60s timeout fires.
	if err := s.updater.StartDownload(context.Background()); err != nil {
		s.writeError(w, http.StatusConflict, err.Error())
		return
	}

	w.WriteHeader(http.StatusAccepted)
	s.writeJSON(w, map[string]string{"status": "downloading"})
}

// handleDesktopUpdateStatus returns the current update status.
// GET /api/desktop/update/status
func (s *Server) handleDesktopUpdateStatus(w http.ResponseWriter, r *http.Request) {
	if s.updater == nil {
		s.writeJSON(w, updater.Status{State: updater.StateIdle})
		return
	}

	s.writeJSON(w, s.updater.Status())
}

// handleDesktopUpdateApply applies the downloaded update (extracts and replaces
// the binary/app bundle). Returns {"status":"applied"} on success. The desktop
// app should trigger a relaunch after receiving this response.
// POST /api/desktop/update/apply
func (s *Server) handleDesktopUpdateApply(w http.ResponseWriter, r *http.Request) {
	if s.updater == nil {
		s.writeError(w, http.StatusNotImplemented, "desktop update not available in this build")
		return
	}

	// Use background context: apply must not be cancelled by the 60s request
	// timeout — a partial apply could leave the installation in a broken state.
	if err := s.updater.Apply(context.Background()); err != nil {
		log.Printf("[desktop-update] Apply failed: %v", err)
		s.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.writeJSON(w, map[string]string{"status": "applied"})

	// Relaunch after a short delay so the HTTP response can flush to the client.
	go func() {
		time.Sleep(500 * time.Millisecond)
		if err := updater.Relaunch(); err != nil {
			log.Printf("[desktop-update] Relaunch failed: %v", err)
		}
	}()
}
