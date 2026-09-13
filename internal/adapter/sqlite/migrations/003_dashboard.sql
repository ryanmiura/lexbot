-- Magic-link tokens for the web dashboard. /dashboard on WhatsApp creates a
-- row here; the dashboard's /d/{token} endpoint consumes it exactly once.
CREATE TABLE IF NOT EXISTS dashboard_tokens (
    token      TEXT     PRIMARY KEY,
    user_id    INTEGER  NOT NULL REFERENCES users(id),
    expires_at DATETIME NOT NULL,
    used_at    DATETIME,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_dashboard_tokens_user ON dashboard_tokens(user_id);
