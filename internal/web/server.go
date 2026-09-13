package web

import (
	"embed"
	"errors"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"strconv"

	"lexbot/internal/service"
)

//go:embed templates/*.html
var templateFS embed.FS

var templates = template.Must(template.ParseFS(templateFS, "templates/*.html"))

// Server implements http.Handler for the web dashboard: it authenticates
// visitors via SessionManager and reads/writes through the same
// repositories the WhatsApp bot uses, so both surfaces stay consistent
// automatically.
type Server struct {
	mux      *http.ServeMux
	sessions *SessionManager
	tokens   service.DashboardTokenRepository
	words    service.WordRepository
	quiz     service.QuizRepository
	users    service.UserRepository
}

func NewServer(sessions *SessionManager, tokens service.DashboardTokenRepository, words service.WordRepository, quiz service.QuizRepository, users service.UserRepository) *Server {
	s := &Server{
		mux:      http.NewServeMux(),
		sessions: sessions,
		tokens:   tokens,
		words:    words,
		quiz:     quiz,
		users:    users,
	}
	s.routes()
	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *Server) routes() {
	s.mux.HandleFunc("GET /d/{token}", s.handleMagicLink)
	s.mux.HandleFunc("GET /dashboard", s.handleDashboard)
	s.mux.HandleFunc("POST /dashboard/words/{id}/delete", s.handleDeleteWord)
	s.mux.HandleFunc("POST /dashboard/preferences", s.handlePreferences)
	s.mux.HandleFunc("GET /logout", s.handleLogout)
}

// handleMagicLink consumes a one-time token and, if valid, starts a
// longer-lived session and redirects to the clean /dashboard URL — the
// token itself never lingers in the address bar past this one request.
func (s *Server) handleMagicLink(w http.ResponseWriter, r *http.Request) {
	token := r.PathValue("token")

	userID, err := s.tokens.ConsumeToken(r.Context(), token)
	if errors.Is(err, service.ErrInvalidToken) {
		s.renderLoginRequired(w, "Esse link é inválido, já foi usado ou expirou. Peça um novo com /dashboard no WhatsApp.")
		return
	}
	if err != nil {
		slog.Error("failed to consume dashboard token", "error", err)
		s.renderLoginRequired(w, "Ocorreu um erro. Tente novamente em alguns instantes.")
		return
	}

	s.sessions.Issue(w, userID)
	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}

// handleDashboard renders the main page: progress stats, a searchable word
// list, and the quiz-hints preference toggle.
func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	userID, err := s.sessions.UserID(r)
	if err != nil {
		s.renderLoginRequired(w, "Sua sessão expirou. Peça um novo link com /dashboard no WhatsApp.")
		return
	}

	query := r.URL.Query().Get("q")
	words, err := s.words.SearchByUser(r.Context(), userID, query)
	if err != nil {
		slog.Error("failed to search words", "user_id", userID, "error", err)
		http.Error(w, "Erro ao carregar suas palavras.", http.StatusInternalServerError)
		return
	}

	stats, err := s.words.GetStats(r.Context(), userID)
	if err != nil {
		slog.Error("failed to get word stats", "user_id", userID, "error", err)
		http.Error(w, "Erro ao carregar suas estatísticas.", http.StatusInternalServerError)
		return
	}

	quizzesCompleted, lastQuizAt, err := s.quiz.GetCompletedStats(r.Context(), userID)
	if err != nil {
		slog.Error("failed to get quiz stats", "user_id", userID, "error", err)
		http.Error(w, "Erro ao carregar suas estatísticas.", http.StatusInternalServerError)
		return
	}

	user, err := s.users.FindUserByID(r.Context(), userID)
	if err != nil || user == nil {
		slog.Error("failed to load user for dashboard", "user_id", userID, "error", err)
		http.Error(w, "Erro ao carregar seu perfil.", http.StatusInternalServerError)
		return
	}

	data := dashboardPageData{
		Stats:            stats,
		QuizzesCompleted: quizzesCompleted,
		Words:            words,
		Query:            query,
		HintsEnabled:     user.QuizHintsEnabled,
	}
	if stats.TimesReviewed > 0 {
		data.Accuracy = fmt.Sprintf("%.0f%%", float64(stats.TimesCorrect)/float64(stats.TimesReviewed)*100)
	}
	if stats.LastAddedAt != nil {
		data.LastWordAt = stats.LastAddedAt.Format("02/01/2006")
	}
	if lastQuizAt != nil {
		data.LastQuizAt = lastQuizAt.Format("02/01/2006")
	}

	s.render(w, "dashboard.html", data)
}

// handleDeleteWord deletes one word. Scoping the delete by the session's
// own userID (never a value from the request body) is what prevents a
// user from deleting another user's word even if they tamper with the id
// in the URL.
func (s *Server) handleDeleteWord(w http.ResponseWriter, r *http.Request) {
	userID, err := s.sessions.UserID(r)
	if err != nil {
		s.renderLoginRequired(w, "Sua sessão expirou. Peça um novo link com /dashboard no WhatsApp.")
		return
	}

	wordID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "ID de palavra inválido.", http.StatusBadRequest)
		return
	}

	deleted, err := s.words.Delete(r.Context(), userID, wordID)
	if err != nil {
		slog.Error("failed to delete word", "user_id", userID, "word_id", wordID, "error", err)
		http.Error(w, "Erro ao excluir a palavra.", http.StatusInternalServerError)
		return
	}
	if deleted {
		slog.Info("word deleted via dashboard", "user_id", userID, "word_id", wordID)
	} else {
		slog.Warn("delete attempted on a word not owned by this user (or nonexistent)",
			"user_id", userID, "word_id", wordID)
	}

	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}

// handlePreferences toggles the quiz-hints preference.
func (s *Server) handlePreferences(w http.ResponseWriter, r *http.Request) {
	userID, err := s.sessions.UserID(r)
	if err != nil {
		s.renderLoginRequired(w, "Sua sessão expirou. Peça um novo link com /dashboard no WhatsApp.")
		return
	}

	hintsEnabled := r.FormValue("hints_enabled") == "on"
	if err := s.users.UpdatePreferences(r.Context(), userID, hintsEnabled); err != nil {
		slog.Error("failed to update preferences", "user_id", userID, "error", err)
		http.Error(w, "Erro ao salvar preferências.", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	s.sessions.Clear(w)
	s.renderLoginRequired(w, "Você saiu. Peça um novo link com /dashboard no WhatsApp quando quiser voltar.")
}

type dashboardPageData struct {
	Stats            *service.WordStats
	Accuracy         string // "" if there's no review data yet
	QuizzesCompleted int
	LastQuizAt       string // "" if the user has never completed a quiz
	LastWordAt       string // "" if the user has no words yet
	Words            []*service.Word
	Query            string
	HintsEnabled     bool
}

func (s *Server) render(w http.ResponseWriter, name string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := templates.ExecuteTemplate(w, name, data); err != nil {
		slog.Error("failed to render template", "template", name, "error", err)
	}
}

func (s *Server) renderLoginRequired(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := templates.ExecuteTemplate(w, "login_required.html", message); err != nil {
		slog.Error("failed to render login_required template", "error", err)
	}
}
