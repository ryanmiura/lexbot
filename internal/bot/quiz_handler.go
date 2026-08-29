package bot

import (
	"context"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"sync"
	"time"

	"lexbot/internal/service"
)

// maxQuizRounds caps how many words a single quiz session can cover,
// mirroring the QUIZ_MAX_WORDS_PER_SESSION default described in the docs.
const maxQuizRounds = 20

// quizQuestionTypes defines the rotation used to vary question formats
// across a session's rounds.
var quizQuestionTypes = []string{"multiple_choice", "complete_sentence", "reverse"}

// activeQuestion holds the ephemeral state of the question currently shown
// to the user. It isn't persisted: multiple-choice option order and the
// exact correct-answer string only matter for the single turn between
// asking and answering.
type activeQuestion struct {
	sessionID     int64
	wordID        int64
	word          string
	questionType  string
	correctAnswer string
	options       []string // populated for multiple_choice only
}

// quizRuntime tracks a single user's in-progress quiz interaction: either
// waiting for them to pick a round count (CONFIGURING) or waiting for an
// answer to the current question (ACTIVE).
type quizRuntime struct {
	pendingConfig  bool
	available      []*service.Word // candidate words, sorted by priority, while CONFIGURING
	maxRounds      int
	session        *service.QuizSession
	current        *activeQuestion
	correctWords   []string
	incorrectWords []string
}

// QuizHandler drives the /quiz state machine: CONFIGURING -> ACTIVE -> COMPLETED.
type QuizHandler struct {
	messenger service.Messenger
	words     service.WordRepository
	quizRepo  service.QuizRepository
	quizSvc   *service.QuizService

	mu      sync.Mutex
	runtime map[int64]*quizRuntime // keyed by user ID
}

func NewQuizHandler(messenger service.Messenger, words service.WordRepository, quizRepo service.QuizRepository, quizSvc *service.QuizService) *QuizHandler {
	return &QuizHandler{
		messenger: messenger,
		words:     words,
		quizRepo:  quizRepo,
		quizSvc:   quizSvc,
		runtime:   make(map[int64]*quizRuntime),
	}
}

// HasPendingInteraction reports whether the user has a quiz awaiting either
// round-count configuration or an answer, without touching the database.
func (h *QuizHandler) HasPendingInteraction(userID int64) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	_, ok := h.runtime[userID]
	return ok
}

// Start handles the "/quiz" command: resumes an in-progress quiz, or begins
// a new one by asking how many words to practice.
func (h *QuizHandler) Start(ctx context.Context, chat string, user *service.User) {
	if h.HasPendingInteraction(user.ID) {
		h.messenger.Send(chat, "Você já tem um quiz em andamento! Responda a pergunta atual para continuar.")
		return
	}

	active, err := h.quizRepo.GetActiveSession(ctx, user.ID)
	if err != nil {
		fmt.Printf("Error checking active quiz session: %v\n", err)
		h.messenger.Send(chat, "❌ Ocorreu um erro ao verificar seu quiz. Tente novamente.")
		return
	}
	if active != nil {
		rt := &quizRuntime{session: active}
		h.setRuntime(user.ID, rt)
		h.messenger.Send(chat, "Você já tem um quiz em andamento! Vamos continuar de onde parou.")
		h.askQuestion(ctx, chat, user, rt, active.CurrentIndex)
		return
	}

	available, err := h.quizSvc.SelectWordsForQuiz(ctx, user.ID, 0)
	if err != nil {
		fmt.Printf("Error selecting quiz words: %v\n", err)
		h.messenger.Send(chat, "❌ Ocorreu um erro ao preparar o quiz. Tente novamente.")
		return
	}
	if len(available) == 0 {
		h.messenger.Send(chat, "📋 Você ainda não tem nenhuma palavra salva. Envie uma palavra em inglês para começar antes de fazer o quiz!")
		return
	}

	maxRounds := min(len(available), maxQuizRounds)
	rt := &quizRuntime{pendingConfig: true, available: available, maxRounds: maxRounds}
	h.setRuntime(user.ID, rt)
	h.messenger.Send(chat, fmt.Sprintf("🎯 Vamos revisar! Quantas palavras você quer praticar? (1 a %d)", maxRounds))
}

// HandleActiveInput consumes msg as part of an in-progress quiz interaction
// (round-count configuration or an answer) if the user has one, resuming
// from persisted state if the process restarted mid-quiz. It returns false
// when there's nothing to handle, so the caller can fall back to normal
// word/command processing.
func (h *QuizHandler) HandleActiveInput(ctx context.Context, chat string, user *service.User, msg string) bool {
	h.mu.Lock()
	rt, ok := h.runtime[user.ID]
	h.mu.Unlock()

	if ok && rt.pendingConfig {
		h.handleConfigReply(ctx, chat, user, rt, msg)
		return true
	}
	if ok && rt.session != nil {
		h.handleAnswer(ctx, chat, user, rt, msg)
		return true
	}

	session, err := h.quizRepo.GetActiveSession(ctx, user.ID)
	if err != nil {
		fmt.Printf("Error checking active quiz session: %v\n", err)
		return false
	}
	if session == nil {
		return false
	}

	// The process must have restarted mid-quiz: rebuild in-memory state and
	// re-show the current question rather than treating msg as an answer to
	// a question the user never actually saw.
	rt = &quizRuntime{session: session}
	h.setRuntime(user.ID, rt)
	h.askQuestion(ctx, chat, user, rt, session.CurrentIndex)
	return true
}

