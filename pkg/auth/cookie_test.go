package auth

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestCreateAndParseSessionCookie(t *testing.T) {
	secret := "test-secret-key"
	user := &User{Username: "alice", Groups: []string{"devs", "admins"}}
	sid := NewSessionID()
	ttl := 1 * time.Hour

	cookie := CreateSessionCookie(user, sid, "", secret, ttl, false)

	// Verify cookie properties
	if cookie.Name != DefaultCookieName {
		t.Errorf("cookie name = %q, want %q", cookie.Name, DefaultCookieName)
	}
	if !cookie.HttpOnly {
		t.Error("cookie should be HttpOnly")
	}
	if cookie.Secure {
		t.Error("cookie should not be Secure when secure=false")
	}
	if cookie.SameSite != http.SameSiteLaxMode {
		t.Errorf("cookie SameSite = %v, want Lax", cookie.SameSite)
	}
	if cookie.MaxAge != 3600 {
		t.Errorf("cookie MaxAge = %d, want 3600", cookie.MaxAge)
	}

	// Parse it back
	req := httptest.NewRequest("GET", "/", nil)
	req.AddCookie(cookie)

	parsed := ParseSessionCookie(req, secret)
	if parsed == nil {
		t.Fatal("ParseSessionCookie returned nil for valid cookie")
	}
	if parsed.User.Username != "alice" {
		t.Errorf("username = %q, want %q", parsed.User.Username, "alice")
	}
	if len(parsed.User.Groups) != 2 || parsed.User.Groups[0] != "devs" || parsed.User.Groups[1] != "admins" {
		t.Errorf("groups = %v, want [devs admins]", parsed.User.Groups)
	}
	if parsed.SID != sid {
		t.Errorf("SID = %q, want %q", parsed.SID, sid)
	}
}

func TestParseSessionCookie_WrongSecret(t *testing.T) {
	user := &User{Username: "alice"}
	cookie := CreateSessionCookie(user, NewSessionID(), "", "secret-1", 1*time.Hour, false)

	req := httptest.NewRequest("GET", "/", nil)
	req.AddCookie(cookie)

	parsed := ParseSessionCookie(req, "secret-2")
	if parsed != nil {
		t.Error("ParseSessionCookie should return nil for wrong secret")
	}
}

func TestParseSessionCookie_Expired(t *testing.T) {
	user := &User{Username: "alice"}
	// TTL of -1 second = already expired
	cookie := CreateSessionCookie(user, NewSessionID(), "", "secret", -1*time.Second, false)

	req := httptest.NewRequest("GET", "/", nil)
	req.AddCookie(cookie)

	parsed := ParseSessionCookie(req, "secret")
	if parsed != nil {
		t.Error("ParseSessionCookie should return nil for expired cookie")
	}
}

func TestParseSessionCookie_NoCookie(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	parsed := ParseSessionCookie(req, "secret")
	if parsed != nil {
		t.Error("ParseSessionCookie should return nil when no cookie present")
	}
}

func TestParseSessionCookie_TamperedPayload(t *testing.T) {
	user := &User{Username: "alice"}
	cookie := CreateSessionCookie(user, NewSessionID(), "", "secret", 1*time.Hour, false)

	// Tamper with the payload (change first char)
	val := cookie.Value
	if val[0] == 'a' {
		cookie.Value = "b" + val[1:]
	} else {
		cookie.Value = "a" + val[1:]
	}

	req := httptest.NewRequest("GET", "/", nil)
	req.AddCookie(cookie)

	parsed := ParseSessionCookie(req, "secret")
	if parsed != nil {
		t.Error("ParseSessionCookie should return nil for tampered cookie")
	}
}

func TestParseSessionCookie_MalformedValue(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.AddCookie(&http.Cookie{Name: DefaultCookieName, Value: "not-a-valid-cookie"})

	parsed := ParseSessionCookie(req, "secret")
	if parsed != nil {
		t.Error("ParseSessionCookie should return nil for malformed cookie (no dot)")
	}
}

func TestCreateSessionCookie_Secure(t *testing.T) {
	user := &User{Username: "alice"}
	cookie := CreateSessionCookie(user, NewSessionID(), "", "secret", 1*time.Hour, true)
	if !cookie.Secure {
		t.Error("cookie should be Secure when secure=true")
	}
}

