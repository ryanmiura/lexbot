-- Usuário identificado pelo número do WhatsApp
CREATE TABLE IF NOT EXISTS users (
    id                  INTEGER  PRIMARY KEY AUTOINCREMENT,
    phone               TEXT     NOT NULL UNIQUE,
    quiz_hints_enabled  INTEGER  DEFAULT 1,
    created_at          DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Palavra inserida pelo usuário com todos os dados retornados pela IA
CREATE TABLE IF NOT EXISTS words (
    id                  INTEGER  PRIMARY KEY AUTOINCREMENT,
    user_id             INTEGER  NOT NULL REFERENCES users(id),
    word                TEXT     NOT NULL,
    translation         TEXT     NOT NULL,
    grammar_class       TEXT     NOT NULL,
    phonetic            TEXT,
    definition_en       TEXT,
    example_en          TEXT     NOT NULL,
    example_pt          TEXT     NOT NULL,
    synonyms            TEXT,
    quiz_tip            TEXT,
    quiz_error_explain  TEXT,
    connected_speech    TEXT,
    difficulty          TEXT     DEFAULT 'new',
    times_reviewed      INTEGER  DEFAULT 0,
    times_correct        INTEGER  DEFAULT 0,
    last_reviewed_at    DATETIME,
    created_at          DATETIME DEFAULT CURRENT_TIMESTAMP,

    UNIQUE(user_id, word)
);

-- Perguntas de quiz geradas no momento do cadastro da palavra
CREATE TABLE IF NOT EXISTS quiz_questions (
    id             INTEGER  PRIMARY KEY AUTOINCREMENT,
    word_id        INTEGER  NOT NULL REFERENCES words(id),
    question_type  TEXT     NOT NULL,  -- multiple_choice | complete_sentence | reverse
    question_text  TEXT     NOT NULL,
    correct_answer TEXT     NOT NULL,
    distractors    TEXT,               -- JSON: ["option2","option3","option4"]
    created_at     DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_words_user_id       ON words(user_id);
CREATE INDEX IF NOT EXISTS idx_words_difficulty    ON words(user_id, difficulty);
CREATE INDEX IF NOT EXISTS idx_words_last_reviewed ON words(user_id, last_reviewed_at);
CREATE INDEX IF NOT EXISTS idx_quiz_questions_word ON quiz_questions(word_id);
