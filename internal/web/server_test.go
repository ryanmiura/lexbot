package web_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"lexbot/internal/adapter/sqlite"
	"lexbot/internal/service"
	"lexbot/internal/web"
)

func newTestServer(t *testing.T) (*web.Server, *sqlite.Adapter, *web.SessionManager) {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	adapter, err := sqlite.NewAdapter(dbPath)
	if err != nil {
		t.Fatalf("failed to create test adapter: %v", err)
	}
	sessions := web.NewSessionManager("test-secret")
	srv := web.NewServer(sessions, adapter, adapter, adapter, adapter, "5543936180556")
	return srv, adapter, sessions
}

func newTestWord(t *testing.T, adapter *sqlite.Adapter, userID int64, word string) *service.Word {
	t.Helper()
	w := &service.Word{
		UserID: userID, Word: word, Translation: "trad", GrammarClass: "noun",
		ExampleEN: "en", ExamplePT: "pt",
		Quiz: service.Quiz{
			MultipleChoice:   service.QuizQuestion{Question: "q1", Correct: word},
			CompleteSentence: service.QuizQuestion{Question: "q2", Correct: word},
			Reverse:          service.QuizQuestion{Question: "q3", Correct: word},
		},
	}
	if err := adapter.Save(context.Background(), w); err != nil {
		t.Fatalf("failed to save word: %v", err)
	}
	return w
}

// sessionCookie logs in as userID directly via the SessionManager, bypassing
// the magic-link flow — used by tests that aren't specifically about that
// flow itself.
func sessionCookie(sessions *web.SessionManager, userID int64) *http.Cookie {
	rec := httptest.NewRecorder()
	sessions.Issue(rec, userID)
	return rec.Result().Cookies()[0]
}

func TestDashboardRequiresSession(t *testing.T) {
	srv, _, _ := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("got status %d, want 200 (login-required page)", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "/dashboard") {
		t.Errorf("expected the login-required page to mention the /dashboard command, got: %s", rec.Body.String())
	}
}

func TestMagicLinkFlow(t *testing.T) {
	srv, adapter, _ := newTestServer(t)
	ctx := context.Background()
	user, err := adapter.Upsert(ctx, "5511999999999")
	if err != nil {
		t.Fatalf("failed to upsert user: %v", err)
	}
	newTestWord(t, adapter, user.ID, "resilient")

	token, err := adapter.CreateToken(ctx, user.ID, 15*time.Minute)
	if err != nil {
		t.Fatalf("CreateToken failed: %v", err)
	}

	// First hit: consume the token, expect a redirect + a session cookie.
	req := httptest.NewRequest(http.MethodGet, "/dashboard/"+token, nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("got status %d, want 303 redirect", rec.Code)
	}
	if loc := rec.Header().Get("Location"); loc != "/dashboard" {
		t.Errorf("got redirect to %q, want /dashboard", loc)
	}
	cookies := rec.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("expected a session cookie to be set, got %d cookies", len(cookies))
	}
	cookie := cookies[0]

	// Follow the redirect with the cookie: should see the dashboard, not the login page.
	req2 := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
	req2.AddCookie(cookie)
	rec2 := httptest.NewRecorder()
	srv.ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusOK {
		t.Fatalf("got status %d, want 200", rec2.Code)
	}
	if !strings.Contains(rec2.Body.String(), "resilient") {
		t.Errorf("expected the dashboard to list the user's word, got: %s", rec2.Body.String())
	}

	// Replaying the same magic link must fail — it's single-use.
	req3 := httptest.NewRequest(http.MethodGet, "/dashboard/"+token, nil)
	rec3 := httptest.NewRecorder()
	srv.ServeHTTP(rec3, req3)
	if rec3.Code != http.StatusOK || strings.Contains(rec3.Body.String(), "resilient") {
		t.Errorf("expected replaying the token to fail (login-required page), got status=%d body=%s", rec3.Code, rec3.Body.String())
	}
}

func TestMagicLinkInvalidToken(t *testing.T) {
	srv, _, _ := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/dashboard/this-token-does-not-exist", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("got status %d, want 200 (login-required page)", rec.Code)
	}
	if strings.Contains(rec.Body.String(), "Seu progresso") {
		t.Errorf("an invalid token must never render the dashboard")
	}
}