func TestCreateSessionCookie_NoGroups(t *testing.T) {
	user := &User{Username: "bob"}
	cookie := CreateSessionCookie(user, NewSessionID(), "", "secret", 1*time.Hour, false)

	req := httptest.NewRequest("GET", "/", nil)
	req.AddCookie(cookie)

	parsed := ParseSessionCookie(req, "secret")
	if parsed == nil {
		t.Fatal("ParseSessionCookie returned nil")
	}
	if parsed.User.Username != "bob" {
		t.Errorf("username = %q, want %q", parsed.User.Username, "bob")
	}
	if len(parsed.User.Groups) != 0 {
		t.Errorf("groups = %v, want empty", parsed.User.Groups)
	}
}

func TestClearSessionCookie(t *testing.T) {
	cookie := ClearSessionCookie()
	if cookie.Name != DefaultCookieName {
		t.Errorf("cookie name = %q, want %q", cookie.Name, DefaultCookieName)
	}
	if cookie.MaxAge != -1 {
		t.Errorf("cookie MaxAge = %d, want -1", cookie.MaxAge)
	}
}

func TestSignData_Deterministic(t *testing.T) {
	sig1 := signData("hello", "secret")
	sig2 := signData("hello", "secret")
	if sig1 != sig2 {
		t.Error("signData should be deterministic")
	}
}

func TestSignData_DifferentInputs(t *testing.T) {
	sig1 := signData("hello", "secret")
	sig2 := signData("world", "secret")
	if sig1 == sig2 {
		t.Error("signData should produce different signatures for different inputs")
	}
}

func TestSignData_DifferentSecrets(t *testing.T) {
	sig1 := signData("hello", "secret1")
	sig2 := signData("hello", "secret2")
	if sig1 == sig2 {
		t.Error("signData should produce different signatures for different secrets")
	}
}

func TestCreateSessionCookie_WithIDToken(t *testing.T) {
	secret := "test-secret"
	user := &User{Username: "alice", Groups: []string{"devs"}}
	sid := NewSessionID()
	idToken := "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.test-payload.test-sig"

	cookie := CreateSessionCookie(user, sid, idToken, secret, 1*time.Hour, false)

	req := httptest.NewRequest("GET", "/", nil)
	req.AddCookie(cookie)

	parsed := ParseSessionCookie(req, secret)
	if parsed == nil {
		t.Fatal("ParseSessionCookie returned nil for cookie with ID token")
	}
	if parsed.User.Username != "alice" {
		t.Errorf("username = %q, want %q", parsed.User.Username, "alice")
	}
	if parsed.IDToken != idToken {
		t.Errorf("IDToken = %q, want %q", parsed.IDToken, idToken)
	}
	if parsed.SID != sid {
		t.Errorf("SID = %q, want %q", parsed.SID, sid)
	}
}

func TestSessionIDToken_NoIDToken(t *testing.T) {
	secret := "test-secret"
	user := &User{Username: "alice"}

	cookie := CreateSessionCookie(user, NewSessionID(), "", secret, 1*time.Hour, false)
	req := httptest.NewRequest("GET", "/", nil)
	req.AddCookie(cookie)

	parsed := ParseSessionCookie(req, secret)
	if parsed == nil {
		t.Fatal("ParseSessionCookie returned nil")
	}
	if parsed.IDToken != "" {
		t.Errorf("IDToken = %q, want empty string", parsed.IDToken)
	}
}

func TestCreateSessionCookie_WithSID(t *testing.T) {
	secret := "test-secret"
	user := &User{Username: "alice"}
	sid := "abcdef0123456789abcdef0123456789"

	cookie := CreateSessionCookie(user, sid, "", secret, 1*time.Hour, false)

	req := httptest.NewRequest("GET", "/", nil)
	req.AddCookie(cookie)

	parsed := ParseSessionCookie(req, secret)
	if parsed == nil {
		t.Fatal("ParseSessionCookie returned nil")
	}
	if parsed.SID != sid {
		t.Errorf("SID = %q, want %q", parsed.SID, sid)
	}
}

