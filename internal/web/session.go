package web

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const sessionCookieName = "lexbot_session"

// sessionTTL is how long a session cookie lasts once issued (after the
// magic link is consumed), independent of the much shorter magic-link
// token TTL — the link only needs to bootstrap this longer session.
const sessionTTL = 7 * 24 * time.Hour

// ErrInvalidSession covers every way a session cookie can fail to
// validate: missing, malformed, tampered with, or expired. Deliberately
// collapsed into one error — callers only need to know "not logged in".
var ErrInvalidSession = errors.New("invalid or expired session")

// SessionManager issues and validates signed session cookies binding a
// browser to a user ID, with no server-side session storage: the cookie
// itself carries the user ID and expiry, and an HMAC signature is what
// makes it unforgeable without knowing secret.
type SessionManager struct {
	secret []byte
}

func NewSessionManager(secret string) *SessionManager {
	return &SessionManager{secret: []byte(secret)}
}

// Issue sets a session cookie for userID on the response.
func (m *SessionManager) Issue(w http.ResponseWriter, userID int64) {
	expires := time.Now().Add(sessionTTL)
	payload := fmt.Sprintf("%d.%d", userID, expires.Unix())
	value := payload + "." + m.sign(payload)

	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    value,
		Path:     "/",
		Expires:  expires,
		HttpOnly: true,
		Secure:   true, // this dashboard is only ever served over HTTPS (via NPM)
		SameSite: http.SameSiteStrictMode,
	})
}

// Clear removes the session cookie (used for logout).
func (m *SessionManager) Clear(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	})
}

// UserID extracts and validates the session cookie from the request.
func (m *SessionManager) UserID(r *http.Request) (int64, error) {
	c, err := r.Cookie(sessionCookieName)
	if err != nil {
		return 0, ErrInvalidSession
	}

	parts := strings.SplitN(c.Value, ".", 3)
	if len(parts) != 3 {
		return 0, ErrInvalidSession
	}
	payload, sig := parts[0]+"."+parts[1], parts[2]
	if !hmac.Equal([]byte(sig), []byte(m.sign(payload))) {
		return 0, ErrInvalidSession
	}

	userID, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return 0, ErrInvalidSession
	}
	expiresUnix, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil || time.Now().Unix() > expiresUnix {
		return 0, ErrInvalidSession
	}

	return userID, nil
}

func (m *SessionManager) sign(payload string) string {
	mac := hmac.New(sha256.New, m.secret)
	mac.Write([]byte(payload))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
