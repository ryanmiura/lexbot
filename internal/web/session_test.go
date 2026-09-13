package web

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func issuedCookie(t *testing.T, m *SessionManager, userID int64) *http.Cookie {
	t.Helper()
	rec := httptest.NewRecorder()
	m.Issue(rec, userID)
	cookies := rec.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("expected exactly one cookie, got %d", len(cookies))
	}
	return cookies[0]
}

func requestWithCookie(c *http.Cookie) *http.Request {
	r := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
	if c != nil {
		r.AddCookie(c)
	}
	return r
}

func TestSessionIssueAndValidate(t *testing.T) {
	m := NewSessionManager("test-secret")
	c := issuedCookie(t, m, 42)

	userID, err := m.UserID(requestWithCookie(c))
	if err != nil {
		t.Fatalf("UserID failed: %v", err)
	}
	if userID != 42 {
		t.Errorf("got userID=%d, want 42", userID)
	}
}

func TestSessionNoCookie(t *testing.T) {
	m := NewSessionManager("test-secret")
	if _, err := m.UserID(requestWithCookie(nil)); !errors.Is(err, ErrInvalidSession) {
		t.Errorf("expected ErrInvalidSession, got %v", err)
	}
}

func TestSessionTamperedValue(t *testing.T) {
	m := NewSessionManager("test-secret")
	c := issuedCookie(t, m, 42)
	c.Value = c.Value[:len(c.Value)-1] + "x" // flip the last char of the signature

	if _, err := m.UserID(requestWithCookie(c)); !errors.Is(err, ErrInvalidSession) {
		t.Errorf("expected ErrInvalidSession for a tampered cookie, got %v", err)
	}
}

func TestSessionWrongSecret(t *testing.T) {
	issuer := NewSessionManager("secret-a")
	verifier := NewSessionManager("secret-b")
	c := issuedCookie(t, issuer, 42)

	if _, err := verifier.UserID(requestWithCookie(c)); !errors.Is(err, ErrInvalidSession) {
		t.Errorf("expected ErrInvalidSession when secrets differ, got %v", err)
	}
}

func TestSessionMalformedValue(t *testing.T) {
	m := NewSessionManager("test-secret")
	c := &http.Cookie{Name: sessionCookieName, Value: "not-a-valid-session-value"}

	if _, err := m.UserID(requestWithCookie(c)); !errors.Is(err, ErrInvalidSession) {
		t.Errorf("expected ErrInvalidSession for a malformed cookie, got %v", err)
	}
}

func TestSessionClear(t *testing.T) {
	m := NewSessionManager("test-secret")
	rec := httptest.NewRecorder()
	m.Clear(rec)

	cookies := rec.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("expected exactly one cookie, got %d", len(cookies))
	}
	if cookies[0].MaxAge >= 0 {
		t.Errorf("expected a negative MaxAge to delete the cookie, got %d", cookies[0].MaxAge)
	}
}
