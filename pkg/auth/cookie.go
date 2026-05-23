package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

// DefaultCookieName is the default session cookie name
const DefaultCookieName = "radar_session"

// maxCookieSize is the safe limit for cookie values. RFC 6265 requires
// browsers to support at least 4096 bytes per cookie, but some proxies
// and CDNs enforce stricter limits. We use 3800 to leave headroom for
// the cookie name, attributes (Path, Secure, HttpOnly, SameSite, MaxAge).
const maxCookieSize = 3800

// Session represents a parsed session cookie.
type Session struct {
	User      *User
	SID       string    // stable session identifier (empty for pre-upgrade cookies)
	IDToken   string    // raw OIDC id_token for RP-Initiated Logout
	ExpiresAt time.Time // when the cookie expires
}

// cookiePayload is the data stored in the session cookie
type cookiePayload struct {
	Username  string   `json:"u"`
	Groups    []string `json:"g,omitempty"`
	ExpiresAt int64    `json:"e"`
	IDToken   string   `json:"t,omitempty"` // raw OIDC id_token for RP-Initiated Logout
	SID       string   `json:"s,omitempty"` // session ID for backchannel logout revocation
}

// NewSessionID generates a random 16-byte hex session ID.
func NewSessionID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic(fmt.Sprintf("[auth] Failed to generate session ID: %v", err))
	}
	return hex.EncodeToString(b)
}

// CreateSessionCookie creates a signed session cookie for the given user.
// Format: base64(json) + "." + base64(hmac-sha256).
// The sid must be non-empty — use NewSessionID() to generate one.
func CreateSessionCookie(user *User, sid, idToken, secret string, ttl time.Duration, secure bool) *http.Cookie {
	if sid == "" {
		panic(fmt.Sprintf("[auth] CreateSessionCookie called with empty sid for user %s", user.Username))
	}

	payload := cookiePayload{
		Username:  user.Username,
		Groups:    user.Groups,
		ExpiresAt: time.Now().Add(ttl).Unix(),
		IDToken:   idToken,
		SID:       sid,
	}

	value := buildCookieValue(payload, secret)

	// Browser cookie size limit is ~4096 bytes. If the payload is too large
	// (many groups + large ID token), drop the ID token first — it's only
	// needed for RP-Initiated Logout's id_token_hint and falls back to
	// client_id gracefully. Log so operators know.
	if len(value) > maxCookieSize && payload.IDToken != "" {
		log.Printf("[auth] Session cookie for %s exceeds %d bytes (%d), dropping ID token to fit",
			user.Username, maxCookieSize, len(value))
		payload.IDToken = ""
		value = buildCookieValue(payload, secret)
	}
	if len(value) > maxCookieSize {
		log.Printf("[auth] WARNING: Session cookie for %s is %d bytes (limit ~%d) — browser may silently drop it. Reduce the number of groups in the OIDC token.",
			user.Username, len(value), maxCookieSize)
	}

	return &http.Cookie{
		Name:     DefaultCookieName,
		Value:    value,
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(ttl.Seconds()),
	}
}

// ParseSessionCookie validates and parses a session cookie.
// Returns nil if the cookie is missing, invalid, or expired.
// Pre-upgrade cookies without a SID parse successfully with Session.SID == "".
func ParseSessionCookie(r *http.Request, secret string) *Session {
	cookie, err := r.Cookie(DefaultCookieName)
	if err != nil {
		return nil
	}

	parts := strings.SplitN(cookie.Value, ".", 2)
	if len(parts) != 2 {
		return nil
	}

	encoded, sig := parts[0], parts[1]

	// Verify HMAC signature
	expected := signData(encoded, secret)
	if !hmac.Equal([]byte(sig), []byte(expected)) {
		log.Printf("[auth] Session cookie HMAC verification failed — possible tampered cookie from %s", r.RemoteAddr)
		return nil
	}

	// Decode payload
	data, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return nil
	}

	var p cookiePayload
	if err := json.Unmarshal(data, &p); err != nil {
		return nil
	}

	// Check expiration
	if time.Now().Unix() > p.ExpiresAt {
		log.Printf("[auth] Session cookie expired for user %q — prompting re-auth", p.Username)
		return nil
	}

	return &Session{
		User: &User{
			Username: p.Username,
			Groups:   p.Groups,
		},
		SID:       p.SID,
		IDToken:   p.IDToken,
		ExpiresAt: time.Unix(p.ExpiresAt, 0),
	}
}

// buildCookieValue marshals the payload and signs it: base64(json) + "." + base64(hmac).
func buildCookieValue(p cookiePayload, secret string) string {
	data, err := json.Marshal(p)
	if err != nil {
		log.Fatalf("[auth] Failed to marshal session cookie payload for user %s: %v", p.Username, err)
	}
	encoded := base64.RawURLEncoding.EncodeToString(data)
	return encoded + "." + signData(encoded, secret)
}

// ClearSessionCookie returns a cookie that clears the session
func ClearSessionCookie() *http.Cookie {
	return &http.Cookie{
		Name:     DefaultCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	}
}

// signData computes HMAC-SHA256 of the given data with the secret
func signData(data, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	fmt.Fprint(mac, data)
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
