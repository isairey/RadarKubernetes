package updater

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/skyhook-io/radar/internal/version"
)

// State represents the current state of the update process.
type State string

const (
	StateIdle        State = "idle"
	StateDownloading State = "downloading"
	StateReady       State = "ready"
	StateApplying    State = "applying"
	StateError       State = "error"
)

// Status contains the current update status for API consumers.
type Status struct {
	State    State   `json:"state"`
	Progress float64 `json:"progress,omitempty"` // 0.0 - 1.0 during download
	Version  string  `json:"version,omitempty"`  // target version
	Error    string  `json:"error,omitempty"`
}

// Updater manages the desktop app self-update lifecycle.
type Updater struct {
	mu        sync.Mutex
	state     State
	progress  float64
	version   string // target version being downloaded/ready
	assetPath string // path to downloaded asset
	assetName string // original asset filename (for checksum lookup)
	err       error
	cancel    context.CancelFunc
}

// New creates a new Updater instance.
func New() *Updater {
	return &Updater{state: StateIdle}
}

// Status returns the current update status.
func (u *Updater) Status() Status {
	u.mu.Lock()
	defer u.mu.Unlock()

	s := Status{
		State:    u.state,
		Progress: u.progress,
		Version:  u.version,
	}
	if u.err != nil {
		s.Error = u.err.Error()
	}
	return s
}

// StartDownload begins an async download of the latest desktop release.
// Returns an error immediately if a download is already in progress or an update is being applied.
func (u *Updater) StartDownload(parentCtx context.Context) error {
	u.mu.Lock()
	if u.state != StateIdle && u.state != StateError {
		st := u.state
		u.mu.Unlock()
		return fmt.Errorf("cannot start download in state %q", st)
	}

	u.state = StateDownloading
	u.progress = 0
	u.err = nil
	u.assetPath = ""

	ctx, cancel := context.WithCancel(parentCtx)
	u.cancel = cancel
	u.mu.Unlock()

	go u.doDownload(ctx)
	return nil
}

func (u *Updater) doDownload(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			u.setError(fmt.Errorf("internal error: %v", r))
		}
		u.mu.Lock()
		u.cancel = nil
		u.mu.Unlock()
	}()

	// Fetch latest release
	log.Printf("[updater] Fetching latest release...")
	release, err := version.FetchRelease(ctx)
	if err != nil {
		u.setError(fmt.Errorf("fetch release: %w", err))
		return
	}

	targetVersion := strings.TrimPrefix(release.TagName, "v")
	u.mu.Lock()
	u.version = targetVersion
	u.mu.Unlock()

	// Find matching desktop asset
	asset := version.FindDesktopAsset(release, runtime.GOOS, runtime.GOARCH)
	if asset == nil {
		u.setError(fmt.Errorf("no desktop build found for %s/%s in release %s",
			runtime.GOOS, runtime.GOARCH, release.TagName))
		return
	}
	log.Printf("[updater] Found asset: %s (%d bytes)", asset.Name, asset.Size)

	// Download to updates dir
	updatesDir, err := version.UpdatesDir()
	if err != nil {
		u.setError(fmt.Errorf("get updates dir: %w", err))
		return
	}
	dest := filepath.Join(updatesDir, asset.Name)

	err = version.DownloadAsset(ctx, asset.BrowserDownloadURL, dest, func(downloaded, total int64) {
		if total > 0 {
			u.mu.Lock()
			u.progress = float64(downloaded) / float64(total)
			u.mu.Unlock()
		}
	})
	if err != nil {
		u.setError(fmt.Errorf("download: %w", err))
		return
	}

	// Verify checksum
	log.Printf("[updater] Verifying checksum...")
	if err := version.VerifyChecksum(ctx, release, dest, asset.Name); err != nil {
		u.setError(fmt.Errorf("checksum verification: %w", err))
		return
	}

	// Mark as ready
	u.mu.Lock()
	u.state = StateReady
	u.progress = 1.0
	u.assetPath = dest
	u.assetName = asset.Name
	u.mu.Unlock()
	log.Printf("[updater] Download complete, ready to apply: %s", dest)
}

// Apply extracts the downloaded update and replaces the current binary/app bundle.
// On success, the caller should relaunch the application.
func (u *Updater) Apply(ctx context.Context) error {
	u.mu.Lock()
	if u.state != StateReady {
		u.mu.Unlock()
		return fmt.Errorf("no update ready to apply (state: %s)", u.state)
	}
	u.state = StateApplying
	assetPath := u.assetPath
	u.mu.Unlock()

	log.Printf("[updater] Applying update from %s...", assetPath)
	if err := applyUpdate(ctx, assetPath); err != nil {
		u.setError(fmt.Errorf("apply: %w", err))
		return err
	}

	u.mu.Lock()
	u.state = StateIdle
	u.mu.Unlock()

	log.Printf("[updater] Update applied successfully, ready for relaunch")
	return nil
}

func (u *Updater) setError(err error) {
	log.Printf("[updater] Error: %v", err)
	u.mu.Lock()
	u.state = StateError
	u.err = err
	u.mu.Unlock()
}

// CleanupOldUpdate removes leftover files from previous updates (e.g., .app.old).
// Called on startup.
func CleanupOldUpdate() {
	cleanupPlatform()
}