func TestParseSessionCookie_LegacyCookieWithoutSID(t *testing.T) {
	// Simulate a pre-upgrade cookie that doesn't have the "s" field
	secret := "test-secret"
	payload := struct {
		Username  string   `json:"u"`
		Groups    []string `json:"g,omitempty"`
		ExpiresAt int64    `json:"e"`
		IDToken   string   `json:"t,omitempty"`
	}{
		Username:  "alice",
		Groups:    []string{"devs"},
		ExpiresAt: time.Now().Add(1 * time.Hour).Unix(),
	}

	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	encoded := base64.RawURLEncoding.EncodeToString(data)
	sig := signData(encoded, secret)

	req := httptest.NewRequest("GET", "/", nil)
	req.AddCookie(&http.Cookie{
		Name:  DefaultCookieName,
		Value: encoded + "." + sig,
	})

	parsed := ParseSessionCookie(req, secret)
	if parsed == nil {
		t.Fatal("ParseSessionCookie should handle legacy cookies without sid")
	}
	if parsed.User.Username != "alice" {
		t.Errorf("username = %q, want %q", parsed.User.Username, "alice")
	}
	if parsed.SID != "" {
		t.Errorf("SID = %q, want empty string for legacy cookie", parsed.SID)
	}
}

func TestNewSessionID_Unique(t *testing.T) {
	id1 := NewSessionID()
	id2 := NewSessionID()

	if id1 == id2 {
		t.Error("NewSessionID should produce unique values")
	}
	if len(id1) != 32 {
		t.Errorf("NewSessionID length = %d, want 32 (16 bytes hex)", len(id1))
	}
	if len(id2) != 32 {
		t.Errorf("NewSessionID length = %d, want 32 (16 bytes hex)", len(id2))
	}
}

func TestParseSessionCookie_ExpiresAt(t *testing.T) {
	secret := "test-secret"
	user := &User{Username: "alice"}
	ttl := 2 * time.Hour

	cookie := CreateSessionCookie(user, NewSessionID(), "", secret, ttl, false)

	req := httptest.NewRequest("GET", "/", nil)
	req.AddCookie(cookie)

	parsed := ParseSessionCookie(req, secret)
	if parsed == nil {
		t.Fatal("ParseSessionCookie returned nil")
	}

	// ExpiresAt should be approximately now + ttl (within a few seconds)
	expected := time.Now().Add(ttl)
	diff := parsed.ExpiresAt.Sub(expected)
	if diff < -5*time.Second || diff > 5*time.Second {
		t.Errorf("ExpiresAt off by %v, want within 5s of now+2h", diff)
	}
}

func TestCreateSessionCookie_DropsIDTokenWhenTooLarge(t *testing.T) {
	secret := "test-secret"
	// Build a cookie that's over 3800 bytes with the ID token, but under without it.
	// 40 groups × ~25 chars ≈ 1000 bytes of groups. 2KB ID token. Together > 3800 after base64.
	groups := make([]string, 40)
	for i := range groups {
		groups[i] = "org:engineering:team-" + strings.Repeat("x", 10)
	}
	user := &User{Username: "alice@example.com", Groups: groups}
	sid := NewSessionID()
	largeIDToken := strings.Repeat("x", 2000)

	// First verify the cookie WITHOUT ID token fits
	smallCookie := CreateSessionCookie(user, sid, "", secret, 1*time.Hour, false)
	if len(smallCookie.Value) > maxCookieSize {
		t.Skipf("groups alone exceed %d bytes (%d) — can't test ID token drop", maxCookieSize, len(smallCookie.Value))
	}

	// Now create with the large ID token — should trigger the drop
	cookie := CreateSessionCookie(user, sid, largeIDToken, secret, 1*time.Hour, false)

	// Parse and verify the cookie is still valid
	req := httptest.NewRequest("GET", "/", nil)
	req.AddCookie(cookie)
	parsed := ParseSessionCookie(req, secret)
	if parsed == nil {
		t.Fatal("ParseSessionCookie returned nil for size-capped cookie")
	}
	if parsed.User.Username != "alice@example.com" {
		t.Errorf("username = %q, want alice@example.com", parsed.User.Username)
	}
	if parsed.SID != sid {
		t.Errorf("SID lost after size cap")
	}
	if parsed.IDToken == largeIDToken {
		t.Error("ID token should have been dropped to fit cookie size limit")
	}
	if len(cookie.Value) > maxCookieSize {
		t.Errorf("cookie still %d bytes after dropping ID token (limit %d)", len(cookie.Value), maxCookieSize)
	}
}