func (h *QuizHandler) handleConfigReply(ctx context.Context, chat string, user *service.User, rt *quizRuntime, msg string) {
	n, err := strconv.Atoi(strings.TrimSpace(msg))
	if err != nil || n < 1 {
		h.messenger.Send(chat, fmt.Sprintf("Por favor, digite um número válido de palavras (1 a %d).", rt.maxRounds))
		return
	}
	if n > rt.maxRounds {
		n = rt.maxRounds
	}

	wordIDs := make([]int64, n)
	for i := 0; i < n; i++ {
		wordIDs[i] = rt.available[i].ID
	}

	session := &service.QuizSession{UserID: user.ID, WordIDs: wordIDs, TotalQuestions: n}
	if err := h.quizRepo.SaveSession(ctx, session); err != nil {
		fmt.Printf("Error saving quiz session: %v\n", err)
		h.messenger.Send(chat, "❌ Ocorreu um erro ao iniciar o quiz. Tente novamente.")
		h.clearRuntime(user.ID)
		return
	}

	rt.pendingConfig = false
	rt.available = nil
	rt.session = session
	h.messenger.Send(chat, fmt.Sprintf("🎯 Vamos lá! %d perguntas.", n))
	h.askQuestion(ctx, chat, user, rt, 0)
}

func (h *QuizHandler) handleAnswer(ctx context.Context, chat string, user *service.User, rt *quizRuntime, msg string) {
	cur := rt.current
	if cur == nil {
		h.askQuestion(ctx, chat, user, rt, rt.session.CurrentIndex)
		return
	}

	userAnswer := strings.TrimSpace(msg)
	if cur.questionType == "multiple_choice" {
		idx, err := strconv.Atoi(userAnswer)
		if err != nil || idx < 1 || idx > len(cur.options) {
			h.messenger.Send(chat, fmt.Sprintf("Por favor, responda com um número de 1 a %d.", len(cur.options)))
			return
		}
		userAnswer = cur.options[idx-1]
	}

	correct, err := h.quizSvc.RecordAnswer(ctx, cur.sessionID, cur.wordID, cur.questionType, userAnswer, cur.correctAnswer)
	if err != nil {
		fmt.Printf("Error recording quiz answer: %v\n", err)
		h.messenger.Send(chat, "❌ Ocorreu um erro ao registrar sua resposta. Tente novamente.")
		return
	}

	if correct {
		rt.correctWords = append(rt.correctWords, cur.word)
		rt.session.CorrectCount++
	} else {
		rt.incorrectWords = append(rt.incorrectWords, cur.word)
	}
	rt.session.CurrentIndex++

	h.sendAnswerFeedback(chat, cur, userAnswer, correct)

	if rt.session.CurrentIndex >= rt.session.TotalQuestions {
		now := time.Now()
		rt.session.Status = "completed"
		rt.session.CompletedAt = &now
		if err := h.quizRepo.UpdateSession(ctx, rt.session); err != nil {
			fmt.Printf("Error completing quiz session: %v\n", err)
		}
		h.sendFinalResult(chat, rt)
		h.clearRuntime(user.ID)
		return
	}

	if err := h.quizRepo.UpdateSession(ctx, rt.session); err != nil {
		fmt.Printf("Error updating quiz session: %v\n", err)
	}
	h.askQuestion(ctx, chat, user, rt, rt.session.CurrentIndex)
}

func (h *QuizHandler) askQuestion(ctx context.Context, chat string, user *service.User, rt *quizRuntime, index int) {
	wordID := rt.session.WordIDs[index]
	word, err := h.words.FindByID(ctx, wordID)
	if err != nil || word == nil {
		fmt.Printf("Error loading quiz word id=%d: %v\n", wordID, err)
		h.messenger.Send(chat, "❌ Ocorreu um erro ao carregar a próxima pergunta. Tente novamente com /quiz.")
		h.clearRuntime(user.ID)
		return
	}

	questionType := quizQuestionTypes[index%len(quizQuestionTypes)]
	q := questionFor(word, questionType)

	var options []string
	if questionType == "multiple_choice" {
		options = append([]string{q.Correct}, q.Distractors...)
		rand.Shuffle(len(options), func(i, j int) { options[i], options[j] = options[j], options[i] })
	}

	rt.current = &activeQuestion{
		sessionID: rt.session.ID, wordID: wordID, word: word.Word,
		questionType: questionType, correctAnswer: q.Correct, options: options,
	}

	showHint := user.QuizHintsEnabled && questionType != "multiple_choice"
	h.messenger.Send(chat, formatQuizQuestion(index+1, rt.session.TotalQuestions, questionType, q, options, showHint, word.QuizTip))
}

func (h *QuizHandler) sendAnswerFeedback(chat string, cur *activeQuestion, userAnswer string, correct bool) {
	h.messenger.Send(chat, formatAnswerFeedback(correct, userAnswer, cur))
}

func (h *QuizHandler) sendFinalResult(chat string, rt *quizRuntime) {
	h.messenger.Send(chat, formatFinalResult(rt.session, rt.correctWords, rt.incorrectWords))
}

func (h *QuizHandler) setRuntime(userID int64, rt *quizRuntime) {
	h.mu.Lock()
	h.runtime[userID] = rt
	h.mu.Unlock()
}

func (h *QuizHandler) clearRuntime(userID int64) {
	h.mu.Lock()
	delete(h.runtime, userID)
	h.mu.Unlock()
}

// questionFor picks the QuizQuestion matching questionType from a word's
// pre-generated quiz set.
func questionFor(word *service.Word, questionType string) service.QuizQuestion {
	switch questionType {
	case "complete_sentence":
		return word.Quiz.CompleteSentence
	case "reverse":
		return word.Quiz.Reverse
	default:
		return word.Quiz.MultipleChoice
	}
}
