package auth

import (
	"log"
	"sync"
	"time"
)

// Ensure MemoryRevoker implements SessionRevoker.
var _ SessionRevoker = (*MemoryRevoker)(nil)

// MemoryRevoker is an in-memory session revocation store for OIDC backchannel
// logout. It tracks revoked session IDs (sid) and deduplicates logout_token
// JTIs to ensure idempotent handling of IdP retries.
//
// Designed for single-replica deployments. Revocations are lost on pod restart
// and are not shared across replicas.
type MemoryRevoker struct {
	mu       sync.RWMutex
	revoked  map[string]time.Time // sid → expiry (auto-cleaned after expiry)
	jtis     map[string]time.Time // jti → expiry (dedupe for IdP retries)
	stopCh   chan struct{}
	gcTicker *time.Ticker
}

// NewMemoryRevoker creates a revocation store and starts a background GC
// goroutine that cleans up expired entries every 60 seconds.
func NewMemoryRevoker() *MemoryRevoker {
	r := &MemoryRevoker{
		revoked:  make(map[string]time.Time),
		jtis:     make(map[string]time.Time),
		stopCh:   make(chan struct{}),
		gcTicker: time.NewTicker(60 * time.Second),
	}
	go r.gc()
	return r
}

// Revoke marks a session ID as revoked until the given expiry time.
// After expiry, the entry is cleaned up by the GC goroutine.
func (r *MemoryRevoker) Revoke(sid string, expiry time.Time) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.revoked[sid] = expiry
	log.Printf("[auth] Session revoked: sid=%s (expires %s)", sid, expiry.Format(time.RFC3339))
}

// IsRevoked returns true if the given session ID has been revoked and the
// revocation has not yet expired.
func (r *MemoryRevoker) IsRevoked(sid string) bool {
	if sid == "" {
		return false // legacy cookies without sid can't be revoked
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	expiry, ok := r.revoked[sid]
	if !ok {
		return false
	}
	return time.Now().Before(expiry)
}

// SeenJTI returns true if this JTI has already been processed (idempotent
// retry from the IdP). If not seen, it records the JTI with the given expiry.
func (r *MemoryRevoker) SeenJTI(jti string, expiry time.Time) bool {
	if jti == "" {
		return false // no jti = can't dedupe, allow processing
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, seen := r.jtis[jti]; seen {
		return true
	}
	r.jtis[jti] = expiry
	return false
}

// Stop halts the GC goroutine. Call on shutdown.
func (r *MemoryRevoker) Stop() {
	r.gcTicker.Stop()
	close(r.stopCh)
}

// gc periodically removes expired revocations and JTIs.
func (r *MemoryRevoker) gc() {
	for {
		select {
		case <-r.gcTicker.C:
			r.mu.Lock()
			now := time.Now()
			for sid, expiry := range r.revoked {
				if now.After(expiry) {
					delete(r.revoked, sid)
				}
			}
			for jti, expiry := range r.jtis {
				if now.After(expiry) {
					delete(r.jtis, jti)
				}
			}
			r.mu.Unlock()
		case <-r.stopCh:
			return
		}
	}
}
