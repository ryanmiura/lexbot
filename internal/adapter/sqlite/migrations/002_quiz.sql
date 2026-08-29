-- Sessão ativa de quiz por usuário
CREATE TABLE IF NOT EXISTS quiz_sessions (
    id              INTEGER  PRIMARY KEY AUTOINCREMENT,
    user_id         INTEGER  NOT NULL REFERENCES users(id),
    status          TEXT     DEFAULT 'active', -- active | completed | abandoned
    word_ids        TEXT     NOT NULL,          -- JSON: [3,7,12,18,25]
    current_index   INTEGER  DEFAULT 0,
    correct_count   INTEGER  DEFAULT 0,
    total_questions INTEGER  NOT NULL,
    started_at      DATETIME DEFAULT CURRENT_TIMESTAMP,
    completed_at    DATETIME
);

-- Registro histórico de cada resposta em cada quiz
CREATE TABLE IF NOT EXISTS quiz_answers (
    id             INTEGER  PRIMARY KEY AUTOINCREMENT,
    session_id     INTEGER  NOT NULL REFERENCES quiz_sessions(id),
    word_id        INTEGER  NOT NULL REFERENCES words(id),
    question_type  TEXT     NOT NULL,
    user_answer    TEXT     NOT NULL,
    correct_answer TEXT     NOT NULL,
    is_correct     INTEGER  NOT NULL, -- 1 = correto, 0 = incorreto
    answered_at    DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_quiz_sessions_user ON quiz_sessions(user_id, status);
CREATE INDEX IF NOT EXISTS idx_quiz_answers_word   ON quiz_answers(word_id);