func TestDeleteWordScopedToOwner(t *testing.T) {
	srv, adapter, sessions := newTestServer(t)
	ctx := context.Background()
	owner, err := adapter.Upsert(ctx, "5511999999999")
	if err != nil {
		t.Fatalf("failed to upsert owner: %v", err)
	}
	other, err := adapter.Upsert(ctx, "5511888888888")
	if err != nil {
		t.Fatalf("failed to upsert other user: %v", err)
	}
	w := newTestWord(t, adapter, owner.ID, "turn")

	t.Run("another user's session cannot delete it", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/dashboard/words/"+strconv.FormatInt(w.ID, 10)+"/delete", nil)
		req.AddCookie(sessionCookie(sessions, other.ID))
		rec := httptest.NewRecorder()
		srv.ServeHTTP(rec, req)

		got, err := adapter.FindByID(ctx, w.ID)
		if err != nil {
			t.Fatalf("FindByID failed: %v", err)
		}
		if got == nil {
			t.Fatalf("expected the word to survive a delete attempt by a different user")
		}
	})

	t.Run("the owner's session deletes it", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/dashboard/words/"+strconv.FormatInt(w.ID, 10)+"/delete", nil)
		req.AddCookie(sessionCookie(sessions, owner.ID))
		rec := httptest.NewRecorder()
		srv.ServeHTTP(rec, req)

		if rec.Code != http.StatusSeeOther {
			t.Fatalf("got status %d, want 303 redirect", rec.Code)
		}
		got, err := adapter.FindByID(ctx, w.ID)
		if err != nil {
			t.Fatalf("FindByID failed: %v", err)
		}
		if got != nil {
			t.Fatalf("expected the word to be gone after the owner deletes it")
		}
	})
}

func TestPreferencesToggle(t *testing.T) {
	srv, adapter, sessions := newTestServer(t)
	ctx := context.Background()
	user, err := adapter.Upsert(ctx, "5511999999999")
	if err != nil {
		t.Fatalf("failed to upsert user: %v", err)
	}
	if !user.QuizHintsEnabled {
		t.Fatalf("expected quiz hints enabled by default")
	}

	form := strings.NewReader("") // hints_enabled omitted == unchecked checkbox
	req := httptest.NewRequest(http.MethodPost, "/dashboard/preferences", form)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(sessionCookie(sessions, user.ID))
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("got status %d, want 303 redirect", rec.Code)
	}
	got, err := adapter.FindUserByID(ctx, user.ID)
	if err != nil {
		t.Fatalf("FindUserByID failed: %v", err)
	}
	if got.QuizHintsEnabled {
		t.Errorf("expected quiz hints disabled after submitting the form without hints_enabled")
	}
}

func TestLogoutClearsSession(t *testing.T) {
	srv, adapter, sessions := newTestServer(t)
	ctx := context.Background()
	user, err := adapter.Upsert(ctx, "5511999999999")
	if err != nil {
		t.Fatalf("failed to upsert user: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/logout", nil)
	req.AddCookie(sessionCookie(sessions, user.ID))
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	cookies := rec.Result().Cookies()
	if len(cookies) != 1 || cookies[0].MaxAge >= 0 {
		t.Fatalf("expected logout to clear the session cookie, got cookies=%+v", cookies)
	}
}

func TestLandingPage(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	t.Run("with WHATSAPP_PHONE configured", func(t *testing.T) {
		srv, _, _ := newTestServer(t) // configured with "5543936180556"
		rec := httptest.NewRecorder()
		srv.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("got status %d, want 200", rec.Code)
		}
		if !strings.Contains(rec.Body.String(), "https://wa.me/5543936180556") {
			t.Errorf("expected the CTA to link to wa.me with the configured number, got: %s", rec.Body.String())
		}
	})

	t.Run("without WHATSAPP_PHONE configured", func(t *testing.T) {
		dbPath := filepath.Join(t.TempDir(), "test.db")
		adapter, err := sqlite.NewAdapter(dbPath)
		if err != nil {
			t.Fatalf("failed to create test adapter: %v", err)
		}
		srv := web.NewServer(web.NewSessionManager("test-secret"), adapter, adapter, adapter, adapter, "")

		rec := httptest.NewRecorder()
		srv.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("got status %d, want 200", rec.Code)
		}
		if strings.Contains(rec.Body.String(), "wa.me") {
			t.Errorf("expected no wa.me CTA when WHATSAPP_PHONE isn't configured, got: %s", rec.Body.String())
		}
	})
}
