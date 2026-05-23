package auth

import (
	"testing"
	"time"
)

func TestMemoryRevoker_RevokeAndCheck(t *testing.T) {
	r := NewMemoryRevoker()
	defer r.Stop()

	sid := "abc123"
	r.Revoke(sid, time.Now().Add(1*time.Hour))

	if !r.IsRevoked(sid) {
		t.Error("expected sid to be revoked")
	}
	if r.IsRevoked("other-sid") {
		t.Error("non-revoked sid should not be reported as revoked")
	}
}

func TestMemoryRevoker_ExpiredRevocation(t *testing.T) {
	r := NewMemoryRevoker()
	defer r.Stop()

	sid := "expired-sid"
	r.Revoke(sid, time.Now().Add(-1*time.Second)) // already expired

	if r.IsRevoked(sid) {
		t.Error("expired revocation should not be reported as revoked")
	}
}

func TestMemoryRevoker_EmptySID(t *testing.T) {
	r := NewMemoryRevoker()
	defer r.Stop()

	// Empty SID (legacy cookie) should never be revoked
	r.Revoke("", time.Now().Add(1*time.Hour))
	if r.IsRevoked("") {
		t.Error("empty SID should never be reported as revoked")
	}
}

func TestMemoryRevoker_SeenJTI(t *testing.T) {
	r := NewMemoryRevoker()
	defer r.Stop()

	jti := "unique-jti-123"
	expiry := time.Now().Add(1 * time.Hour)

	// First time: not seen
	if r.SeenJTI(jti, expiry) {
		t.Error("first call to SeenJTI should return false")
	}

	// Second time: seen (idempotent)
	if !r.SeenJTI(jti, expiry) {
		t.Error("second call to SeenJTI should return true")
	}
}

func TestMemoryRevoker_SeenJTI_EmptyJTI(t *testing.T) {
	r := NewMemoryRevoker()
	defer r.Stop()

	// Empty JTI = can't dedupe, always returns false
	if r.SeenJTI("", time.Now().Add(1*time.Hour)) {
		t.Error("empty JTI should always return false (can't dedupe)")
	}
	if r.SeenJTI("", time.Now().Add(1*time.Hour)) {
		t.Error("empty JTI should always return false even on repeat")
	}
}

func TestMemoryRevoker_GC(t *testing.T) {
	r := &MemoryRevoker{
		revoked:  make(map[string]time.Time),
		jtis:     make(map[string]time.Time),
		stopCh:   make(chan struct{}),
		gcTicker: time.NewTicker(10 * time.Millisecond), // fast GC for test
	}
	go r.gc()
	defer r.Stop()

	// Add entries that are already expired
	r.Revoke("old-sid", time.Now().Add(-1*time.Second))
	r.mu.Lock()
	r.jtis["old-jti"] = time.Now().Add(-1 * time.Second)
	r.mu.Unlock()

	// Add entries that are still valid
	r.Revoke("fresh-sid", time.Now().Add(1*time.Hour))

	// Wait for GC to run
	time.Sleep(50 * time.Millisecond)

	r.mu.RLock()
	_, hasOldSID := r.revoked["old-sid"]
	_, hasOldJTI := r.jtis["old-jti"]
	_, hasFreshSID := r.revoked["fresh-sid"]
	r.mu.RUnlock()

	if hasOldSID {
		t.Error("GC should have removed expired revocation")
	}
	if hasOldJTI {
		t.Error("GC should have removed expired JTI")
	}
	if !hasFreshSID {
		t.Error("GC should not remove non-expired revocation")
	}
}
